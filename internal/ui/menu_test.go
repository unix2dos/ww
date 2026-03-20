package ui

import (
	"bytes"
	"strings"
	"testing"

	"ww/internal/worktree"
)

func TestRenderMenuIncludesIndexBranchPathAndActiveStatus(t *testing.T) {
	var buf bytes.Buffer

	RenderMenu(&buf, []worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		{Index: 2, BranchLabel: "feat-a", Path: "/repo/.worktrees/feat-a"},
	})

	got := buf.String()
	if !strings.Contains(got, "[1] ACTIVE main /repo") {
		t.Fatalf("expected current row, got %q", got)
	}
	if !strings.Contains(got, "[2]        feat-a /repo/.worktrees/feat-a") {
		t.Fatalf("expected non-current row, got %q", got)
	}
	if !strings.Contains(got, "Select a worktree [number]: ") {
		t.Fatalf("expected prompt, got %q", got)
	}
}

func TestReadSelectionRetriesAfterInvalidInput(t *testing.T) {
	var stderr bytes.Buffer

	index, err := ReadSelection(strings.NewReader("abc\n2\n"), &stderr, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if index != 2 {
		t.Fatalf("expected selection 2, got %d", index)
	}
	if !strings.Contains(stderr.String(), "invalid worktree selection") {
		t.Fatalf("expected invalid selection message, got %q", stderr.String())
	}
}
