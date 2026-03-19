package app

import (
	"bytes"
	"context"
	"testing"

	"wt/internal/git"
	"wt/internal/worktree"
)

func TestRunHelpPrintsUsageAndExitsZero(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"--help"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("wt [--fzf] [index]")) {
		t.Fatalf("expected usage to mention wt [--fzf] [index], got %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunIndexModePrintsSelectedPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		worktrees: []worktree.Worktree{
			{Index: 1, Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Index: 2, Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
	}

	code := Run(context.Background(), []string{"2"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "/repo/.worktrees/alpha\n" {
		t.Fatalf("expected selected path on stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunNonRepoReturnsExit3(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"1"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{
		err: git.ErrNotGitRepository,
	})

	if code != 3 {
		t.Fatalf("expected exit code 3, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("not a git repository")) {
		t.Fatalf("expected non-repo message, got %q", stderr.String())
	}
}

func TestRunRejectsInvalidIndex(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"abc"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("invalid worktree index")) {
		t.Fatalf("expected invalid index message, got %q", stderr.String())
	}
}

func TestRunRejectsOutOfRangeIndex(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		worktrees: []worktree.Worktree{
			{Index: 1, Path: "/repo", BranchLabel: "main", IsCurrent: true},
		},
	}

	code := Run(context.Background(), []string{"2"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("out of range")) {
		t.Fatalf("expected out-of-range message, got %q", stderr.String())
	}
}

type fakeDeps struct {
	worktrees []worktree.Worktree
	err       error
}

func (f fakeDeps) ListWorktrees(context.Context) ([]worktree.Worktree, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]worktree.Worktree(nil), f.worktrees...), nil
}
