package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ww/internal/worktree"
)

func TestFormatFzfCandidatesIncludesIndexStatusBranchAndPath(t *testing.T) {
	got := string(formatFzfCandidates([]worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true, IsDirty: true},
		{Index: 2, BranchLabel: "feat-a", Path: "/repo/.worktrees/feat-a", IsDirty: true},
	}))

	// IsDirty without IsMerged shows no DIRTY tag; IsCurrent shows [CURRENT]
	if !strings.Contains(got, "1\t[CURRENT]         \tmain  \t/repo") {
		t.Fatalf("expected current candidate, got %q", got)
	}
	// dirty-only without merged: empty status field (padded to humanStatusWidth=18)
	if !strings.Contains(got, "2\t                  \tfeat-a\t/repo/.worktrees/feat-a") {
		t.Fatalf("expected non-current candidate, got %q", got)
	}
}

func TestFormatFzfCandidatesPadsStatusAndBranchFields(t *testing.T) {
	got := string(formatFzfCandidates([]worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		{Index: 2, BranchLabel: "codex/current-dirty-status", Path: "/repo/.worktrees/current-dirty-status", IsDirty: true},
	}))

	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two candidates, got %q", got)
	}

	first := strings.Split(lines[0], "\t")
	second := strings.Split(lines[1], "\t")
	if len(first) != 4 || len(second) != 4 {
		t.Fatalf("expected four tab-separated fields, got %q", got)
	}

	if len(first[1]) != len(second[1]) {
		t.Fatalf("expected padded status fields, got %q and %q", first[1], second[1])
	}
	if len(first[2]) != len(second[2]) {
		t.Fatalf("expected padded branch fields, got %q and %q", first[2], second[2])
	}
	if strings.TrimSpace(first[2]) != "main" || strings.TrimSpace(second[2]) != "codex/current-dirty-status" {
		t.Fatalf("expected branch names to survive padding, got %q", got)
	}
}

func TestSelectWorktreeWithFzfReturnsSelectedWorktree(t *testing.T) {
	runner := &fakeFzfRunner{
		lookPath: "/usr/bin/fzf",
		stdout:   []byte("2\t                  \tfeat-a\t/repo/.worktrees/feat-a\n"),
	}

	got, err := SelectWorktreeWithFzf(context.Background(), []worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		{Index: 2, BranchLabel: "feat-a", Path: "/repo/.worktrees/feat-a", IsDirty: true},
	}, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Path != "/repo/.worktrees/feat-a" {
		t.Fatalf("expected selected worktree, got %#v", got)
	}
	if !strings.Contains(strings.Join(runner.gotArgs, " "), "--nth=2..") {
		t.Fatalf("expected fzf to search non-index fields without rewriting output, args=%q", runner.gotArgs)
	}
	if !strings.Contains(strings.Join(runner.gotArgs, " "), "--pointer=*") {
		t.Fatalf("expected fzf pointer marker to follow active selection, args=%q", runner.gotArgs)
	}
	if !strings.Contains(strings.Join(runner.gotArgs, " "), "--tac") {
		t.Fatalf("expected fzf to keep the list near the prompt while rendering top-down, args=%q", runner.gotArgs)
	}
	if !strings.Contains(strings.Join(runner.gotArgs, " "), "--bind=load:pos(2)") {
		t.Fatalf("expected fzf to focus current worktree by default, args=%q", runner.gotArgs)
	}
}

func TestSelectWorktreeWithFzfFocusesCurrentWorktreeByDefault(t *testing.T) {
	runner := &fakeFzfRunner{
		lookPath: "/usr/bin/fzf",
		stdout:   []byte("2\t[CURRENT]          \tmain\t/repo\n"),
	}

	_, err := SelectWorktreeWithFzf(context.Background(), []worktree.Worktree{
		{Index: 1, BranchLabel: "alpha", Path: "/repo/.worktrees/alpha"},
		{Index: 2, BranchLabel: "main", Path: "/repo", IsCurrent: true, IsDirty: true},
	}, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.Join(runner.gotArgs, " "), "--bind=load:pos(1)") {
		t.Fatalf("expected fzf to position cursor on current worktree, args=%q", runner.gotArgs)
	}
}

func TestSelectWorktreeWithFzfReturnsErrFzfNotInstalled(t *testing.T) {
	_, err := SelectWorktreeWithFzf(context.Background(), nil, &fakeFzfRunner{
		lookPathErr: errors.New("missing"),
	})
	if !errors.Is(err, ErrFzfNotInstalled) {
		t.Fatalf("expected ErrFzfNotInstalled, got %v", err)
	}
}

func TestSelectWorktreeWithFzfReturnsErrSelectionCanceled(t *testing.T) {
	_, err := SelectWorktreeWithFzf(context.Background(), []worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
	}, &fakeFzfRunner{
		lookPath: "/usr/bin/fzf",
		err:      exitError{code: 130},
	})
	if !errors.Is(err, ErrSelectionCanceled) {
		t.Fatalf("expected ErrSelectionCanceled, got %v", err)
	}
}

func TestFormatFzfCandidatesShowsMergedTag(t *testing.T) {
	got := string(formatFzfCandidates([]worktree.Worktree{
		{Index: 1, BranchLabel: "main", Path: "/repo", IsCurrent: true},
		{Index: 2, BranchLabel: "fix/typo", Path: "/wt/fix-typo", IsMerged: true},
	}))

	if !strings.Contains(got, "[MERGED]") {
		t.Fatalf("expected [MERGED] in fzf output, got %q", got)
	}
	if !strings.Contains(got, "[CURRENT]") {
		t.Fatalf("expected [CURRENT] in fzf output, got %q", got)
	}
}

type fakeFzfRunner struct {
	lookPath    string
	lookPathErr error
	stdout      []byte
	stderr      []byte
	err         error
	gotArgs     []string
}

func (f fakeFzfRunner) LookPath(string) (string, error) {
	return f.lookPath, f.lookPathErr
}

func (f *fakeFzfRunner) Run(_ context.Context, _ string, stdin []byte, args ...string) ([]byte, []byte, error) {
	f.gotArgs = append([]string(nil), args...)
	return append([]byte(nil), f.stdout...), append([]byte(nil), f.stderr...), f.err
}

type exitError struct {
	code int
}

func (e exitError) Error() string {
	return "exit status"
}

func (e exitError) ExitCode() int {
	return e.code
}
