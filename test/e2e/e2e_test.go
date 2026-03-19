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

func TestCLISelectsWorktreePathByName(t *testing.T) {
	repo := newTestRepo(t)
	alpha := repo.AddWorktree(t, "alpha")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "switch-path", "alpha")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got != alpha {
		t.Fatalf("expected named worktree path %q, got %q", alpha, got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestCLISelectsWorktreePathByUniquePrefix(t *testing.T) {
	repo := newTestRepo(t)
	alpha := repo.AddWorktree(t, "alpha")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "switch-path", "alp")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got != alpha {
		t.Fatalf("expected unique-prefix worktree path %q, got %q", alpha, got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestCLIAmbiguousPrefixReturnsError(t *testing.T) {
	repo := newTestRepo(t)
	runGit(t, repo.Root, "branch", "alpine")
	repo.AddWorktree(t, "alpha")
	repo.AddWorktree(t, "alpine")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "switch-path", "alp")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected ambiguous prefix to fail")
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "ambiguous worktree match") {
		t.Fatalf("expected ambiguous-match message, got %q", stderr.String())
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

func TestCLIListReflectsMRUAfterSwitch(t *testing.T) {
	repo := newTestRepo(t)
	repo.AddWorktree(t, "alpha")
	runGit(t, repo.Root, "branch", "beta")
	repo.AddWorktree(t, "beta")
	bin := buildCLI(t)
	stateHome := t.TempDir()

	switchCmd := exec.CommandContext(context.Background(), bin, "switch-path", "beta")
	switchCmd.Dir = repo.Root
	switchCmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateHome)
	if out, err := switchCmd.CombinedOutput(); err != nil {
		t.Fatalf("switch-path beta failed: %v\n%s", err, out)
	}

	listCmd := exec.CommandContext(context.Background(), bin, "list")
	listCmd.Dir = repo.Root
	listCmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateHome)

	var stdout, stderr bytes.Buffer
	listCmd.Stdout = &stdout
	listCmd.Stderr = &stderr

	if err := listCmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	got := stdout.String()
	if strings.Index(got, "/.worktrees/beta") > strings.Index(got, "/.worktrees/alpha") {
		t.Fatalf("expected beta before alpha after MRU update, got %q", got)
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
