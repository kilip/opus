package domain

import (
	"regexp"
	"strings"
)

const (
	RepoProviderGitHub = "github"
	DefaultBranchMain  = "main"
)

var (
	slugRegex    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
	repoRefRegex = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]*/[a-zA-Z0-9_][a-zA-Z0-9_.-]*$`)
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

// ValidateRepoRef checks that repo ref matches owner/name format without leading '-' or '..'.
func ValidateRepoRef(ref string) error {
	if strings.HasPrefix(ref, "-") || strings.Contains(ref, "..") || !repoRefRegex.MatchString(ref) {
		return ErrInvalidRepo
	}
	return nil
}
