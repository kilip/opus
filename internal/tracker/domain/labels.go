package domain

import (
	"maps"
	"strings"
	"unicode/utf8"

	"github.com/kilip/opus/internal/task"
)

// MaxLabelLength is the maximum permitted length in runes for a status label.
const MaxLabelLength = 50

var defaultStatusLabels = map[string]string{
	string(task.StatusBacklog):    "status:backlog",
	string(task.StatusTodo):       "status:todo",
	string(task.StatusInProgress): "status:in-progress",
	string(task.StatusReview):     "status:review",
	string(task.StatusBlocked):    "status:blocked",
}

// DefaultStatusLabels returns a copy of the built-in status-to-label map.
// StatusDone intentionally has no label because completion maps from issue/PR state.
func DefaultStatusLabels() map[string]string {
	return maps.Clone(defaultStatusLabels)
}

// IsMappableStatus reports whether s is a valid task status that can have a tracker label.
func IsMappableStatus(s string) bool {
	_, ok := defaultStatusLabels[s]
	return ok
}

// ValidateLabelOverrides validates a partial or full status-to-label override map.
func ValidateLabelOverrides(overrides map[string]string) error {
	for k, v := range overrides {
		if !IsMappableStatus(k) {
			return ErrInvalidLabelConfig
		}
		if v == "" || strings.TrimSpace(v) != v || utf8.RuneCountInString(v) > MaxLabelLength {
			return ErrInvalidLabelConfig
		}
	}
	return nil
}

// ResolveEffectiveLabels merges default, global, and project label overrides per key.
func ResolveEffectiveLabels(global, project map[string]string) EffectiveLabelMap {
	statusToLabel := DefaultStatusLabels()
	for k, v := range global {
		if IsMappableStatus(k) && v != "" {
			statusToLabel[k] = v
		}
	}
	for k, v := range project {
		if IsMappableStatus(k) && v != "" {
			statusToLabel[k] = v
		}
	}

	labelToStatus := make(map[string]task.TaskStatus, len(statusToLabel))
	for status, label := range statusToLabel {
		labelToStatus[label] = task.TaskStatus(status)
	}

	return EffectiveLabelMap{
		StatusToLabel: statusToLabel,
		LabelToStatus: labelToStatus,
	}
}

// ValidateEffective checks that no two statuses resolve to the same label in the merged map.
func ValidateEffective(global, project map[string]string) error {
	effective := ResolveEffectiveLabels(global, project)
	seen := make(map[string]struct{}, len(effective.StatusToLabel))
	for _, label := range effective.StatusToLabel {
		if _, dup := seen[label]; dup {
			return ErrInvalidLabelConfig
		}
		seen[label] = struct{}{}
	}
	return nil
}

// ValidateGlobalLabels validates global tracker settings statusLabels against defaults.
func ValidateGlobalLabels(global map[string]string) error {
	if global == nil {
		return ErrInvalidLabelConfig
	}
	if err := ValidateLabelOverrides(global); err != nil {
		return err
	}
	return ValidateEffective(global, nil)
}

// ValidateProjectLabels validates project-level overrides against global settings and defaults.
func ValidateProjectLabels(global, project map[string]string) error {
	if err := ValidateLabelOverrides(project); err != nil {
		return err
	}
	return ValidateEffective(global, project)
}
