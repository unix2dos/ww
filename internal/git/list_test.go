package git

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestListWorktreesParsesAndNormalizes(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"): "/repo\n",
			key("git", "-C", "/repo", "worktree", "list", "--porcelain", "-z"): strings.Join([]string{
				"worktree /repo",
				"HEAD 1111111",
				"branch refs/heads/main",
				"",
				"worktree /repo/.worktrees/feat-a",
				"HEAD 2222222",
				"branch refs/heads/feat-a",
				"",
			}, "\x00"),
		},
	}

	got, err := ListWorktrees(context.Background(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(got))
	}
	if !got[0].IsCurrent || got[0].Index != 1 {
		t.Fatalf("expected current worktree first, got %#v", got[0])
	}
	if got[1].Path != "/repo/.worktrees/feat-a" || got[1].Index != 2 {
		t.Fatalf("expected normalized numbering, got %#v", got[1])
	}
}

func TestListWorktreesMapsNonRepoError(t *testing.T) {
	runner := fakeRunner{
		errors: map[string]error{
			key("git", "rev-parse", "--show-toplevel"): errCommand("fatal: not a git repository (or any of the parent directories): .git"),
		},
	}

	_, err := ListWorktrees(context.Background(), runner)
	if !errors.Is(err, ErrNotGitRepository) {
		t.Fatalf("expected ErrNotGitRepository, got %v", err)
	}
}

type fakeRunner struct {
	outputs map[string]string
	errors  map[string]error
}

func (f fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	k := key(append([]string{name}, args...)...)
	if err, ok := f.errors[k]; ok {
		return nil, err
	}
	if out, ok := f.outputs[k]; ok {
		return []byte(out), nil
	}
	return nil, nil
}

func key(parts ...string) string {
	return strings.Join(parts, "\x00")
}

type errCommand string

func (e errCommand) Error() string { return string(e) }
