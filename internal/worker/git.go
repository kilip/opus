package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitManager manages repository clones and per-task git worktrees under <APP_HOME>.
type GitManager struct {
	homeDir string
}

// NewGitManager returns a GitManager rooted at homeDir (<APP_HOME>).
func NewGitManager(homeDir string) *GitManager {
	return &GitManager{homeDir: homeDir}
}

// Git returns a GitManager rooted at the worker's configured home directory.
func (w *Worker) Git() *GitManager {
	return NewGitManager(w.homeDir)
}

// EnsureRepo ensures a git repository is available on disk.
// When mappedPath is non-empty, it validates that mappedPath exists and is a git
// repository without cloning. Otherwise it clones repoURL into <APP_HOME>/repos/<slug>
// (skipping clone if already present).
func (m *GitManager) EnsureRepo(ctx context.Context, slug, repoURL, mappedPath string) (string, error) {
	if mappedPath != "" {
		info, err := os.Stat(mappedPath)
		if err != nil {
			return "", fmt.Errorf("mapped repository path %q: %w", mappedPath, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("mapped repository path %q is not a directory", mappedPath)
		}
		if !isGitRepo(ctx, mappedPath) {
			return "", fmt.Errorf("mapped repository path %q is not a git repository", mappedPath)
		}
		return mappedPath, nil
	}

	if err := validatePathSegment("project slug", slug); err != nil {
		return "", err
	}

	repoDir := filepath.Join(m.homeDir, "repos", slug)
	if isGitRepo(ctx, repoDir) {
		return repoDir, nil
	}

	if repoURL == "" {
		return "", fmt.Errorf("repo URL is required to clone %q", slug)
	}
	if err := ValidateRepoURL(repoURL); err != nil {
		return "", fmt.Errorf("invalid repo URL for %s: %w", slug, err)
	}
	if err := os.MkdirAll(filepath.Dir(repoDir), 0o755); err != nil {
		return "", fmt.Errorf("create repos dir: %w", err)
	}

	if _, err := runGit(ctx, "", "clone", "--", repoURL, repoDir); err != nil {
		return "", fmt.Errorf("clone %s: %w", slug, err)
	}
	return repoDir, nil
}

// ValidateRepoURL enforces an allowlist of safe git clone targets (https://, ssh://,
// git@host:path, or absolute local paths) and rejects flag injection / ext:: / file://.
func ValidateRepoURL(repoURL string) error {
	if repoURL == "" {
		return errors.New("empty repo URL")
	}
	for i := 0; i < len(repoURL); i++ {
		if repoURL[i] < 0x20 || repoURL[i] == 0x7f {
			return errors.New("repo URL contains control characters")
		}
	}
	if strings.HasPrefix(repoURL, "-") {
		return errors.New("repo URL must not start with '-'")
	}
	low := strings.ToLower(repoURL)
	if strings.HasPrefix(low, "ext::") || strings.HasPrefix(low, "file://") {
		return errors.New("unsupported repo URL scheme")
	}
	if strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "ssh://") {
		rest := repoURL[strings.Index(repoURL, "://")+3:]
		if rest == "" || strings.ContainsAny(rest, " \t\r\n") {
			return errors.New("invalid repo URL")
		}
		return nil
	}
	if strings.HasPrefix(repoURL, "git@") {
		rest := repoURL[len("git@"):]
		host, pathPart, ok := strings.Cut(rest, ":")
		if !ok || host == "" || pathPart == "" || strings.HasPrefix(host, "-") || strings.ContainsAny(repoURL, " \t\r\n") {
			return errors.New("invalid git@ repo URL")
		}
		return nil
	}
	if filepath.IsAbs(repoURL) {
		return nil
	}
	return errors.New("unsupported repo URL format")
}

// PrepareWorktree fetches origin, cleans any stale worktree/branch for taskKey/branch,
// and creates a fresh worktree at <APP_HOME>/wt/<projectSlug>/<taskKey> on branch
// starting from origin/<defaultBranch>.
func (m *GitManager) PrepareWorktree(ctx context.Context, repoDir, projectSlug, taskKey, branch, defaultBranch string) (string, error) {
	if repoDir == "" {
		return "", errors.New("repoDir is required")
	}
	if err := validatePathSegment("project slug", projectSlug); err != nil {
		return "", err
	}
	if err := validatePathSegment("task key", taskKey); err != nil {
		return "", err
	}
	if branch == "" {
		return "", errors.New("branch is required")
	}
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	if _, err := runGit(ctx, repoDir, "fetch", "origin"); err != nil {
		return "", fmt.Errorf("fetch origin: %w", err)
	}

	wtDir := m.worktreeDir(projectSlug, taskKey)

	// Clean up any stale worktree or branch left over from an earlier run.
	_, _ = runGit(ctx, repoDir, "worktree", "remove", "--force", wtDir)
	_ = os.RemoveAll(wtDir)
	_, _ = runGit(ctx, repoDir, "worktree", "prune")
	_, _ = runGit(ctx, repoDir, "branch", "-D", branch)

	if err := os.MkdirAll(filepath.Dir(wtDir), 0o755); err != nil {
		return "", fmt.Errorf("create worktree parent dir: %w", err)
	}

	startPoint := "origin/" + defaultBranch
	if _, err := runGit(ctx, repoDir, "worktree", "add", "-b", branch, wtDir, startPoint); err != nil {
		return "", fmt.Errorf("create worktree %s: %w", wtDir, err)
	}
	return wtDir, nil
}

// CleanupWorktree removes the worktree directory and deletes the local branch
// without failing if either is already absent.
func (m *GitManager) CleanupWorktree(ctx context.Context, repoDir, projectSlug, taskKey, branch string) error {
	if err := validatePathSegment("project slug", projectSlug); err != nil {
		return err
	}
	if err := validatePathSegment("task key", taskKey); err != nil {
		return err
	}

	wtDir := m.worktreeDir(projectSlug, taskKey)
	if repoDir != "" && isGitRepo(ctx, repoDir) {
		_, _ = runGit(ctx, repoDir, "worktree", "remove", "--force", wtDir)
		_, _ = runGit(ctx, repoDir, "worktree", "prune")
		if branch != "" {
			_, _ = runGit(ctx, repoDir, "branch", "-D", branch)
		}
	}

	if err := os.RemoveAll(wtDir); err != nil {
		return fmt.Errorf("remove worktree dir %s: %w", wtDir, err)
	}
	return nil
}

func (m *GitManager) worktreeDir(projectSlug, taskKey string) string {
	return filepath.Join(m.homeDir, "wt", projectSlug, taskKey)
}

func validatePathSegment(label, val string) error {
	if val == "" || val == "." || val == ".." || strings.ContainsAny(val, `/\`) {
		return fmt.Errorf("invalid %s: %q", label, val)
	}
	return nil
}

func isGitRepo(ctx context.Context, dir string) bool {
	if _, err := os.Stat(dir); err != nil {
		return false
	}
	_, err := runGit(ctx, dir, "rev-parse", "--git-dir")
	return err == nil
}

func runGit(ctx context.Context, dir string, args ...string) ([]byte, error) {
	fullArgs := args
	if dir != "" {
		fullArgs = append([]string{"-C", dir}, args...)
	}
	cmd := exec.CommandContext(ctx, "git", fullArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errText := strings.TrimSpace(stderr.String())
		if errText != "" {
			return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, errText)
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return stdout.Bytes(), nil
}
