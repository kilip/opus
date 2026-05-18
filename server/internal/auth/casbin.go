package auth

import (
	"context"
	"fmt"

	"github.com/casbin/casbin/v2"
)

// CasbinPolicyManager wraps the Casbin enforcer for role checks.
type CasbinPolicyManager struct {
	enforcer casbin.IEnforcer
}

// NewCasbinPolicyManager registers standard Casbin operations.
func NewCasbinPolicyManager(enforcer casbin.IEnforcer) *CasbinPolicyManager {
	return &CasbinPolicyManager{
		enforcer: enforcer,
	}
}

// Enforce checks if a subject has permissions on a resource under a workspace.
func (m *CasbinPolicyManager) Enforce(sub, dom, obj, act string) (bool, error) {
	return m.enforcer.Enforce(sub, dom, obj, act)
}

// AssignRole grants a workspace-scoped role to a user.
func (m *CasbinPolicyManager) AssignRole(ctx context.Context, user, domain, role string) (bool, error) {
	ok, err := m.enforcer.AddGroupingPolicy(user, role, domain)
	if err != nil {
		return false, fmt.Errorf("casbin: failed to assign role: %w", err)
	}
	return ok, nil
}

// RevokeRole removes a workspace-scoped role from a user.
func (m *CasbinPolicyManager) RevokeRole(ctx context.Context, user, domain, role string) (bool, error) {
	ok, err := m.enforcer.RemoveGroupingPolicy(user, role, domain)
	if err != nil {
		return false, fmt.Errorf("casbin: failed to revoke role: %w", err)
	}
	return ok, nil
}
