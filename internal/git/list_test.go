package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListWorktreesReturnsRepoKeyAndRawItems(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                   "/repo/worktrees/current\n",
			key("git", "-C", "/repo/worktrees/current", "rev-parse", "--git-common-dir"): "/repo/.git\n",
			key("git", "-C", "/repo/worktrees/current", "worktree", "list", "--porcelain", "-z"): strings.Join([]string{
				"worktree /repo/worktrees/current",
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

	repoKey, got, err := ListWorktrees(context.Background(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repoKey != "/repo/.git" {
		t.Fatalf("expected repo key /repo/.git, got %q", repoKey)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(got))
	}
	if !got[0].IsCurrent {
		t.Fatalf("expected current worktree marked current, got %#v", got[0])
	}
	if got[1].Path != "/repo/.worktrees/feat-a" {
		t.Fatalf("expected raw worktree order preserved, got %#v", got[1])
	}
}

func TestListWorktreesMapsNonRepoError(t *testing.T) {
	runner := fakeRunner{
		errors: map[string]error{
			key("git", "rev-parse", "--show-toplevel"): errCommand("exit status 128"),
		},
		stderr: map[string]string{
			key("git", "rev-parse", "--show-toplevel"): "fatal: not a git repository (or any of the parent directories): .git\n",
		},
	}

	_, _, err := ListWorktrees(context.Background(), runner)
	if !errors.Is(err, ErrNotGitRepository) {
		t.Fatalf("expected ErrNotGitRepository, got %v", err)
	}
}

func TestListWorktreesIgnoresStderrOnSuccess(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                   "/repo/worktrees/current\n",
			key("git", "-C", "/repo/worktrees/current", "rev-parse", "--git-common-dir"): "/repo/.git\n",
			key("git", "-C", "/repo/worktrees/current", "worktree", "list", "--porcelain", "-z"): strings.Join([]string{
				"worktree /repo/worktrees/current",
				"HEAD 1111111",
				"branch refs/heads/main",
				"",
			}, "\x00"),
		},
		stderr: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                           "hint: noisy but harmless\n",
			key("git", "-C", "/repo/worktrees/current", "rev-parse", "--git-common-dir"):         "hint: noisy but harmless\n",
			key("git", "-C", "/repo/worktrees/current", "worktree", "list", "--porcelain", "-z"): "hint: noisy but harmless\n",
		},
	}

	repoKey, got, err := ListWorktrees(context.Background(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repoKey != "/repo/.git" {
		t.Fatalf("expected repo key /repo/.git, got %q", repoKey)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(got))
	}
	if got[0].Path != "/repo/worktrees/current" {
		t.Fatalf("expected parsed stdout only, got %#v", got[0])
	}
}

func TestListWorktreesAnnotatesCreationTimesWhenPathsExist(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, "current")
	feature := filepath.Join(root, ".worktrees", "feat-a")
	if err := os.MkdirAll(current, 0o755); err != nil {
		t.Fatalf("mkdir current: %v", err)
	}
	if err := os.MkdirAll(feature, 0o755); err != nil {
		t.Fatalf("mkdir feature: %v", err)
	}

	runner := fakeRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                 current + "\n",
			key("git", "-C", current, "rev-parse", "--git-common-dir"): filepath.Join(root, ".git") + "\n",
			key("git", "-C", current, "worktree", "list", "--porcelain", "-z"): strings.Join([]string{
				"worktree " + current,
				"HEAD 1111111",
				"branch refs/heads/main",
				"",
				"worktree " + feature,
				"HEAD 2222222",
				"branch refs/heads/feat-a",
				"",
			}, "\x00"),
		},
	}

	_, got, err := ListWorktrees(context.Background(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].CreatedAt == 0 {
		t.Fatalf("expected current worktree creation time, got %#v", got[0])
	}
	if got[1].CreatedAt == 0 {
		t.Fatalf("expected linked worktree creation time, got %#v", got[1])
	}
}

func TestCurrentRepoKeyReturnsCanonicalGitCommonDir(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                    "/repo/.worktrees/current\n",
			key("git", "-C", "/repo/.worktrees/current", "rev-parse", "--git-common-dir"): "/repo/.git\n",
		},
	}

	repoKey, err := CurrentRepoKey(context.Background(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repoKey != "/repo/.git" {
		t.Fatalf("expected repo key /repo/.git, got %q", repoKey)
	}
}

func TestCurrentRepoKeyMapsNonRepoError(t *testing.T) {
	runner := fakeRunner{
		errors: map[string]error{
			key("git", "rev-parse", "--show-toplevel"): errCommand("exit status 128"),
		},
		stderr: map[string]string{
			key("git", "rev-parse", "--show-toplevel"): "fatal: not a git repository (or any of the parent directories): .git\n",
		},
	}

	_, err := CurrentRepoKey(context.Background(), runner)
	if !errors.Is(err, ErrNotGitRepository) {
		t.Fatalf("expected ErrNotGitRepository, got %v", err)
	}
}

type fakeRunner struct {
	outputs map[string]string
	stderr  map[string]string
	errors  map[string]error
}

func (f fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	k := key(append([]string{name}, args...)...)
	if err, ok := f.errors[k]; ok {
		return nil, []byte(f.stderr[k]), err
	}
	out := []byte(f.outputs[k])
	errOut := []byte(f.stderr[k])
	if out != nil || errOut != nil {
		return out, errOut, nil
	}
	return nil, nil, nil
}

func key(parts ...string) string {
	return strings.Join(parts, "\x00")
}

type errCommand string

func (e errCommand) Error() string { return fmt.Sprintf("%s", string(e)) }
