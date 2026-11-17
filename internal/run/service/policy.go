package service

import (
	"fmt"

	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/worker"
)

const vibeQuestionAutoReply = "decide yourself and record the assumption in the PR description"

// BuildPolicy constructs the worker execution policy spec for a run mode and worktree root.
func BuildPolicy(mode domain.RunMode, worktreeRoot string) (worker.Policy, error) {
	// ponytail: MVP supports "vibe" mode only; add policy profiles for "to_pr" and "plan" post-MVP.
	if mode != domain.ModeVibe {
		return worker.Policy{}, fmt.Errorf("%w: %s", domain.ErrInvalidMode, mode)
	}

	return worker.Policy{
		WorktreeRoot:      worktreeRoot,
		AllowedSubagents:  []string{"scout", "planner", "coder", "verifier", "pr-manager"},
		AllowedMCPTools:   []string{"get_task", "list_tasks", "set_status", "add_activity", "submit_report"},
		QuestionAutoReply: vibeQuestionAutoReply,
		MaxQuestions:      1,
		AllowMerge:        true,
	}, nil
}
