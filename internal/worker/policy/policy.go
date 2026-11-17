// Package policy implements pure ACP permission and question enforcement against a worker.Policy spec.
package policy

import (
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/kilip/opus/internal/worker/domain"
)

// PermissionKind identifies the category of an ACP permission request.
type PermissionKind string

const (
	KindCommand        PermissionKind = "command"
	KindFileEdit       PermissionKind = "file_edit"
	KindGit            PermissionKind = "git"
	KindInvokeSubagent PermissionKind = "invoke_subagent"
	KindMCPTool        PermissionKind = "mcp_tool"
)

// PermissionRequest describes an ACP tool/operation request to evaluate.
type PermissionRequest struct {
	Kind     PermissionKind
	Path     string
	ToolName string
	Subagent string
	Command  string
}

// Decision is the result of evaluating a PermissionRequest.
type Decision struct {
	Allowed bool
	Reason  string
}

// QuestionDecision is the result of evaluating an ACP question form.
type QuestionDecision struct {
	Answered bool
	Answer   string
	LimitHit bool
	Reason   string
}

// EvaluatePermission checks whether req is allowed under spec.
func EvaluatePermission(spec domain.Policy, req PermissionRequest) Decision {
	switch req.Kind {
	case KindCommand, KindFileEdit, KindGit:
		if !isWithinWorktree(spec.WorktreeRoot, req.Path) {
			return Decision{Allowed: false, Reason: "path outside worktree root"}
		}
		if req.Kind == KindGit && !spec.AllowMerge && containsMergeCommand(req.Command) {
			return Decision{Allowed: false, Reason: "merge not allowed by policy"}
		}
		return Decision{Allowed: true}

	case KindInvokeSubagent:
		if req.Subagent != "" && slices.Contains(spec.AllowedSubagents, req.Subagent) {
			return Decision{Allowed: true}
		}
		return Decision{Allowed: false, Reason: "subagent not allowed by policy"}

	case KindMCPTool:
		if req.ToolName != "" && slices.Contains(spec.AllowedMCPTools, req.ToolName) {
			return Decision{Allowed: true}
		}
		return Decision{Allowed: false, Reason: "mcp tool not allowed by policy"}

	default:
		return Decision{Allowed: false, Reason: "unknown permission kind"}
	}
}

// EvaluateQuestion checks whether a question can be auto-answered given priorQuestions asked in the run.
func EvaluateQuestion(spec domain.Policy, priorQuestions int) QuestionDecision {
	if priorQuestions < spec.MaxQuestions {
		return QuestionDecision{
			Answered: true,
			Answer:   spec.QuestionAutoReply,
		}
	}
	return QuestionDecision{
		LimitHit: true,
		Reason:   "question limit exceeded",
	}
}

// Enforcer tracks per-run question count and evaluates permissions against a Policy spec.
type Enforcer struct {
	spec      domain.Policy
	mu        sync.Mutex
	questions int
}

// NewEnforcer creates an Enforcer for a single run.
func NewEnforcer(spec domain.Policy) *Enforcer {
	return &Enforcer{spec: spec}
}

// CheckPermission evaluates req against the enforcer's Policy spec.
func (e *Enforcer) CheckPermission(req PermissionRequest) Decision {
	return EvaluatePermission(e.spec, req)
}

// HandleQuestion evaluates an incoming question form and increments the per-run counter.
func (e *Enforcer) HandleQuestion() QuestionDecision {
	e.mu.Lock()
	defer e.mu.Unlock()
	dec := EvaluateQuestion(e.spec, e.questions)
	e.questions++
	return dec
}

func isWithinWorktree(root, target string) bool {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(target) == "" {
		return false
	}
	cleanRoot := filepath.Clean(root)
	cleanTarget := target
	if !filepath.IsAbs(cleanTarget) {
		cleanTarget = filepath.Join(cleanRoot, cleanTarget)
	} else {
		cleanTarget = filepath.Clean(cleanTarget)
	}
	rel, err := filepath.Rel(cleanRoot, cleanTarget)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func containsMergeCommand(cmd string) bool {
	fields := strings.Fields(strings.ToLower(cmd))
	for _, f := range fields {
		if f == "merge" {
			return true
		}
	}
	return false
}
