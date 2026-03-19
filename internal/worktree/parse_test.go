package worktree

import (
	"strings"
	"testing"
)

func TestParsePorcelainZ(t *testing.T) {
	raw := strings.Join([]string{
		"worktree /repo",
		"HEAD 1111111",
		"branch refs/heads/main",
		"",
		"worktree /repo/.worktrees/feat-a",
		"HEAD 2222222",
		"branch refs/heads/feat-a",
		"",
	}, "\x00")

	got, err := ParsePorcelainZ(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[1].BranchRef != "refs/heads/feat-a" {
		t.Fatalf("expected branch ref refs/heads/feat-a, got %q", got[1].BranchRef)
	}
	if got[1].BranchLabel != "feat-a" {
		t.Fatalf("expected branch label feat-a, got %q", got[1].BranchLabel)
	}
}

func TestParsePorcelainZDetachedWorktree(t *testing.T) {
	raw := strings.Join([]string{
		"worktree /repo",
		"HEAD 1111111",
		"branch refs/heads/main",
		"",
		"worktree /repo/.worktrees/detached",
		"HEAD 2222222",
		"detached",
		"",
	}, "\x00")

	got, err := ParsePorcelainZ(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if !got[1].IsDetached {
		t.Fatalf("expected detached worktree")
	}
	if got[1].BranchLabel != "(detached)" {
		t.Fatalf("expected detached label, got %q", got[1].BranchLabel)
	}
}

func TestParsePorcelainZIgnoresKnownExtraTokens(t *testing.T) {
	raw := strings.Join([]string{
		"worktree /repo",
		"HEAD 1111111",
		"branch refs/heads/main",
		"locked by another process",
		"",
		"worktree /repo/.worktrees/feat-a",
		"HEAD 2222222",
		"branch refs/heads/feat-a",
		"prunable gitdir file points to non-existent location",
		"",
	}, "\x00")

	got, err := ParsePorcelainZ(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[1].BranchLabel != "feat-a" {
		t.Fatalf("expected branch label feat-a, got %q", got[1].BranchLabel)
	}
}

func TestParsePorcelainZRejectsMalformedRecords(t *testing.T) {
	_, err := ParsePorcelainZ("branch refs/heads/main\x00")
	if err == nil {
		t.Fatal("expected malformed input error")
	}
}
