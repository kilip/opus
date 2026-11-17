package github

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestDecodeIssues_Fixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "issues.json"))
	if err != nil {
		t.Fatalf("read issues fixture: %v", err)
	}
	issues, err := DecodeIssues(data)
	if err != nil {
		t.Fatalf("DecodeIssues failed: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}

	first := issues[0]
	if first.Number != 104 || first.State != "OPEN" || first.URL != "https://github.com/kilip/opus/issues/104" {
		t.Fatalf("unexpected first issue: %+v", first)
	}
	if len(first.Labels) != 2 || first.Labels[0].Name != "status:in-progress" {
		t.Fatalf("unexpected first issue labels: %+v", first.Labels)
	}
	wantTime := time.Date(2026, 11, 17, 1, 30, 0, 0, time.UTC)
	if !first.UpdatedAt.Equal(wantTime) {
		t.Fatalf("got UpdatedAt %v, want %v", first.UpdatedAt, wantTime)
	}

	second := issues[1]
	if second.Number != 99 || second.State != "CLOSED" || second.StateReason != "COMPLETED" {
		t.Fatalf("unexpected second issue: %+v", second)
	}
}

func TestDecodePRs_Fixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "prs.json"))
	if err != nil {
		t.Fatalf("read prs fixture: %v", err)
	}
	prs, err := DecodePRs(data)
	if err != nil {
		t.Fatalf("DecodePRs failed: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}

	first := prs[0]
	if first.Number != 105 || !first.IsDraft || first.ReviewDecision != "REVIEW_REQUIRED" {
		t.Fatalf("unexpected first PR: %+v", first)
	}
	if len(first.StatusCheckRollup) != 2 {
		t.Fatalf("expected 2 status checks, got %d", len(first.StatusCheckRollup))
	}
	if first.StatusCheckRollup[0].Name != "test" || first.StatusCheckRollup[0].Conclusion != "SUCCESS" {
		t.Fatalf("unexpected CheckRun: %+v", first.StatusCheckRollup[0])
	}
	if first.StatusCheckRollup[1].Context != "ci/lint" || first.StatusCheckRollup[1].State != "PENDING" {
		t.Fatalf("unexpected StatusContext: %+v", first.StatusCheckRollup[1])
	}
	if len(first.ClosingIssuesReferences) != 1 || first.ClosingIssuesReferences[0].Number != 104 {
		t.Fatalf("unexpected closingIssuesReferences: %+v", first.ClosingIssuesReferences)
	}

	second := prs[1]
	if second.Number != 101 || second.State != "MERGED" || second.IsDraft || second.ReviewDecision != "APPROVED" {
		t.Fatalf("unexpected second PR: %+v", second)
	}
}

func TestDecodeInvalidJSON(t *testing.T) {
	if _, err := DecodeIssues([]byte("{bad")); err == nil {
		t.Fatal("expected error for invalid issues JSON")
	}
	if _, err := DecodePRs([]byte("{bad")); err == nil {
		t.Fatal("expected error for invalid prs JSON")
	}
}

func TestListIssues_CursorArgs(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "issues.json"))
	if err != nil {
		t.Fatalf("read issues fixture: %v", err)
	}

	var gotArgs []string
	runner := func(_ context.Context, args ...string) ([]byte, error) {
		gotArgs = slices.Clone(args)
		return data, nil
	}

	// Without cursor: no --search flag
	issues, err := ListIssues(context.Background(), runner, "kilip/opus", "")
	if err != nil {
		t.Fatalf("ListIssues without cursor: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	wantNoCursor := []string{"issue", "list", "--repo", "kilip/opus", "--state", "all", "--limit", "100", "--json", IssueJSONFields}
	if !slices.Equal(gotArgs, wantNoCursor) {
		t.Fatalf("args without cursor = %v, want %v", gotArgs, wantNoCursor)
	}

	// With cursor: includes --search updated:>=<cursor>
	cursor := "2026-11-17T01:30:00Z"
	if _, err := ListIssues(context.Background(), runner, "kilip/opus", cursor); err != nil {
		t.Fatalf("ListIssues with cursor: %v", err)
	}
	wantWithCursor := append(slices.Clone(wantNoCursor), "--search", "updated:>="+cursor)
	if !slices.Equal(gotArgs, wantWithCursor) {
		t.Fatalf("args with cursor = %v, want %v", gotArgs, wantWithCursor)
	}
}
