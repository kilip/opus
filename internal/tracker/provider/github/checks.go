package github

import (
	"strings"

	"github.com/kilip/opus/internal/tracker/domain"
)

// DeriveChecksSummary aggregates GitHub statusCheckRollup entries into a ChecksSummary:
// any failure -> failing; else any pending -> pending; all success -> passing; empty -> none.
func DeriveChecksSummary(checks []StatusCheck) domain.ChecksSummary {
	if len(checks) == 0 {
		return domain.ChecksNone
	}

	hasPending := false
	for _, c := range checks {
		switch classifyStatusCheck(c) {
		case domain.ChecksFailing:
			return domain.ChecksFailing
		case domain.ChecksPending:
			hasPending = true
		}
	}
	if hasPending {
		return domain.ChecksPending
	}
	return domain.ChecksPassing
}

func classifyStatusCheck(c StatusCheck) domain.ChecksSummary {
	// StatusContext uses State; CheckRun uses Status + Conclusion.
	if c.State != "" && c.Status == "" && c.Conclusion == "" {
		switch strings.ToUpper(strings.TrimSpace(c.State)) {
		case "FAILURE", "ERROR":
			return domain.ChecksFailing
		case "SUCCESS":
			return domain.ChecksPassing
		default:
			return domain.ChecksPending
		}
	}

	status := strings.ToUpper(strings.TrimSpace(c.Status))
	if status != "" && status != "COMPLETED" {
		return domain.ChecksPending
	}

	switch strings.ToUpper(strings.TrimSpace(c.Conclusion)) {
	case "FAILURE", "TIMED_OUT", "CANCELLED", "ACTION_REQUIRED", "STARTUP_FAILURE":
		return domain.ChecksFailing
	case "SUCCESS", "NEUTRAL", "SKIPPED":
		return domain.ChecksPassing
	default:
		return domain.ChecksPending
	}
}
