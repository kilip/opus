// Package policy implements pure ACP permission and question enforcement against a worker.Policy spec.
package policy

import (
	"os"
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

	resolvedRoot := resolveWithDeepestParent(cleanRoot)
	resolvedTarget := resolveWithDeepestParent(cleanTarget)

	rel, err := filepath.Rel(resolvedRoot, resolvedTarget)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func resolveWithDeepestParent(p string) string {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return filepath.Clean(resolved)
	}
	var tail []string
	cur := filepath.Clean(p)
	for {
		parent := filepath.Dir(cur)
		if parent == cur {
			return filepath.Clean(p)
		}
		tail = append(tail, filepath.Base(cur))
		cur = parent
		if _, err := os.Lstat(cur); err == nil {
			if resolved, err := filepath.EvalSymlinks(cur); err == nil {
				parts := make([]string, 0, len(tail)+1)
				parts = append(parts, resolved)
				for i := len(tail) - 1; i >= 0; i-- {
					parts = append(parts, tail[i])
				}
				return filepath.Clean(filepath.Join(parts...))
			}
		}
	}
}

func containsMergeCommand(cmd string) bool {
	for _, args := range splitShellCommands(cmd) {
		if len(args) == 0 {
			continue
		}
		bin := strings.ToLower(filepath.Base(args[0]))
		switch bin {
		case "gh":
			if hasSubcommandSequence(args[1:], "pr", "merge") {
				return true
			}
		case "git":
			sub, rest := firstGitSubcommand(args[1:])
			if sub == "merge" {
				return true
			}
			if sub == "push" && pushesToDefaultBranch(rest) {
				return true
			}
		}
	}
	return false
}

func hasSubcommandSequence(args []string, first, second string) bool {
	foundFirst := false
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		low := strings.ToLower(a)
		if !foundFirst {
			if low == first {
				foundFirst = true
			}
			continue
		}
		return low == second
	}
	return false
}

func firstGitSubcommand(args []string) (string, []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-C" || a == "-c" || a == "--git-dir" || a == "--work-tree" {
			i++
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		return strings.ToLower(a), args[i+1:]
	}
	return "", nil
}

func pushesToDefaultBranch(args []string) bool {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		low := strings.ToLower(a)
		if idx := strings.LastIndex(low, ":"); idx >= 0 {
			low = low[idx+1:]
		}
		low = strings.TrimPrefix(low, "refs/heads/")
		if low == "main" || low == "master" {
			return true
		}
	}
	return false
}

func splitShellCommands(cmd string) [][]string {
	var commands [][]string
	var cur []string
	var tok strings.Builder
	inToken := false
	var quote byte

	flushTok := func() {
		if inToken {
			cur = append(cur, tok.String())
			tok.Reset()
			inToken = false
		}
	}
	flushCmd := func() {
		flushTok()
		if len(cur) > 0 {
			commands = append(commands, cur)
			cur = nil
		}
	}

	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]
		if quote != 0 {
			if ch == '\\' && quote == '"' && i+1 < len(cmd) {
				i++
				tok.WriteByte(cmd[i])
				continue
			}
			if ch == quote {
				quote = 0
				continue
			}
			tok.WriteByte(ch)
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
			inToken = true
		case '\\':
			if i+1 < len(cmd) {
				i++
				tok.WriteByte(cmd[i])
				inToken = true
			}
		case ' ', '\t', '\r':
			flushTok()
		case '\n', ';', '|', '&':
			flushCmd()
		default:
			tok.WriteByte(ch)
			inToken = true
		}
	}
	flushCmd()
	return commands
}
