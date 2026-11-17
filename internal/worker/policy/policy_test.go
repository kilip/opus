package policy_test

import (
	"testing"

	"github.com/kilip/opus/internal/worker/domain"
	"github.com/kilip/opus/internal/worker/policy"
)

func vibeTestPolicy() domain.Policy {
	return domain.Policy{
		WorktreeRoot:      "/workspace/opus-wt",
		AllowedSubagents:  []string{"scout", "planner", "coder", "verifier", "pr-manager"},
		AllowedMCPTools:   []string{"get_task", "list_tasks", "set_status", "add_activity", "submit_report"},
		QuestionAutoReply: "decide yourself and record the assumption in the PR description",
		MaxQuestions:      1,
		AllowMerge:        true,
	}
}

func TestEvaluatePermission(t *testing.T) {
	spec := vibeTestPolicy()

	cases := []struct {
		name    string
		spec    domain.Policy
		req     policy.PermissionRequest
		allowed bool
	}{
		{
			name:    "allow file edit inside worktree (abs)",
			spec:    spec,
			req:     policy.PermissionRequest{Kind: policy.KindFileEdit, Path: "/workspace/opus-wt/internal/run/run.go"},
			allowed: true,
		},
		{
			name:    "allow file edit inside worktree (rel)",
			spec:    spec,
			req:     policy.PermissionRequest{Kind: policy.KindFileEdit, Path: "internal/run/run.go"},
			allowed: true,
		},
		{
			name:    "allow command at worktree root",
			spec:    spec,
			req:     policy.PermissionRequest{Kind: policy.KindCommand, Path: "/workspace/opus-wt", Command: "go test ./..."},
			allowed: true,
		},
		{
			name:    "allow git/gh inside worktree",
			spec:    spec,
			req:     policy.PermissionRequest{Kind: policy.KindGit, Path: "/workspace/opus-wt", Command: "gh pr merge 123"},
			allowed: true,
		},
		{
			name:    "deny file edit outside worktree (abs)",
			spec:    spec,
			req:     policy.PermissionRequest{Kind: policy.KindFileEdit, Path: "/etc/passwd"},
			allowed: false,
		},
		{
			name:    "deny file edit outside worktree (traversal)",
			spec:    spec,
			req:     policy.PermissionRequest{Kind: policy.KindFileEdit, Path: "../../secrets.txt"},
			allowed: false,
		},
		{
			name:    "deny sibling directory prefix trick",
			spec:    spec,
			req:     policy.PermissionRequest{Kind: policy.KindCommand, Path: "/workspace/opus-wt-other"},
			allowed: false,
		},
		{
			name:    "deny merge when AllowMerge false",
			spec:    domain.Policy{WorktreeRoot: "/workspace/opus-wt", AllowMerge: false},
			req:     policy.PermissionRequest{Kind: policy.KindGit, Path: "/workspace/opus-wt", Command: "gh pr merge 123"},
			allowed: false,
		},
		{
			name:    "allow configured subagent",
			spec:    spec,
			req:     policy.PermissionRequest{Kind: policy.KindInvokeSubagent, Subagent: "coder"},
			allowed: true,
		},
		{
			name:    "deny unconfigured subagent",
			spec:    spec,
			req:     policy.PermissionRequest{Kind: policy.KindInvokeSubagent, Subagent: "rogue-agent"},
			allowed: false,
		},
		{
			name:    "allow configured opus mcp tool",
			spec:    spec,
			req:     policy.PermissionRequest{Kind: policy.KindMCPTool, ToolName: "submit_report"},
			allowed: true,
		},
		{
			name:    "deny unconfigured mcp tool",
			spec:    spec,
			req:     policy.PermissionRequest{Kind: policy.KindMCPTool, ToolName: "drop_database"},
			allowed: false,
		},
		{
			name:    "deny unknown permission kind",
			spec:    spec,
			req:     policy.PermissionRequest{Kind: policy.PermissionKind("unknown")},
			allowed: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dec := policy.EvaluatePermission(tc.spec, tc.req)
			if dec.Allowed != tc.allowed {
				t.Fatalf("EvaluatePermission(%+v) = %+v, want allowed=%v", tc.req, dec, tc.allowed)
			}
		})
	}
}

func TestEnforcer_OneShotQuestionThenLimit(t *testing.T) {
	spec := vibeTestPolicy()
	enf := policy.NewEnforcer(spec)

	first := enf.HandleQuestion()
	if !first.Answered || first.LimitHit || first.Answer != spec.QuestionAutoReply {
		t.Fatalf("first question = %+v, want auto-answered with %q", first, spec.QuestionAutoReply)
	}

	second := enf.HandleQuestion()
	if second.Answered || !second.LimitHit {
		t.Fatalf("second question = %+v, want LimitHit=true", second)
	}
}
