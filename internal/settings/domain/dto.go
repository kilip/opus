package domain

import "time"

// GitHubTokenStatusResponse describes the configured GitHub token without exposing the secret.
type GitHubTokenStatusResponse struct {
	Configured   bool       `json:"configured"`
	Last4        string     `json:"last4,omitempty"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	EnvOverride  bool       `json:"envOverride"`
	NeedsReentry bool       `json:"needsReentry"`
}

// SetGitHubTokenRequest is the request payload for storing a GitHub token.
type SetGitHubTokenRequest struct {
	Token string `json:"token"`
}
