package domain_test

import (
	"errors"
	"maps"
	"strings"
	"testing"

	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker/domain"
)

func TestDefaultStatusLabels(t *testing.T) {
	defaults := domain.DefaultStatusLabels()
	want := map[string]string{
		string(task.StatusBacklog):    "status:backlog",
		string(task.StatusTodo):       "status:todo",
		string(task.StatusInProgress): "status:in-progress",
		string(task.StatusReview):     "status:review",
		string(task.StatusBlocked):    "status:blocked",
	}
	if !maps.Equal(defaults, want) {
		t.Fatalf("got defaults %v, want %v", defaults, want)
	}
	if _, ok := defaults[string(task.StatusDone)]; ok {
		t.Fatal("expected StatusDone not to have a default label")
	}

	// Returned map must be a defensive copy.
	defaults["todo"] = "mutated"
	if domain.DefaultStatusLabels()["todo"] != "status:todo" {
		t.Fatal("DefaultStatusLabels returned mutable package map")
	}
}

func TestValidateGlobalLabels(t *testing.T) {
	tests := []struct {
		name    string
		labels  map[string]string
		wantErr error
	}{
		{
			name:    "nil map rejected",
			labels:  nil,
			wantErr: domain.ErrInvalidLabelConfig,
		},
		{
			name:   "empty map resets to defaults",
			labels: map[string]string{},
		},
		{
			name:   "partial override valid",
			labels: map[string]string{"todo": "queue", "blocked": "waiting"},
		},
		{
			name:   "max length 50 runes valid",
			labels: map[string]string{"todo": strings.Repeat("a", 50)},
		},
		{
			name:   "swap two default labels valid",
			labels: map[string]string{"backlog": "status:todo", "todo": "status:backlog"},
		},
		{
			name:    "done status rejected",
			labels:  map[string]string{"done": "status:done"},
			wantErr: domain.ErrInvalidLabelConfig,
		},
		{
			name:    "unknown status rejected",
			labels:  map[string]string{"wip": "status:wip"},
			wantErr: domain.ErrInvalidLabelConfig,
		},
		{
			name:    "empty value rejected",
			labels:  map[string]string{"todo": ""},
			wantErr: domain.ErrInvalidLabelConfig,
		},
		{
			name:    "whitespace only value rejected",
			labels:  map[string]string{"todo": "   "},
			wantErr: domain.ErrInvalidLabelConfig,
		},
		{
			name:    "leading whitespace rejected",
			labels:  map[string]string{"todo": " queue"},
			wantErr: domain.ErrInvalidLabelConfig,
		},
		{
			name:    "trailing whitespace rejected",
			labels:  map[string]string{"todo": "queue "},
			wantErr: domain.ErrInvalidLabelConfig,
		},
		{
			name:    "over 50 runes rejected",
			labels:  map[string]string{"todo": strings.Repeat("a", 51)},
			wantErr: domain.ErrInvalidLabelConfig,
		},
		{
			name:    "duplicate label within global overrides",
			labels:  map[string]string{"todo": "same", "blocked": "same"},
			wantErr: domain.ErrInvalidLabelConfig,
		},
		{
			name:    "collision with unoverridden default label",
			labels:  map[string]string{"todo": "status:backlog"},
			wantErr: domain.ErrInvalidLabelConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateGlobalLabels(tt.labels)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateProjectLabels_AndResolveEffective(t *testing.T) {
	global := map[string]string{
		"backlog": "icebox",
		"todo":    "queue",
	}

	tests := []struct {
		name    string
		global  map[string]string
		project map[string]string
		wantErr error
	}{
		{
			name:    "nil project override allowed",
			global:  global,
			project: nil,
		},
		{
			name:    "valid project override",
			global:  global,
			project: map[string]string{"todo": "next-up"},
		},
		{
			name:    "project overrides to collide with global",
			global:  global,
			project: map[string]string{"in-progress": "icebox"},
			wantErr: domain.ErrInvalidLabelConfig,
		},
		{
			name:    "project overrides to collide with default",
			global:  global,
			project: map[string]string{"todo": "status:review"},
			wantErr: domain.ErrInvalidLabelConfig,
		},
		{
			name:    "project swaps with global on overridden key",
			global:  map[string]string{"todo": "status:backlog", "backlog": "icebox"},
			project: map[string]string{"todo": "triage"},
		},
		{
			name:    "done key in project rejected",
			global:  global,
			project: map[string]string{"done": "completed"},
			wantErr: domain.ErrInvalidLabelConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateProjectLabels(tt.global, tt.project)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
		})
	}

	eff := domain.ResolveEffectiveLabels(global, map[string]string{"todo": "sprint-ready"})
	wantStatusToLabel := map[string]string{
		"backlog":     "icebox",
		"todo":        "sprint-ready",
		"in-progress": "status:in-progress",
		"review":      "status:review",
		"blocked":     "status:blocked",
	}
	if !maps.Equal(eff.StatusToLabel, wantStatusToLabel) {
		t.Fatalf("got StatusToLabel %v, want %v", eff.StatusToLabel, wantStatusToLabel)
	}
	if eff.LabelToStatus["sprint-ready"] != task.StatusTodo {
		t.Fatalf("expected reverse lookup sprint-ready -> todo, got %q", eff.LabelToStatus["sprint-ready"])
	}
	if eff.LabelToStatus["icebox"] != task.StatusBacklog {
		t.Fatalf("expected reverse lookup icebox -> backlog, got %q", eff.LabelToStatus["icebox"])
	}
}
