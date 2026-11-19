package worker_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kilip/opus/internal/worker"
)

func gitExec(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v (dir=%s) failed: %v\n%s", args, dir, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

// setupLocalBareRepo creates a bare origin repository with an initial commit on "main"
// and returns (bareRepoDir, seedWorkDir).
func setupLocalBareRepo(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	bareDir := filepath.Join(root, "origin.git")
	seedDir := filepath.Join(root, "seed")

	gitExec(t, "", "init", "--bare", "--initial-branch=main", bareDir)
	gitExec(t, "", "clone", bareDir, seedDir)
	gitExec(t, seedDir, "config", "user.email", "test@opus.local")
	gitExec(t, seedDir, "config", "user.name", "Opus Test")
	gitExec(t, seedDir, "checkout", "-B", "main")

	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("hello opus\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	gitExec(t, seedDir, "add", "README.md")
	gitExec(t, seedDir, "commit", "-m", "initial commit")
	gitExec(t, seedDir, "push", "-u", "origin", "main")

	return bareDir, seedDir
}

func TestGitManager_EnsureRepo(t *testing.T) {
	ctx := context.Background()
	appHome := t.TempDir()
	gm := worker.NewGitManager(appHome)

	bareDir, seedDir := setupLocalBareRepo(t)

	// 1. Clones into <APP_HOME>/repos/<slug> when mappedPath is empty.
	repoDir, err := gm.EnsureRepo(ctx, "my-project", bareDir, "")
	if err != nil {
		t.Fatalf("EnsureRepo clone failed: %v", err)
	}
	expectedDir := filepath.Join(appHome, "repos", "my-project")
	if repoDir != expectedDir {
		t.Fatalf("expected repoDir %q, got %q", expectedDir, repoDir)
	}
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err != nil {
		t.Fatalf("expected .git in cloned repo: %v", err)
	}

	// 2. Second call skips clone (idempotent).
	repoDir2, err := gm.EnsureRepo(ctx, "my-project", bareDir, "")
	if err != nil {
		t.Fatalf("EnsureRepo second call failed: %v", err)
	}
	if repoDir2 != expectedDir {
		t.Fatalf("expected repoDir %q, got %q", expectedDir, repoDir2)
	}

	// 3. Valid mappedPath is returned directly without cloning.
	mappedDir, err := gm.EnsureRepo(ctx, "mapped-proj", "", seedDir)
	if err != nil {
		t.Fatalf("EnsureRepo mappedPath failed: %v", err)
	}
	if mappedDir != seedDir {
		t.Fatalf("expected mappedDir %q, got %q", seedDir, mappedDir)
	}
	if _, err := os.Stat(filepath.Join(appHome, "repos", "mapped-proj")); !os.IsNotExist(err) {
		t.Fatalf("expected no clone under repos/ when mappedPath is set")
	}

	// 4. Missing mappedPath returns a clear error instead of cloning.
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := gm.EnsureRepo(ctx, "missing-proj", bareDir, missing); err == nil {
		t.Fatal("expected error for missing mappedPath")
	}

	// 5. Non-git directory mappedPath returns a clear error.
	notGit := t.TempDir()
	if _, err := gm.EnsureRepo(ctx, "notgit-proj", bareDir, notGit); err == nil {
		t.Fatal("expected error for non-git mappedPath")
	}
}

func TestGitManager_PrepareAndCleanupWorktree(t *testing.T) {
	ctx := context.Background()
	appHome := t.TempDir()
	gm := worker.NewGitManager(appHome)

	bareDir, seedDir := setupLocalBareRepo(t)
	repoDir, err := gm.EnsureRepo(ctx, "demo-proj", bareDir, "")
	if err != nil {
		t.Fatalf("EnsureRepo failed: %v", err)
	}

	// Push a new commit to origin/main after initial clone so PrepareWorktree must fetch.
	if err := os.WriteFile(filepath.Join(seedDir, "NEW.txt"), []byte("from origin\n"), 0o644); err != nil {
		t.Fatalf("write NEW.txt: %v", err)
	}
	gitExec(t, seedDir, "add", "NEW.txt")
	gitExec(t, seedDir, "commit", "-m", "second commit")
	gitExec(t, seedDir, "push", "origin", "main")

	// Pre-create a stale worktree and stale branch to verify cleanup before creation.
	wtDir, err := gm.PrepareWorktree(ctx, repoDir, "demo-proj", "OPUS-1", "opus/OPUS-1", "main")
	if err != nil {
		t.Fatalf("initial PrepareWorktree failed: %v", err)
	}
	// Modify a file inside stale worktree to ensure --force removal works.
	if err := os.WriteFile(filepath.Join(wtDir, "STALE.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	// Re-prepare same taskKey/branch: should clean stale worktree/branch and recreate cleanly.
	wtDir2, err := gm.PrepareWorktree(ctx, repoDir, "demo-proj", "OPUS-1", "opus/OPUS-1", "main")
	if err != nil {
		t.Fatalf("re-run PrepareWorktree failed: %v", err)
	}
	expectedWT := filepath.Join(appHome, "wt", "demo-proj", "OPUS-1")
	if wtDir2 != expectedWT {
		t.Fatalf("expected wtDir %q, got %q", expectedWT, wtDir2)
	}
	if _, err := os.Stat(filepath.Join(wtDir2, "STALE.txt")); !os.IsNotExist(err) {
		t.Fatal("expected stale worktree file to be gone")
	}
	data, err := os.ReadFile(filepath.Join(wtDir2, "NEW.txt"))
	if err != nil || string(data) != "from origin\n" {
		t.Fatalf("expected fetched NEW.txt in worktree, got %q (err=%v)", string(data), err)
	}
	curBranch := gitExec(t, wtDir2, "rev-parse", "--abbrev-ref", "HEAD")
	if curBranch != "opus/OPUS-1" {
		t.Fatalf("expected branch opus/OPUS-1, got %q", curBranch)
	}

	// CleanupWorktree removes directory and deletes local branch.
	if err := gm.CleanupWorktree(ctx, repoDir, "demo-proj", "OPUS-1", "opus/OPUS-1"); err != nil {
		t.Fatalf("CleanupWorktree failed: %v", err)
	}
	if _, err := os.Stat(wtDir2); !os.IsNotExist(err) {
		t.Fatalf("expected worktree dir %s to be removed", wtDir2)
	}
	branches := gitExec(t, repoDir, "branch", "--list", "opus/OPUS-1")
	if branches != "" {
		t.Fatalf("expected branch opus/OPUS-1 to be deleted, got %q", branches)
	}

	// Idempotent cleanup when already absent.
	if err := gm.CleanupWorktree(ctx, repoDir, "demo-proj", "OPUS-1", "opus/OPUS-1"); err != nil {
		t.Fatalf("second CleanupWorktree should not fail: %v", err)
	}
}

func TestGitManager_MappedPath_LeavesUserWorkingTreeUntouched(t *testing.T) {
	ctx := context.Background()
	appHome := t.TempDir()
	gm := worker.NewGitManager(appHome)

	_, userRepo := setupLocalBareRepo(t)

	// User checks out a feature branch, stages a file, edits a tracked file, and adds an untracked file.
	gitExec(t, userRepo, "checkout", "-b", "user/wip-branch")
	if err := os.WriteFile(filepath.Join(userRepo, "staged.txt"), []byte("staged content\n"), 0o644); err != nil {
		t.Fatalf("write staged.txt: %v", err)
	}
	gitExec(t, userRepo, "add", "staged.txt")
	if err := os.WriteFile(filepath.Join(userRepo, "README.md"), []byte("dirty unstaged edit\n"), 0o644); err != nil {
		t.Fatalf("edit README.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(userRepo, "untracked.txt"), []byte("untracked\n"), 0o644); err != nil {
		t.Fatalf("write untracked.txt: %v", err)
	}

	headBefore := gitExec(t, userRepo, "rev-parse", "HEAD")
	branchBefore := gitExec(t, userRepo, "rev-parse", "--abbrev-ref", "HEAD")
	statusBefore := gitExec(t, userRepo, "status", "--porcelain")
	indexBefore := gitExec(t, userRepo, "diff", "--cached")

	repoDir, err := gm.EnsureRepo(ctx, "mapped-proj", "", userRepo)
	if err != nil {
		t.Fatalf("EnsureRepo mappedPath failed: %v", err)
	}
	wtDir, err := gm.PrepareWorktree(ctx, repoDir, "mapped-proj", "OPUS-42", "opus/OPUS-42", "main")
	if err != nil {
		t.Fatalf("PrepareWorktree on mappedPath failed: %v", err)
	}
	if err := gm.CleanupWorktree(ctx, repoDir, "mapped-proj", "OPUS-42", "opus/OPUS-42"); err != nil {
		t.Fatalf("CleanupWorktree on mappedPath failed: %v", err)
	}
	_ = wtDir

	headAfter := gitExec(t, userRepo, "rev-parse", "HEAD")
	branchAfter := gitExec(t, userRepo, "rev-parse", "--abbrev-ref", "HEAD")
	statusAfter := gitExec(t, userRepo, "status", "--porcelain")
	indexAfter := gitExec(t, userRepo, "diff", "--cached")

	if headBefore != headAfter || branchBefore != branchAfter {
		t.Fatalf("HEAD/branch changed: before=(%s,%s) after=(%s,%s)", headBefore, branchBefore, headAfter, branchAfter)
	}
	if statusBefore != statusAfter {
		t.Fatalf("working tree status changed:\nbefore:\n%s\nafter:\n%s", statusBefore, statusAfter)
	}
	if indexBefore != indexAfter {
		t.Fatalf("staged index changed:\nbefore:\n%s\nafter:\n%s", indexBefore, indexAfter)
	}
	readmeData, err := os.ReadFile(filepath.Join(userRepo, "README.md"))
	if err != nil || string(readmeData) != "dirty unstaged edit\n" {
		t.Fatalf("expected unstaged edit preserved, got %q (err=%v)", string(readmeData), err)
	}
}

func TestValidateRepoURL(t *testing.T) {
	cases := []struct {
		url     string
		wantErr bool
	}{
		{"https://github.com/kilip/opus.git", false},
		{"ssh://git@github.com/kilip/opus.git", false},
		{"git@github.com:kilip/opus.git", false},
		{"/tmp/local-repo", false},
		{"", true},
		{"-u./payload", true},
		{"--upload-pack=evil", true},
		{"ext::sh -c evil", true},
		{"EXT::sh -c evil", true},
		{"file:///etc/passwd", true},
		{"FILE:///etc/passwd", true},
		{"https://github.com/a\nb", true},
	}
	for _, tc := range cases {
		err := worker.ValidateRepoURL(tc.url)
		if (err != nil) != tc.wantErr {
			t.Fatalf("ValidateRepoURL(%q) err=%v, wantErr=%v", tc.url, err, tc.wantErr)
		}
	}
}
