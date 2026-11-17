package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RunMode defines the execution mode of a Run.
// ponytail: MVP supports "vibe" mode only; add "to_pr" / "plan" modes and gate states post-MVP.
type RunMode string

const (
	ModeVibe RunMode = "vibe"
)

// IsValidMode reports whether m is a supported run mode.
func IsValidMode(m RunMode) bool {
	return m == ModeVibe
}

// RunState represents the lifecycle state of a Run.
type RunState string

const (
	StateQueued    RunState = "queued"
	StatePreparing RunState = "preparing"
	StateRunning   RunState = "running"
	StateMerged    RunState = "merged"
	StateFailed    RunState = "failed"
	StateNeedsInfo RunState = "needs-info"
	StatePaused    RunState = "paused"
	StateCancelled RunState = "cancelled"
)

// AllStates lists every valid RunState.
var AllStates = []RunState{
	StateQueued,
	StatePreparing,
	StateRunning,
	StateMerged,
	StateFailed,
	StateNeedsInfo,
	StatePaused,
	StateCancelled,
}

// IsValidState reports whether s is a known run state.
func IsValidState(s RunState) bool {
	switch s {
	case StateQueued,
		StatePreparing,
		StateRunning,
		StateMerged,
		StateFailed,
		StateNeedsInfo,
		StatePaused,
		StateCancelled:
		return true
	default:
		return false
	}
}

// IsTerminalState reports whether s is a terminal run state with no outgoing transitions.
func IsTerminalState(s RunState) bool {
	switch s {
	case StateMerged, StateFailed, StateNeedsInfo, StateCancelled:
		return true
	default:
		return false
	}
}

// IsFinishedOutcome reports whether s is an outcome that emits run.finished.
func IsFinishedOutcome(s RunState) bool {
	switch s {
	case StateMerged, StateFailed, StateNeedsInfo, StatePaused, StateCancelled:
		return true
	default:
		return false
	}
}

var validTransitions = map[RunState]map[RunState]bool{
	StateQueued: {
		StatePreparing: true,
		StatePaused:    true,
		StateFailed:    true,
		StateCancelled: true,
	},
	StatePreparing: {
		StateRunning:   true,
		StateFailed:    true,
		StateNeedsInfo: true,
		StatePaused:    true,
		StateCancelled: true,
	},
	StateRunning: {
		StateMerged:    true,
		StateFailed:    true,
		StateNeedsInfo: true,
		StatePaused:    true,
		StateCancelled: true,
	},
	StatePaused: {
		StateQueued:    true,
		StateFailed:    true,
		StateCancelled: true,
	},
}

// CanTransition reports whether transitioning from `from` to `to` is valid.
func CanTransition(from, to RunState) bool {
	targets, ok := validTransitions[from]
	if !ok {
		return false
	}
	return targets[to]
}

// Transition validates a state transition from `from` to `to`.
func Transition(from, to RunState) error {
	if !IsValidState(from) || !IsValidState(to) {
		return ErrInvalidState
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
	}
	return nil
}

// Run represents an automated execution attempt for a Task.
type Run struct {
	ID            uuid.UUID
	TaskID        uuid.UUID
	ProjectID     uuid.UUID
	NodeID        *uuid.UUID
	SessionID     *uuid.UUID
	Mode          RunMode
	State         RunState
	StageAttempts int
	Branch        string
	PRNumber      *int
	Report        json.RawMessage
	PausedUntil   *time.Time
	FailureReason *string
	CreatedAt     time.Time
	StartedAt     *time.Time
	EndedAt       *time.Time
	UpdatedAt     time.Time
}

// CreateInput holds parameters for creating a queued Run.
type CreateInput struct {
	TaskID    uuid.UUID
	ProjectID uuid.UUID
	NodeID    *uuid.UUID
	Mode      RunMode
	Branch    string
}

// TransitionInput holds optional fields updated alongside a state transition.
type TransitionInput struct {
	NodeID           *uuid.UUID
	SessionID        *uuid.UUID
	Branch           *string
	PRNumber         *int
	Report           json.RawMessage
	PausedUntil      *time.Time
	FailureReason    *string
	IncrementAttempt bool
}

// ListFilter filters runs in List queries.
type ListFilter struct {
	TaskID    *uuid.UUID
	ProjectID *uuid.UUID
	State     *RunState
}
