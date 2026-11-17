package domain

import "errors"

var (
	// ErrProjectNotFound indicates the project does not exist.
	ErrProjectNotFound = errors.New("PROJECT_NOT_FOUND")
	// ErrInvalidSlug indicates the project slug format is invalid.
	ErrInvalidSlug = errors.New("PROJECT_INVALID_SLUG")
	// ErrDuplicateSlug indicates a project with this slug already exists.
	ErrDuplicateSlug = errors.New("PROJECT_DUPLICATE_SLUG")
	// ErrInvalidRepo indicates the repo provider or repo ref is invalid.
	ErrInvalidRepo = errors.New("PROJECT_INVALID_REPO")
	// ErrPatchForbidden indicates forbidden immutable fields were present in PATCH payload.
	ErrPatchForbidden = errors.New("PROJECT_PATCH_FORBIDDEN")
	// ErrProjectHasTasks indicates the project cannot be deleted because it has associated tasks.
	ErrProjectHasTasks = errors.New("PROJECT_HAS_TASKS")
	// ErrInvalidPath indicates the project node path is not an absolute path.
	ErrInvalidPath = errors.New("PROJECT_INVALID_PATH")
	// ErrNodeNotFound indicates the node does not exist.
	ErrNodeNotFound = errors.New("NODE_NOT_FOUND")
	// ErrInvalidTrackerLabels indicates project tracker status-label overrides are invalid.
	ErrInvalidTrackerLabels = errors.New("PROJECT_INVALID_TRACKER_LABELS")
)
