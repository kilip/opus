package domain

import "time"

// Size, count, and rate limits enforced by the agent tool service.
const (
	MaxDescriptionBytes  = 8 * 1024
	MaxListTasksLimit    = 50
	MaxActivityBodyBytes = 16 * 1024
	MaxReportBytes       = 64 * 1024
	RateLimitPerMinute   = 30
	RateLimitWindow      = time.Minute
	MaxRateLimitBuckets  = 10_000
)

// CallerContext identifies the authenticated agent caller, its active run/session
// bindings, and its allowed project scope.
type CallerContext struct {
	UserID            string
	NodeID            string
	RunID             string
	SessionID         string
	TaskID            string
	ProjectID         string
	AllowedProjectIDs []string
}

// AllowsProject reports whether projectID is within the caller's allowed scope.
func (c CallerContext) AllowsProject(projectID string) bool {
	if projectID == "" {
		return false
	}
	if c.ProjectID != "" && projectID == c.ProjectID {
		return true
	}
	for _, allowed := range c.AllowedProjectIDs {
		if allowed == projectID {
			return true
		}
	}
	return false
}

// ScopeProjectIDs returns the deduplicated list of project IDs accessible to the caller.
func (c CallerContext) ScopeProjectIDs() []string {
	seen := make(map[string]struct{}, len(c.AllowedProjectIDs)+1)
	var out []string
	add := func(id string) {
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	add(c.ProjectID)
	for _, id := range c.AllowedProjectIDs {
		add(id)
	}
	return out
}

// RateLimitKey returns the bucket key used to rate-limit this caller.
func (c CallerContext) RateLimitKey() string {
	switch {
	case c.RunID != "":
		return "run:" + c.RunID
	case c.UserID != "":
		return "user:" + c.UserID
	case c.NodeID != "":
		return "node:" + c.NodeID
	default:
		return "anonymous"
	}
}

// AcceptanceCriterion records verification status for a single acceptance criterion.
type AcceptanceCriterion struct {
	ID        string `json:"id,omitempty" jsonschema:"Criterion identifier (e.g. AC1)"`
	Criterion string `json:"criterion,omitempty" jsonschema:"Criterion description"`
	Met       bool   `json:"met" jsonschema:"Whether the acceptance criterion was met"`
	Evidence  string `json:"evidence,omitempty" jsonschema:"Evidence or verification output"`
}

// Report is the structured completion report persisted on a Run.
type Report struct {
	Summary            string                `json:"summary" jsonschema:"Concise summary of changes and outcome"`
	PRURL              string                `json:"pr_url,omitempty" jsonschema:"Pull request URL if created"`
	CommitSHA          string                `json:"commit_sha,omitempty" jsonschema:"Git commit SHA if committed"`
	Assumptions        []string              `json:"assumptions,omitempty" jsonschema:"Assumptions made during implementation"`
	AcceptanceCriteria []AcceptanceCriterion `json:"acceptance_criteria,omitempty" jsonschema:"Acceptance criteria verification results"`
}
