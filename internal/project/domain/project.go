package domain

import (
	"regexp"
)

const (
	RepoProviderGitHub = "github"
	DefaultBranchMain  = "main"
)

var (
	slugRegex    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
	repoRefRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)
)

// ValidateSlug checks if slug matches ^[a-z0-9][a-z0-9-]{0,62}$.
func ValidateSlug(slug string) error {
	if !slugRegex.MatchString(slug) {
		return ErrInvalidSlug
	}
	return nil
}

// ValidateRepoProvider checks that repo provider is "github".
func ValidateRepoProvider(provider string) error {
	if provider != RepoProviderGitHub {
		return ErrInvalidRepo
	}
	return nil
}

// ValidateRepoRef checks that repo ref matches owner/name format.
func ValidateRepoRef(ref string) error {
	if !repoRefRegex.MatchString(ref) {
		return ErrInvalidRepo
	}
	return nil
}
