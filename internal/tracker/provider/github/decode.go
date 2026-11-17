package github

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	// IssueJSONFields lists the JSON fields requested from `gh issue list`.
	IssueJSONFields = "number,title,body,state,stateReason,labels,updatedAt,url"
	// PRJSONFields lists the JSON fields requested from `gh pr list`.
	PRJSONFields = "number,title,state,isDraft,reviewDecision,statusCheckRollup,closingIssuesReferences,updatedAt,url"
)

// Label represents a GitHub label returned by gh CLI JSON output.
type Label struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

// Issue represents a GitHub issue returned by `gh issue list --json ...`.
type Issue struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	State       string    `json:"state"`
	StateReason string    `json:"stateReason"`
	Labels      []Label   `json:"labels"`
	UpdatedAt   time.Time `json:"updatedAt"`
	URL         string    `json:"url"`
}

// StatusCheck represents an entry in `statusCheckRollup` (CheckRun or StatusContext).
type StatusCheck struct {
	TypeName   string `json:"__typename,omitempty"`
	Name       string `json:"name,omitempty"`
	Status     string `json:"status,omitempty"`
	Conclusion string `json:"conclusion,omitempty"`
	Context    string `json:"context,omitempty"`
	State      string `json:"state,omitempty"`
}

// ClosingIssueRef represents an entry in `closingIssuesReferences`.
type ClosingIssueRef struct {
	ID     string `json:"id,omitempty"`
	Number int    `json:"number"`
	URL    string `json:"url,omitempty"`
}

// PullRequest represents a GitHub pull request returned by `gh pr list --json ...`.
type PullRequest struct {
	Number                  int               `json:"number"`
	Title                   string            `json:"title"`
	State                   string            `json:"state"`
	IsDraft                 bool              `json:"isDraft"`
	ReviewDecision          string            `json:"reviewDecision"`
	StatusCheckRollup       []StatusCheck     `json:"statusCheckRollup"`
	ClosingIssuesReferences []ClosingIssueRef `json:"closingIssuesReferences"`
	UpdatedAt               time.Time         `json:"updatedAt"`
	URL                     string            `json:"url"`
}

// DecodeIssues decodes JSON output from `gh issue list --json ...`.
func DecodeIssues(data []byte) ([]Issue, error) {
	var issues []Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, fmt.Errorf("decode github issues: %w", err)
	}
	return issues, nil
}

// DecodePRs decodes JSON output from `gh pr list --json ...`.
func DecodePRs(data []byte) ([]PullRequest, error) {
	var prs []PullRequest
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, fmt.Errorf("decode github prs: %w", err)
	}
	return prs, nil
}

// CheckAuth verifies `gh auth status`.
func CheckAuth(ctx context.Context, runner Runner) error {
	_, err := runner(ctx, "auth", "status")
	return err
}

// ListIssues runs `gh issue list` for repo and decodes the JSON response.
// When since is non-empty, it filters issues via `--search "updated:>=" + since`.
func ListIssues(ctx context.Context, runner Runner, repo, since string) ([]Issue, error) {
	args := []string{"issue", "list", "--repo", repo, "--state", "all", "--limit", "100", "--json", IssueJSONFields}
	if since != "" {
		args = append(args, "--search", "updated:>="+since)
	}
	out, err := runner(ctx, args...)
	if err != nil {
		return nil, err
	}
	return DecodeIssues(out)
}

// ListPRs runs `gh pr list` for repo and decodes the JSON response.
// When since is non-empty, it filters PRs via `--search "updated:>=" + since`.
func ListPRs(ctx context.Context, runner Runner, repo, since string) ([]PullRequest, error) {
	args := []string{"pr", "list", "--repo", repo, "--state", "all", "--limit", "100", "--json", PRJSONFields}
	if since != "" {
		args = append(args, "--search", "updated:>="+since)
	}
	out, err := runner(ctx, args...)
	if err != nil {
		return nil, err
	}
	return DecodePRs(out)
}
