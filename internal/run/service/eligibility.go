package service

import (
	"cmp"
	"context"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/kilip/opus/internal/run/domain"
	"github.com/kilip/opus/internal/task"
)

const (
	labelVibe          = "vibe"
	labelVibeFailed    = "vibe:failed"
	labelVibeNeedsInfo = "vibe:needs-info"
)

var (
	blockedByLineRe = regexp.MustCompile(`(?im)blocked\s+by:\s*([^\n]+)`)
	issueRefRe      = regexp.MustCompile(`#(\d+)`)
	scopeHeadingRe  = regexp.MustCompile(`(?im)^##\s+scope\b`)
	acHeadingRe     = regexp.MustCompile(`(?im)^##\s+acceptance\s+criteria\b`)
	dodHeadingRe    = regexp.MustCompile(`(?im)^##\s+definition\s+of\s+done\b`)
	backtickCmdRe   = regexp.MustCompile("`[^`\\n]+`")
)

func hasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if strings.EqualFold(strings.TrimSpace(l), target) {
			return true
		}
	}
	return false
}

// isBasicEligible checks AC2 task-level criteria (labels, source, status).
func isBasicEligible(t task.Record) bool {
	if t.Source != task.SourceGitHub || t.Status == task.StatusDone {
		return false
	}
	if !hasLabel(t.Labels, labelVibe) {
		return false
	}
	if hasLabel(t.Labels, labelVibeFailed) || hasLabel(t.Labels, labelVibeNeedsInfo) {
		return false
	}
	return true
}

// parseBlockedByIssues extracts issue numbers referenced in `Blocked by: #N` clauses.
func parseBlockedByIssues(description string) []string {
	matches := blockedByLineRe.FindAllStringSubmatch(description, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var refs []string
	for _, m := range matches {
		for _, numMatch := range issueRefRe.FindAllStringSubmatch(m[1], -1) {
			num := numMatch[1]
			if _, ok := seen[num]; !ok {
				seen[num] = struct{}{}
				refs = append(refs, num)
			}
		}
	}
	return refs
}

// lintTaskStructure validates AC4 required headings and backticked command.
// Returns missing items if any, or nil when valid.
func lintTaskStructure(description string) []string {
	var missing []string
	if !scopeHeadingRe.MatchString(description) {
		missing = append(missing, "`## Scope` heading")
	}
	if !acHeadingRe.MatchString(description) {
		missing = append(missing, "`## Acceptance criteria` heading")
	}
	if !dodHeadingRe.MatchString(description) {
		missing = append(missing, "`## Definition of Done` heading")
	}
	if !backtickCmdRe.MatchString(description) {
		missing = append(missing, "at least one backticked verification command")
	}
	return missing
}

// selectNode picks an online, agy-ready node mapped to the project.
// Prefers mappedPath != "" first, then sorts deterministically by node name A-Z.
func selectNode(
	ctx context.Context,
	candidates []domain.ProjectNodeCandidate,
	isAgyReady func(context.Context, uuid.UUID) bool,
) (domain.ProjectNodeCandidate, bool) {
	var ready []domain.ProjectNodeCandidate
	for _, c := range candidates {
		if !c.Online {
			continue
		}
		if isAgyReady != nil && !isAgyReady(ctx, c.NodeID) {
			continue
		}
		ready = append(ready, c)
	}
	if len(ready) == 0 {
		return domain.ProjectNodeCandidate{}, false
	}

	// ponytail: deterministic first-match selection (mappedPath first, then name A-Z); add round-robin/failover post-MVP.
	slices.SortFunc(ready, func(a, b domain.ProjectNodeCandidate) int {
		aMapped := a.MappedPath != ""
		bMapped := b.MappedPath != ""
		if aMapped != bMapped {
			if aMapped {
				return -1
			}
			return 1
		}
		if c := cmp.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		return cmp.Compare(a.NodeID.String(), b.NodeID.String())
	})
	return ready[0], true
}
