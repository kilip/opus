package service_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/run/service"
)

func TestBuildPolicy(t *testing.T) {
	spec, err := service.BuildPolicy(domain.ModeVibe, "/tmp/wt-120")
	if err != nil {
		t.Fatalf("BuildPolicy(vibe): %v", err)
	}

	if spec.WorktreeRoot != "/tmp/wt-120" {
		t.Fatalf("WorktreeRoot = %q, want /tmp/wt-120", spec.WorktreeRoot)
	}
	if !spec.AllowMerge {
		t.Fatalf("expected AllowMerge = true for vibe mode")
	}
	if spec.MaxQuestions != 1 {
		t.Fatalf("MaxQuestions = %d, want 1", spec.MaxQuestions)
	}
	if spec.QuestionAutoReply != "decide yourself and record the assumption in the PR description" {
		t.Fatalf("unexpected QuestionAutoReply: %q", spec.QuestionAutoReply)
	}

	wantSubagents := []string{"scout", "planner", "coder", "verifier", "pr-manager"}
	if !slices.Equal(spec.AllowedSubagents, wantSubagents) {
		t.Fatalf("AllowedSubagents = %v, want %v", spec.AllowedSubagents, wantSubagents)
	}
	wantMCP := []string{"get_task", "list_tasks", "set_status", "add_activity", "submit_report"}
	if !slices.Equal(spec.AllowedMCPTools, wantMCP) {
		t.Fatalf("AllowedMCPTools = %v, want %v", spec.AllowedMCPTools, wantMCP)
	}

	if _, err := service.BuildPolicy(domain.RunMode("to_pr"), "/tmp/wt-120"); !errors.Is(err, domain.ErrInvalidMode) {
		t.Fatalf("expected ErrInvalidMode for unsupported mode, got %v", err)
	}
}
