package ui

import (
	"testing"

	"ww/internal/worktree"
)

func TestStatusTagsShowsCurrentForCurrentWorktree(t *testing.T) {
	got := StatusTags(worktree.Worktree{IsCurrent: true})
	if len(got) != 1 || got[0] != "[CURRENT]" {
		t.Fatalf("expected [CURRENT], got %v", got)
	}
}

func TestStatusTagsShowsMergedForMergedWorktree(t *testing.T) {
	got := StatusTags(worktree.Worktree{IsMerged: true})
	if len(got) != 1 || got[0] != "[MERGED]" {
		t.Fatalf("expected [MERGED], got %v", got)
	}
}

func TestStatusTagsShowsCurrentAndMergedTogether(t *testing.T) {
	got := StatusTags(worktree.Worktree{IsCurrent: true, IsMerged: true})
	if len(got) != 2 || got[0] != "[CURRENT]" || got[1] != "[MERGED]" {
		t.Fatalf("expected [CURRENT] [MERGED], got %v", got)
	}
}

func TestStatusTagsOmitsDirty(t *testing.T) {
	got := StatusTags(worktree.Worktree{IsDirty: true})
	if len(got) != 0 {
		t.Fatalf("expected no tags for dirty-only worktree, got %v", got)
	}
}

func TestStatusTagsIsBlankForCleanNonCurrentWorktree(t *testing.T) {
	got := StatusTags(worktree.Worktree{})
	if len(got) != 0 {
		t.Fatalf("expected blank status tags, got %v", got)
	}
}

func TestFormatFileChangesShowsStagedUnstagedUntracked(t *testing.T) {
	got := FormatFileChanges(3, 1, 2)
	stripped := StripAnsi(got)
	if stripped != "+3 ~1 ?2" {
		t.Fatalf("expected '+3 ~1 ?2', got %q", stripped)
	}
}

func TestFormatFileChangesOmitsZeroCounts(t *testing.T) {
	got := StripAnsi(FormatFileChanges(0, 1, 0))
	if got != "~1" {
		t.Fatalf("expected '~1', got %q", got)
	}
}

func TestFormatFileChangesReturnsEmptyWhenAllZero(t *testing.T) {
	got := FormatFileChanges(0, 0, 0)
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestFormatAheadBehindShowsBoth(t *testing.T) {
	got := StripAnsi(FormatAheadBehind(5, 3))
	if got != "↑5 ↓3" {
		t.Fatalf("expected '↑5 ↓3', got %q", got)
	}
}

func TestFormatAheadBehindOmitsZeroDirection(t *testing.T) {
	got := StripAnsi(FormatAheadBehind(5, 0))
	if got != "↑5" {
		t.Fatalf("expected '↑5', got %q", got)
	}

	got = StripAnsi(FormatAheadBehind(0, 3))
	if got != "↓3" {
		t.Fatalf("expected '↓3', got %q", got)
	}
}

func TestFormatAheadBehindReturnsEmptyWhenBothZero(t *testing.T) {
	got := FormatAheadBehind(0, 0)
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
