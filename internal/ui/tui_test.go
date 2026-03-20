package ui

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"ww/internal/worktree"
)

func TestRenderTUIShowsActiveStatusAndActiveRow(t *testing.T) {
	var buf bytes.Buffer

	RenderTUI(&buf, []worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		{Index: 2, BranchLabel: "feat-a", Path: "/repo/.worktrees/feat-a"},
	}, 1)

	got := buf.String()
	if !strings.Contains(got, "  [1] ACTIVE main /repo") {
		t.Fatalf("expected current row, got %q", got)
	}
	if !strings.Contains(got, "* [2]        feat-a /repo/.worktrees/feat-a") {
		t.Fatalf("expected active row, got %q", got)
	}
	if !strings.Contains(got, "Enter to confirm") {
		t.Fatalf("expected tui instructions, got %q", got)
	}
}

func TestSelectWorktreeWithTUIArrowDownThenEnterReturnsSelectedWorktree(t *testing.T) {
	var out bytes.Buffer

	got, err := SelectWorktreeWithTUI(
		strings.NewReader("\x1b[B\r"),
		&out,
		[]worktree.Worktree{
			{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
			{Index: 2, BranchLabel: "feat-a", Path: "/repo/.worktrees/feat-a"},
		},
		nopRawMode{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Path != "/repo/.worktrees/feat-a" {
		t.Fatalf("expected second worktree, got %#v", got)
	}
	if !strings.Contains(out.String(), "* [2]        feat-a /repo/.worktrees/feat-a") {
		t.Fatalf("expected moved selection to render, got %q", out.String())
	}
}

func TestSelectWorktreeWithTUIArrowUpWrapsToLastWorktree(t *testing.T) {
	got, err := SelectWorktreeWithTUI(
		strings.NewReader("\x1b[A\r"),
		io.Discard,
		[]worktree.Worktree{
			{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
			{Index: 2, BranchLabel: "feat-a", Path: "/repo/.worktrees/feat-a"},
			{Index: 3, BranchLabel: "feat-b", Path: "/repo/.worktrees/feat-b"},
		},
		nopRawMode{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Path != "/repo/.worktrees/feat-b" {
		t.Fatalf("expected wrap to last worktree, got %#v", got)
	}
}

func TestSelectWorktreeWithTUIEnterDefaultsToCurrentWorktree(t *testing.T) {
	got, err := SelectWorktreeWithTUI(
		strings.NewReader("\r"),
		io.Discard,
		[]worktree.Worktree{
			{Index: 1, BranchLabel: "alpha", Path: "/repo/.worktrees/alpha"},
			{Index: 2, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		},
		nopRawMode{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Path != "/repo" {
		t.Fatalf("expected current worktree by default, got %#v", got)
	}
}

func TestSelectWorktreeWithTUIEscapeCancels(t *testing.T) {
	_, err := SelectWorktreeWithTUI(
		strings.NewReader("\x1b"),
		io.Discard,
		[]worktree.Worktree{
			{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		},
		nopRawMode{},
	)
	if !errors.Is(err, ErrSelectionCanceled) {
		t.Fatalf("expected ErrSelectionCanceled, got %v", err)
	}
}

func TestSelectWorktreeWithTUICtrlCCancels(t *testing.T) {
	_, err := SelectWorktreeWithTUI(
		strings.NewReader("\x03"),
		io.Discard,
		[]worktree.Worktree{
			{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		},
		nopRawMode{},
	)
	if !errors.Is(err, ErrSelectionCanceled) {
		t.Fatalf("expected ErrSelectionCanceled, got %v", err)
	}
}

type nopRawMode struct{}

func (nopRawMode) Prepare(io.Reader) (func(), error) {
	return func() {}, nil
}
