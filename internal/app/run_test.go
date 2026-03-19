package app

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"wt/internal/git"
	"wt/internal/ui"
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

func TestRunRejectsExtraArgsAfterIndex(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		worktrees: []worktree.Worktree{
			{Index: 1, Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Index: 2, Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
	}

	code := Run(context.Background(), []string{"2", "junk"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("unexpected extra arguments")) {
		t.Fatalf("expected extra-args message, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
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

func TestRunFzfModePrintsSelectedPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		worktrees: []worktree.Worktree{
			{Index: 1, Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Index: 2, Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
		fzfSelected: worktree.Worktree{Index: 2, Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
	}

	code := Run(context.Background(), []string{"--fzf"}, bytes.NewReader(nil), stdout, stderr, deps)

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

func TestRunFzfModeReturnsExit3WhenFzfMissing(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		worktrees: []worktree.Worktree{
			{Index: 1, Path: "/repo", BranchLabel: "main", IsCurrent: true},
		},
		fzfErr: ui.ErrFzfNotInstalled,
	}

	code := Run(context.Background(), []string{"--fzf"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 3 {
		t.Fatalf("expected exit code 3, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("fzf is not installed")) {
		t.Fatalf("expected missing fzf message, got %q", stderr.String())
	}
}

func TestRunFzfModeReturns130WhenCanceled(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		worktrees: []worktree.Worktree{
			{Index: 1, Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Index: 2, Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
		fzfErr: ui.ErrSelectionCanceled,
	}

	code := Run(context.Background(), []string{"--fzf"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 130 {
		t.Fatalf("expected exit code 130, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
}

func TestRunRejectsExtraArgsAfterFzf(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"--fzf", "junk"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("unexpected extra arguments")) {
		t.Fatalf("expected extra-args message, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
}

func TestRunInteractiveSelectionWritesMenuToStderrAndPathToStdout(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		worktrees: []worktree.Worktree{
			{Index: 1, Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Index: 2, Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
	}

	code := Run(context.Background(), nil, strings.NewReader("2\n"), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "/repo/.worktrees/alpha\n" {
		t.Fatalf("expected selected path on stdout, got %q", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("[1] * main /repo")) {
		t.Fatalf("expected menu on stderr, got %q", stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Select a worktree")) {
		t.Fatalf("expected prompt on stderr, got %q", stderr.String())
	}
}

func TestRunInteractiveSelectionReturnsNonZeroOnEOFWithoutSelection(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		worktrees: []worktree.Worktree{
			{Index: 1, Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Index: 2, Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
	}

	code := Run(context.Background(), nil, strings.NewReader(""), stdout, stderr, deps)

	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
}

type fakeDeps struct {
	worktrees   []worktree.Worktree
	err         error
	fzfSelected worktree.Worktree
	fzfErr      error
}

func (f fakeDeps) ListWorktrees(context.Context) ([]worktree.Worktree, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]worktree.Worktree(nil), f.worktrees...), nil
}

func (f fakeDeps) SelectWorktreeWithFzf(context.Context, []worktree.Worktree) (worktree.Worktree, error) {
	if f.fzfErr != nil {
		return worktree.Worktree{}, f.fzfErr
	}
	if f.fzfSelected.Path != "" || f.fzfSelected.Index != 0 {
		return f.fzfSelected, nil
	}
	if len(f.worktrees) > 0 {
		return f.worktrees[0], nil
	}
	return worktree.Worktree{}, nil
}
