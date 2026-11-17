package domain_test

import (
	"errors"
	"testing"

	"github.com/kilip/opus/internal/run/domain"
)

func TestTransition_Matrix(t *testing.T) {
	expectedValid := map[domain.RunState]map[domain.RunState]bool{
		domain.StateQueued: {
			domain.StatePreparing: true,
			domain.StatePaused:    true,
			domain.StateFailed:    true,
			domain.StateCancelled: true,
		},
		domain.StatePreparing: {
			domain.StateRunning:   true,
			domain.StateFailed:    true,
			domain.StateNeedsInfo: true,
			domain.StatePaused:    true,
			domain.StateCancelled: true,
		},
		domain.StateRunning: {
			domain.StateMerged:    true,
			domain.StateFailed:    true,
			domain.StateNeedsInfo: true,
			domain.StatePaused:    true,
			domain.StateCancelled: true,
		},
		domain.StatePaused: {
			domain.StateQueued:    true,
			domain.StateFailed:    true,
			domain.StateCancelled: true,
		},
	}

	for _, from := range domain.AllStates {
		for _, to := range domain.AllStates {
			wantValid := expectedValid[from][to]
			t.Run(string(from)+"->"+string(to), func(t *testing.T) {
				gotValid := domain.CanTransition(from, to)
				if gotValid != wantValid {
					t.Fatalf("CanTransition(%s, %s) = %v, want %v", from, to, gotValid, wantValid)
				}

				err := domain.Transition(from, to)
				if wantValid {
					if err != nil {
						t.Fatalf("Transition(%s, %s) unexpected error: %v", from, to, err)
					}
					return
				}
				if !errors.Is(err, domain.ErrInvalidTransition) {
					t.Fatalf("Transition(%s, %s) error = %v, want ErrInvalidTransition", from, to, err)
				}
			})
		}
	}
}

func TestTransition_InvalidStates(t *testing.T) {
	cases := []struct {
		name string
		from domain.RunState
		to   domain.RunState
	}{
		{name: "unknown from", from: domain.RunState("bogus"), to: domain.StatePreparing},
		{name: "unknown to", from: domain.StateQueued, to: domain.RunState("bogus")},
		{name: "empty from", from: domain.RunState(""), to: domain.StateQueued},
		{name: "empty to", from: domain.StateQueued, to: domain.RunState("")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if domain.CanTransition(tc.from, tc.to) {
				t.Fatalf("CanTransition(%q, %q) = true, want false", tc.from, tc.to)
			}
			err := domain.Transition(tc.from, tc.to)
			if !errors.Is(err, domain.ErrInvalidState) {
				t.Fatalf("Transition(%q, %q) error = %v, want ErrInvalidState", tc.from, tc.to, err)
			}
		})
	}
}

func TestModeAndStateHelpers(t *testing.T) {
	if !domain.IsValidMode(domain.ModeVibe) {
		t.Fatalf("expected ModeVibe to be valid")
	}
	if domain.IsValidMode(domain.RunMode("to_pr")) {
		t.Fatalf("expected unimplemented mode to be invalid in MVP")
	}

	terminal := map[domain.RunState]bool{
		domain.StateMerged:    true,
		domain.StateFailed:    true,
		domain.StateNeedsInfo: true,
		domain.StateCancelled: true,
	}
	finished := map[domain.RunState]bool{
		domain.StateMerged:    true,
		domain.StateFailed:    true,
		domain.StateNeedsInfo: true,
		domain.StatePaused:    true,
		domain.StateCancelled: true,
	}

	for _, s := range domain.AllStates {
		if !domain.IsValidState(s) {
			t.Fatalf("IsValidState(%s) = false, want true", s)
		}
		if got := domain.IsTerminalState(s); got != terminal[s] {
			t.Fatalf("IsTerminalState(%s) = %v, want %v", s, got, terminal[s])
		}
		if got := domain.IsFinishedOutcome(s); got != finished[s] {
			t.Fatalf("IsFinishedOutcome(%s) = %v, want %v", s, got, finished[s])
		}
	}
}
