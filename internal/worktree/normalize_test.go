package worktree

import "testing"

func TestNormalizeOrdersCurrentFirstThenMRU(t *testing.T) {
	items := []Worktree{
		{Path: "/repo/.worktrees/beta", BranchLabel: "beta", LastUsedAt: 20},
		{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", LastUsedAt: 10},
		{Path: "/repo", BranchLabel: "main", IsCurrent: true},
	}

	got := Normalize(items)

	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if !got[0].IsCurrent || got[0].Path != "/repo" {
		t.Fatalf("expected current worktree first, got %#v", got[0])
	}
	if got[0].Index != 1 || got[1].Index != 2 || got[2].Index != 3 {
		t.Fatalf("expected 1-based sequential indexes, got %#v", got)
	}
	if got[1].BranchLabel != "beta" || got[2].BranchLabel != "alpha" {
		t.Fatalf("expected MRU ordering, got %#v", got)
	}
}

func TestNormalizeFallsBackToDeterministicNameOrderingWhenMRUMissing(t *testing.T) {
	items := []Worktree{
		{Path: "/repo/.worktrees/zeta", BranchLabel: "zeta", LastUsedAt: 50},
		{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		{Path: "/repo/.worktrees/beta", BranchLabel: "beta"},
		{Path: "/repo", BranchLabel: "main", IsCurrent: true},
	}

	got := Normalize(items)

	if got[1].BranchLabel != "alpha" || got[2].BranchLabel != "beta" || got[3].BranchLabel != "zeta" {
		t.Fatalf("expected deterministic name fallback, got %#v", got)
	}
}
