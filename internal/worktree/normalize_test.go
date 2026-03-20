package worktree

import "testing"

func TestNormalizeOrdersByCreatedAtAndKeepsCurrentInSortedPosition(t *testing.T) {
	items := []Worktree{
		{Path: "/repo/.worktrees/beta", BranchLabel: "beta", CreatedAt: 30},
		{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", CreatedAt: 20},
		{Path: "/repo", BranchLabel: "main", IsCurrent: true, CreatedAt: 10},
	}

	got := Normalize(items)

	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[0].BranchLabel != "main" || got[1].BranchLabel != "alpha" || got[2].BranchLabel != "beta" {
		t.Fatalf("expected created-at ordering, got %#v", got)
	}
	if !got[0].IsCurrent || got[0].Path != "/repo" {
		t.Fatalf("expected current worktree to keep created-at position, got %#v", got[0])
	}
	if got[0].Index != 1 || got[1].Index != 2 || got[2].Index != 3 {
		t.Fatalf("expected 1-based sequential indexes, got %#v", got)
	}
}

func TestNormalizeIgnoresMRUWhenOrdering(t *testing.T) {
	items := []Worktree{
		{Path: "/repo/.worktrees/zeta", BranchLabel: "zeta", LastUsedAt: 50, CreatedAt: 30},
		{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", CreatedAt: 10},
		{Path: "/repo/.worktrees/beta", BranchLabel: "beta", CreatedAt: 20},
	}

	got := Normalize(items)

	if got[0].BranchLabel != "alpha" || got[1].BranchLabel != "beta" || got[2].BranchLabel != "zeta" {
		t.Fatalf("expected MRU to be ignored for ordering, got %#v", got)
	}
}

func TestNormalizeFallsBackToDeterministicNameOrderingAmongTies(t *testing.T) {
	items := []Worktree{
		{Path: "/repo/.worktrees/beta", BranchLabel: "beta"},
		{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
	}

	got := Normalize(items)

	if got[0].BranchLabel != "alpha" || got[1].BranchLabel != "beta" {
		t.Fatalf("expected deterministic name ordering among missing MRU items, got %#v", got)
	}
}
