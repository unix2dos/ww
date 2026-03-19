package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLISelectsSecondWorktreePath(t *testing.T) {
	repo := newTestRepo(t)
	second := repo.AddWorktree(t, "alpha")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "switch-path", "2")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got != second {
		t.Fatalf("expected second worktree path %q, got %q", second, got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestCLIListsWorktrees(t *testing.T) {
	repo := newTestRepo(t)
	repo.AddWorktree(t, "alpha")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "list")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "[1]") || !strings.Contains(stdout.String(), "/.worktrees/alpha") {
		t.Fatalf("expected human-readable list output, got %q", stdout.String())
	}
}

func TestCLICreatesNewWorktreePath(t *testing.T) {
	repo := newTestRepo(t)
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "new-path", "beta")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	want := filepath.Join(repo.Root, ".worktrees", "beta")
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Fatalf("expected new worktree path %q, got %q", want, got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected worktree path to exist: %v", err)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return root
}
