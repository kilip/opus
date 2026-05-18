package auth_test

import (
	"context"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/kilip/opus/server/internal/auth"
)

func TestCasbinPolicyEvaluation(t *testing.T) {
	ctx := context.Background()
	// Init standard enforcer in memory
	m, err := model.NewModelFromString(`
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
`)
	if err != nil {
		t.Fatalf("failed to parse casbin config model: %v", err)
	}

	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatalf("failed to init enforcer: %v", err)
	}

	policy := auth.NewCasbinPolicyManager(enforcer)

	// Assign roles under Acme workspace domain
	ok, err := policy.AssignRole(ctx, "user-bob", "ws-acme", "admin")
	if err != nil || !ok {
		t.Fatalf("failed to link grouping policy role: %v", err)
	}

	// Declare explicit resource access permission rules
	_, err = enforcer.AddPolicy("admin", "ws-acme", "/vault/files", "read")
	if err != nil {
		t.Fatalf("failed to append target permission rule: %v", err)
	}

	allowed, err := policy.Enforce("user-bob", "ws-acme", "/vault/files", "read")
	if err != nil || !allowed {
		t.Errorf("expected permission evaluation success")
	}

	forbidden, err := policy.Enforce("user-bob", "ws-acme", "/vault/files", "write")
	if err != nil || forbidden {
		t.Errorf("expected unauthorized evaluation failure")
	}
}
