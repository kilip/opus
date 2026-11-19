package domain

import "errors"

var (
	// ErrEmptyGitHubToken is returned when an empty GitHub token is submitted.
	ErrEmptyGitHubToken = errors.New("GitHub token is required")

	// ErrInvalidGitHubToken is returned when GitHub rejects the provided token.
	ErrInvalidGitHubToken = errors.New("invalid GitHub token")
)
