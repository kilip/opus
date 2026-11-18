package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kilip/opus/internal/worker/domain"
	"github.com/kilip/opus/internal/worker/policy"
	"github.com/kilip/opus/internal/worker/provider/agy"
)

func (w *Worker) resolveAgyBin() string {
	if w.cfg.AgyBin != "" {
		return w.cfg.AgyBin
	}
	if env := os.Getenv("WORKER_AGY_BIN"); env != "" {
		return env
	}
	return "agy"
}

func (w *Worker) runStart(ctx context.Context, cmd StartRun) error {
	bin := w.resolveAgyBin()

	// Preflight agy availability and auth before creating any git worktree.
	installed, authenticated := agy.ProbeAuth(ctx, bin)
	if !installed {
		return fmt.Errorf("agy is not installed (%s)", bin)
	}
	if !authenticated {
		return errors.New("agy is not authenticated (run: agy auth login)")
	}

	gitMgr := w.Git()
	repoDir, err := gitMgr.EnsureRepo(ctx, cmd.ProjectSlug, cmd.RepoURL, cmd.MappedPath)
	if err != nil {
		return err
	}
	wt, err := gitMgr.PrepareWorktree(ctx, repoDir, cmd.ProjectSlug, cmd.TaskKey, cmd.Branch, cmd.DefaultBranch)
	if err != nil {
		return err
	}

	var policySpec domain.Policy
	if cmd.Policy != nil {
		policySpec = *cmd.Policy
	}
	if policySpec.WorktreeRoot == "" {
		policySpec.WorktreeRoot = wt
	}
	enforcer := policy.NewEnforcer(policySpec)

	client, err := agy.Start(ctx, agy.Config{
		Bin: bin,
		Cwd: wt,
		Env: os.Environ(),
		OnUpdate: func(params agy.SessionUpdateParams) {
			raw, marshalErr := json.Marshal(params)
			if marshalErr != nil {
				return
			}
			_ = w.sendWorkerEvent(w.baseContext(), domain.Event{
				Type:      domain.EventSessionUpdate,
				Kind:      domain.EventSessionUpdate,
				RunID:     cmd.RunID,
				SessionID: cmd.SessionID,
				Payload:   raw,
			})
		},
		OnPermission: func(_ context.Context, req agy.PermissionRequest) (agy.PermissionResponse, error) {
			permReq := parsePermissionRequest(wt, req.ToolCall)
			dec := enforcer.CheckPermission(permReq)
			if !dec.Allowed {
				return agy.PermissionResponse{
					Outcome: agy.PermissionOutcome{Outcome: "cancelled"},
				}, nil
			}
			optID := pickAllowOptionID(req.Options)
			return agy.PermissionResponse{
				Outcome: agy.PermissionOutcome{Outcome: "selected", OptionID: optID},
			}, nil
		},
		OnQuestion: func(_ context.Context, _ agy.QuestionRequest) (agy.QuestionResponse, error) {
			dec := enforcer.HandleQuestion()
			if !dec.Answered {
				return agy.QuestionResponse{Skipped: true}, nil
			}
			return agy.QuestionResponse{
				Answers: map[string]any{"answer": dec.Answer},
			}, nil
		},
	})
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	if _, err := client.Initialize(ctx, agy.InitializeParams{ProtocolVersion: 1}); err != nil {
		return err
	}

	sessRes, err := w.StartAgySession(ctx, client, wt, cmd, "", nil)
	if err != nil {
		return err
	}

	_, promptErr := client.Prompt(ctx, agy.SessionPromptParams{
		SessionID: sessRes.SessionID,
		Prompt:    []agy.PromptContent{{Type: "text", Text: cmd.Prompt}},
	})
	if ctx.Err() != nil {
		cancelCtx, cancel := context.WithTimeout(context.Background(), agy.DefaultCancelGrace)
		_ = client.Cancel(cancelCtx, sessRes.SessionID)
		cancel()
		return nil
	}
	if promptErr != nil {
		return promptErr
	}

	return w.sendWorkerEvent(w.baseContext(), domain.Event{
		Type:      domain.EventSessionCompleted,
		Kind:      domain.EventSessionCompleted,
		RunID:     cmd.RunID,
		SessionID: cmd.SessionID,
	})
}

func parsePermissionRequest(worktreeRoot string, raw json.RawMessage) policy.PermissionRequest {
	var tc struct {
		Kind     string `json:"kind"`
		Name     string `json:"name"`
		ToolName string `json:"toolName"`
		Path     string `json:"path"`
		Subagent string `json:"subagent"`
		Command  string `json:"command"`
	}
	_ = json.Unmarshal(raw, &tc)

	targetPath := tc.Path
	if targetPath != "" && !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(worktreeRoot, targetPath)
	}
	if targetPath == "" && (tc.Kind == "command" || tc.Kind == "file_edit" || tc.Kind == "git") {
		targetPath = worktreeRoot
	}

	toolName := tc.ToolName
	if toolName == "" {
		toolName = tc.Name
	}
	kind := policy.PermissionKind(strings.ToLower(tc.Kind))
	if kind == "" && toolName != "" {
		kind = policy.KindMCPTool
	}
	return policy.PermissionRequest{
		Kind:     kind,
		Path:     targetPath,
		ToolName: toolName,
		Subagent: tc.Subagent,
		Command:  tc.Command,
	}
}

func pickAllowOptionID(opts []agy.PermissionOption) string {
	for _, o := range opts {
		if strings.Contains(strings.ToLower(o.Kind), "allow") {
			return o.OptionID
		}
	}
	if len(opts) > 0 {
		return opts[0].OptionID
	}
	return "allow"
}
