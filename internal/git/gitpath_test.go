package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWorktreeGitPathRootWorktree(t *testing.T) {
	repoRoot := initGitRepo(t)

	got, err := WorktreeGitPath(context.Background(), ExecRunner{}, repoRoot, "ww/task-note.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(repoRoot, ".git", "ww", "task-note.md")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestWorktreeGitPathLinkedWorktree(t *testing.T) {
	repoRoot := initGitRepo(t)
	canonicalRoot := canonicalPath(t, repoRoot)
	worktreePath := filepath.Join(repoRoot, ".worktrees", "feat-a")
	runGit(t, repoRoot, "worktree", "add", "-b", "feat-a", worktreePath, "HEAD")

	got, err := WorktreeGitPath(context.Background(), ExecRunner{}, worktreePath, "ww/task-note.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(canonicalRoot, ".git", "worktrees", "feat-a", "ww", "task-note.md")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()

	repoRoot := t.TempDir()
	runGit(t, repoRoot, "init")
	runGit(t, repoRoot, "config", "user.name", "Test User")
	runGit(t, repoRoot, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGit(t, repoRoot, "add", "README.md")
	runGit(t, repoRoot, "commit", "-m", "init")

	return repoRoot
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func canonicalPath(t *testing.T, path string) string {
	t.Helper()

	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("eval symlinks for %q: %v", path, err)
	}
	return canonical
}
