package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	// IssueViewJSONFields lists JSON fields requested from `gh issue view`.
	IssueViewJSONFields = "number,state,stateReason,labels,url,comments"

	opusActivityMarkerPrefix = "<!-- opus:activity:"
	opusActivityMarkerSuffix = " -->"
)

var (
	// ErrIssueNotFound indicates the GitHub issue was deleted or does not exist.
	ErrIssueNotFound = errors.New("github issue not found")
	// ErrIssueTransferred indicates the GitHub issue was transferred to another repository.
	ErrIssueTransferred = errors.New("github issue transferred")
)

// IssueComment represents a comment entry returned by `gh issue view --json comments`.
type IssueComment struct {
	ID   string `json:"id,omitempty"`
	Body string `json:"body"`
}

// IssueView represents detailed issue state returned by `gh issue view`.
type IssueView struct {
	Number      int            `json:"number"`
	State       string         `json:"state"`
	StateReason string         `json:"stateReason"`
	Labels      []Label        `json:"labels"`
	URL         string         `json:"url"`
	Comments    []IssueComment `json:"comments"`
}

// FormatActivityMarker returns the hidden HTML comment marker for an Opus activity ID.
func FormatActivityMarker(activityID string) string {
	return opusActivityMarkerPrefix + activityID + opusActivityMarkerSuffix
}

// HasOpusActivityMarker reports whether body contains an Opus activity echo marker.
func HasOpusActivityMarker(body string) bool {
	return strings.Contains(body, opusActivityMarkerPrefix)
}

// ViewIssue fetches the current remote issue state, labels, URL, and comments.
func ViewIssue(ctx context.Context, runner Runner, repo string, number int) (IssueView, error) {
	out, err := runner(ctx, "issue", "view", strconv.Itoa(number), "--repo", repo, "--json", IssueViewJSONFields)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "could not resolve to an issue") ||
			strings.Contains(lower, "not found") ||
			strings.Contains(lower, "http 404") {
			return IssueView{}, fmt.Errorf("%w: %w", ErrIssueNotFound, err)
		}
		if strings.Contains(lower, "transferred") {
			return IssueView{}, fmt.Errorf("%w: %w", ErrIssueTransferred, err)
		}
		return IssueView{}, err
	}

	var view IssueView
	if err := json.Unmarshal(out, &view); err != nil {
		return IssueView{}, fmt.Errorf("decode github issue view: %w", err)
	}
	if view.URL != "" && !strings.Contains(strings.ToLower(view.URL), "/"+strings.ToLower(strings.TrimSpace(repo))+"/issues/") {
		return IssueView{}, fmt.Errorf("%w: %s", ErrIssueTransferred, view.URL)
	}
	return view, nil
}

// ListRepoLabels returns existing label names in repo.
func ListRepoLabels(ctx context.Context, runner Runner, repo string) ([]string, error) {
	out, err := runner(ctx, "label", "list", "--repo", repo, "--limit", "100", "--json", "name")
	if err != nil {
		return nil, err
	}
	var labels []Label
	if err := json.Unmarshal(out, &labels); err != nil {
		return nil, fmt.Errorf("decode github labels: %w", err)
	}
	names := make([]string, len(labels))
	for i, l := range labels {
		names[i] = l.Name
	}
	return names, nil
}

// CreateLabel creates (or updates) a label in repo.
func CreateLabel(ctx context.Context, runner Runner, repo, name string) error {
	_, err := runner(ctx, "label", "create", name, "--repo", repo, "--force")
	return err
}

// EditIssueLabels adds and removes labels on an issue.
func EditIssueLabels(ctx context.Context, runner Runner, repo string, number int, add, remove []string) error {
	if len(add) == 0 && len(remove) == 0 {
		return nil
	}
	args := []string{"issue", "edit", strconv.Itoa(number), "--repo", repo}
	for _, l := range add {
		args = append(args, "--add-label", l)
	}
	for _, l := range remove {
		args = append(args, "--remove-label", l)
	}
	_, err := runner(ctx, args...)
	return err
}

// CloseIssue closes an issue with the given reason (e.g. "completed").
func CloseIssue(ctx context.Context, runner Runner, repo string, number int, reason string) error {
	args := []string{"issue", "close", strconv.Itoa(number), "--repo", repo}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	_, err := runner(ctx, args...)
	return err
}

// ReopenIssue reopens a closed issue.
func ReopenIssue(ctx context.Context, runner Runner, repo string, number int) error {
	_, err := runner(ctx, "issue", "reopen", strconv.Itoa(number), "--repo", repo)
	return err
}

// CommentIssue posts a comment on an issue.
func CommentIssue(ctx context.Context, runner Runner, repo string, number int, body string) error {
	_, err := runner(ctx, "issue", "comment", strconv.Itoa(number), "--repo", repo, "--body", body)
	return err
}
