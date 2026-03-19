package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"wt/internal/worktree"
)

func TestFormatFzfCandidatesIncludesIndexMarkerBranchAndPath(t *testing.T) {
	got := string(formatFzfCandidates([]worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		{Index: 2, BranchLabel: "feat-a", Path: "/repo/.worktrees/feat-a"},
	}))

	if !strings.Contains(got, "1\t*\tmain\t/repo") {
		t.Fatalf("expected current candidate, got %q", got)
	}
	if !strings.Contains(got, "2\t \tfeat-a\t/repo/.worktrees/feat-a") {
		t.Fatalf("expected non-current candidate, got %q", got)
	}
}

func TestSelectWorktreeWithFzfReturnsSelectedWorktree(t *testing.T) {
	runner := fakeFzfRunner{
		lookPath: "/usr/bin/fzf",
		stdout:   []byte("2\t \tfeat-a\t/repo/.worktrees/feat-a\n"),
	}

	got, err := SelectWorktreeWithFzf(context.Background(), []worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		{Index: 2, BranchLabel: "feat-a", Path: "/repo/.worktrees/feat-a"},
	}, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Path != "/repo/.worktrees/feat-a" {
		t.Fatalf("expected selected worktree, got %#v", got)
	}
}

func TestSelectWorktreeWithFzfReturnsErrFzfNotInstalled(t *testing.T) {
	_, err := SelectWorktreeWithFzf(context.Background(), nil, fakeFzfRunner{
		lookPathErr: errors.New("missing"),
	})
	if !errors.Is(err, ErrFzfNotInstalled) {
		t.Fatalf("expected ErrFzfNotInstalled, got %v", err)
	}
}

func TestSelectWorktreeWithFzfReturnsErrSelectionCanceled(t *testing.T) {
	_, err := SelectWorktreeWithFzf(context.Background(), []worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
	}, fakeFzfRunner{
		lookPath: "/usr/bin/fzf",
		err:      exitError{code: 130},
	})
	if !errors.Is(err, ErrSelectionCanceled) {
		t.Fatalf("expected ErrSelectionCanceled, got %v", err)
	}
}

type fakeFzfRunner struct {
	lookPath    string
	lookPathErr error
	stdout      []byte
	stderr      []byte
	err         error
}

func (f fakeFzfRunner) LookPath(string) (string, error) {
	return f.lookPath, f.lookPathErr
}

func (f fakeFzfRunner) Run(_ context.Context, _ string, stdin []byte, _ ...string) ([]byte, []byte, error) {
	return append([]byte(nil), f.stdout...), append([]byte(nil), f.stderr...), f.err
}

type exitError struct {
	code int
}

func (e exitError) Error() string {
	return "exit status"
}

func (e exitError) ExitCode() int {
	return e.code
}
