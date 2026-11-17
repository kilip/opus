package domain

import "strings"

// NormalizePRState maps a GitHub PR state string to PRState.
func NormalizePRState(raw string) PRState {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "MERGED":
		return PRStateMerged
	case "CLOSED":
		return PRStateClosed
	default:
		return PRStateOpen
	}
}
