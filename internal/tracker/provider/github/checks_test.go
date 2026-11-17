package github

import (
	"testing"

	"github.com/kilip/opus/internal/tracker/domain"
)

func TestDeriveChecksSummary(t *testing.T) {
	tests := []struct {
		name   string
		checks []StatusCheck
		want   domain.ChecksSummary
	}{
		{
			name:   "empty",
			checks: nil,
			want:   domain.ChecksNone,
		},
		{
			name: "all success checkrun and statuscontext",
			checks: []StatusCheck{
				{TypeName: "CheckRun", Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
				{TypeName: "CheckRun", Name: "optional", Status: "COMPLETED", Conclusion: "SKIPPED"},
				{TypeName: "StatusContext", Context: "ci/lint", State: "SUCCESS"},
			},
			want: domain.ChecksPassing,
		},
		{
			name: "pending checkrun in progress",
			checks: []StatusCheck{
				{TypeName: "CheckRun", Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
				{TypeName: "CheckRun", Name: "build", Status: "IN_PROGRESS"},
			},
			want: domain.ChecksPending,
		},
		{
			name: "pending statuscontext",
			checks: []StatusCheck{
				{TypeName: "CheckRun", Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
				{TypeName: "StatusContext", Context: "ci/lint", State: "PENDING"},
			},
			want: domain.ChecksPending,
		},
		{
			name: "failing checkrun overrides pending and success",
			checks: []StatusCheck{
				{TypeName: "CheckRun", Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
				{TypeName: "StatusContext", Context: "ci/lint", State: "PENDING"},
				{TypeName: "CheckRun", Name: "vet", Status: "COMPLETED", Conclusion: "FAILURE"},
			},
			want: domain.ChecksFailing,
		},
		{
			name: "failing statuscontext",
			checks: []StatusCheck{
				{TypeName: "CheckRun", Name: "test", Status: "COMPLETED", Conclusion: "SUCCESS"},
				{TypeName: "StatusContext", Context: "ci/deploy", State: "ERROR"},
			},
			want: domain.ChecksFailing,
		},
		{
			name: "timed out checkrun is failing",
			checks: []StatusCheck{
				{TypeName: "CheckRun", Name: "e2e", Status: "COMPLETED", Conclusion: "TIMED_OUT"},
			},
			want: domain.ChecksFailing,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := DeriveChecksSummary(tc.checks); got != tc.want {
				t.Fatalf("DeriveChecksSummary() = %q, want %q", got, tc.want)
			}
		})
	}
}
