package ui

import (
	"bytes"
	"strings"
	"testing"

	"ww/internal/worktree"
)

func TestRenderMenuShowsEnhancedFormat(t *testing.T) {
	var buf bytes.Buffer

	items := []worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true, Staged: 3, Unstaged: 1, Untracked: 2, Ahead: 2, IsDirty: true},
		{Index: 2, BranchLabel: "feat-a", Path: "/repo/.worktrees/feat-a", Staged: 1, Untracked: 1, Ahead: 5, Behind: 3, IsDirty: true},
		{Index: 3, BranchLabel: "fix/typo", Path: "/repo/.worktrees/fix-typo", IsMerged: true},
	}

	RenderMenu(&buf, items)
	got := buf.String()

	// Current worktree has ★ marker
	if !strings.Contains(got, "★") {
		t.Fatalf("expected ★ marker for current worktree, got %q", got)
	}

	// Prompt uses Select [1-N]
	if !strings.Contains(got, "Select [1-3]>") {
		t.Fatalf("expected 'Select [1-3]>' prompt, got %q", got)
	}
}

func TestRenderMenuShowsSummaryWithSafeToRemove(t *testing.T) {
	var buf bytes.Buffer

	items := []worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		{Index: 2, BranchLabel: "fix/typo", Path: "/wt/fix-typo", IsMerged: true},
		{Index: 3, BranchLabel: "feat-a", Path: "/wt/feat-a", Ahead: 5},
	}

	RenderMenu(&buf, items)
	got := buf.String()

	if !strings.Contains(got, "3 worktrees") {
		t.Fatalf("expected '3 worktrees' in summary, got %q", got)
	}
	if !strings.Contains(got, "safe to remove") {
		t.Fatalf("expected 'safe to remove' in summary, got %q", got)
	}
	if !strings.Contains(got, "ww rm 2") {
		t.Fatalf("expected 'ww rm 2' hint in summary, got %q", got)
	}
}

func TestRenderMenuSummaryOmitsSafeToRemoveWhenNone(t *testing.T) {
	var buf bytes.Buffer

	items := []worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		{Index: 2, BranchLabel: "feat-a", Path: "/wt/feat-a", Ahead: 5},
	}

	RenderMenu(&buf, items)
	got := buf.String()

	if !strings.Contains(got, "2 worktrees") {
		t.Fatalf("expected '2 worktrees' in summary, got %q", got)
	}
	if strings.Contains(got, "safe to remove") {
		t.Fatalf("did not expect 'safe to remove', got %q", got)
	}
}

func TestRenderMenuSingularWorktree(t *testing.T) {
	var buf bytes.Buffer

	items := []worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
	}

	RenderMenu(&buf, items)
	got := buf.String()

	if !strings.Contains(got, "1 worktree") {
		t.Fatalf("expected '1 worktree' (singular), got %q", got)
	}
	// Check it's not "worktrees" (plural)
	if strings.Contains(got, "1 worktrees") {
		t.Fatalf("expected singular 'worktree', got %q", got)
	}
}

func TestRenderMenuSummaryShowsOnlyFirstSafeIndex(t *testing.T) {
	var buf bytes.Buffer

	items := []worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		{Index: 2, BranchLabel: "fix/typo", Path: "/wt/fix-typo", IsMerged: true},
		{Index: 3, BranchLabel: "fix/other", Path: "/wt/fix-other", IsMerged: true},
	}

	RenderMenu(&buf, items)
	got := buf.String()

	if !strings.Contains(got, "2 safe to remove") {
		t.Fatalf("expected '2 safe to remove' in summary, got %q", got)
	}
	if !strings.Contains(got, "ww rm 2)") {
		t.Fatalf("expected hint with only first index 'ww rm 2)', got %q", got)
	}
	if strings.Contains(got, "ww rm 2 3") {
		t.Fatalf("should not show all indices, got %q", got)
	}
}

func TestRenderMenuMergedWithDirtyNotSafeToRemove(t *testing.T) {
	var buf bytes.Buffer

	items := []worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		{Index: 2, BranchLabel: "fix/typo", Path: "/wt/fix-typo", IsMerged: true, IsDirty: true, Unstaged: 2},
	}

	RenderMenu(&buf, items)
	got := buf.String()

	if strings.Contains(got, "safe to remove") {
		t.Fatalf("merged+dirty should not be safe to remove, got %q", got)
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
