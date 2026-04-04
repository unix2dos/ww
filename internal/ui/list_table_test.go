package ui

import (
	"strings"
	"testing"

	"ww/internal/worktree"
)

func TestFormatListTableUsesUnicodeBoxBorders(t *testing.T) {
	got := FormatListTable([]ListTableEntry{
		{Worktree: worktree.Worktree{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true, Staged: 2, Unstaged: 1}},
		{Worktree: worktree.Worktree{Index: 2, BranchLabel: "feat-a", Path: "/repo/.worktrees/feat-a", IsMerged: true}},
	})

	stripped := StripAnsi(got)
	for _, fragment := range []string{
		"┌",
		"┬",
		"│ INDEX",
		"│ STATUS",
		"│ AHEAD/BEHIND",
		"│ CHANGES",
		"├",
		"┼",
		"└",
		"┴",
		"│ 1",
		"[CURRENT]",
		"[MERGED]",
	} {
		if !strings.Contains(stripped, fragment) {
			t.Fatalf("expected %q in table output, got %q", fragment, stripped)
		}
	}
}

func TestFormatListTableWrapsLongPathInsidePathCell(t *testing.T) {
	got := FormatListTable([]ListTableEntry{
		{
			Worktree: worktree.Worktree{
				Index:       2,
				BranchLabel: "codex/current-dirty-status",
				Path:        "/Users/liuwei/workspace/ww/.worktrees/current-dirty-status/very/long/path/for/wrapping",
				Unstaged:    1,
				IsDirty:     true,
			},
		},
	})

	stripped := StripAnsi(got)
	if !strings.Contains(stripped, "│ 2") {
		t.Fatalf("expected first row for wrapped item, got %q", stripped)
	}
	if !strings.Contains(stripped, "current-dirty-status") || !strings.Contains(stripped, "very/long/path") {
		t.Fatalf("expected full path content across wrapped lines, got %q", stripped)
	}
}
