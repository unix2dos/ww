package git

import (
	"context"
	"testing"
)

func TestFileChangeCountsParsesStagedUnstagedUntracked(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string]string{
			key("git", "-C", "/repo", "status", "--porcelain", "--", ".", ":(exclude).worktrees"): "A  new.go\n M modified.go\n?? scratch.txt\n?? temp.log\nM  staged.go\n",
		},
	}

	staged, unstaged, untracked, err := FileChangeCounts(context.Background(), runner, "/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if staged != 2 {
		t.Fatalf("expected 2 staged, got %d", staged)
	}
	if unstaged != 1 {
		t.Fatalf("expected 1 unstaged, got %d", unstaged)
	}
	if untracked != 2 {
		t.Fatalf("expected 2 untracked, got %d", untracked)
	}
}

func TestFileChangeCountsReturnsZerosForCleanWorktree(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string]string{
			key("git", "-C", "/repo", "status", "--porcelain", "--", ".", ":(exclude).worktrees"): "",
		},
	}

	staged, unstaged, untracked, err := FileChangeCounts(context.Background(), runner, "/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if staged != 0 || unstaged != 0 || untracked != 0 {
		t.Fatalf("expected all zeros, got %d %d %d", staged, unstaged, untracked)
	}
}

func TestAheadBehindParsesRevListOutput(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string]string{
			key("git", "-C", "/repo", "rev-list", "--left-right", "--count", "feat-a...main"): "5\t3\n",
		},
	}

	ahead, behind, err := AheadBehind(context.Background(), runner, "/repo", "feat-a", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ahead != 5 {
		t.Fatalf("expected ahead 5, got %d", ahead)
	}
	if behind != 3 {
		t.Fatalf("expected behind 3, got %d", behind)
	}
}

func TestAheadBehindReturnsZerosWhenEqual(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string]string{
			key("git", "-C", "/repo", "rev-list", "--left-right", "--count", "main...main"): "0\t0\n",
		},
	}

	ahead, behind, err := AheadBehind(context.Background(), runner, "/repo", "main", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ahead != 0 || behind != 0 {
		t.Fatalf("expected zeros, got %d %d", ahead, behind)
	}
}

func TestBranchMergedIntoBaseReturnsTrueWhenMerged(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string]string{
			key("git", "-C", "/repo", "branch", "--format=%(refname:short)", "--merged", "main"): "feat-a\nfix-b\n",
		},
	}

	merged, err := BranchMergedIntoBase(context.Background(), runner, "/repo", "feat-a", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !merged {
		t.Fatal("expected merged=true")
	}
}

func TestBranchMergedIntoBaseReturnsFalseWhenNotMerged(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string]string{
			key("git", "-C", "/repo", "branch", "--format=%(refname:short)", "--merged", "main"): "fix-b\n",
		},
	}

	merged, err := BranchMergedIntoBase(context.Background(), runner, "/repo", "feat-a", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if merged {
		t.Fatal("expected merged=false")
	}
}
