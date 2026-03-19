package git

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateWorktreeCreatesRelativePathFromRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	runner := &createFakeRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"): repoRoot + "\n",
		},
	}

	got, err := CreateWorktree(context.Background(), runner, "feat-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(repoRoot, ".worktrees", "feat-a")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	wantCmd := key("git", "-C", repoRoot, "worktree", "add", "-b", "feat-a", filepath.Join(".worktrees", "feat-a"), "HEAD")
	if _, ok := runner.commands[wantCmd]; !ok {
		t.Fatalf("expected command %q to be executed, got %#v", wantCmd, runner.commands)
	}
}

func TestCreateWorktreeMapsNonRepoError(t *testing.T) {
	runner := &createFakeRunner{
		errors: map[string]error{
			key("git", "rev-parse", "--show-toplevel"): errCommand("exit status 128"),
		},
		stderr: map[string]string{
			key("git", "rev-parse", "--show-toplevel"): "fatal: not a git repository (or any of the parent directories): .git\n",
		},
	}

	_, err := CreateWorktree(context.Background(), runner, "feat-a")
	if !errors.Is(err, ErrNotGitRepository) {
		t.Fatalf("expected ErrNotGitRepository, got %v", err)
	}
}

func TestCreateWorktreeReturnsCommandError(t *testing.T) {
	repoRoot := t.TempDir()
	runner := &createFakeRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"): repoRoot + "\n",
		},
		errors: map[string]error{
			key("git", "-C", repoRoot, "worktree", "add", "-b", "feat-a", filepath.Join(".worktrees", "feat-a"), "HEAD"): errCommand("exit status 128"),
		},
		stderr: map[string]string{
			key("git", "-C", repoRoot, "worktree", "add", "-b", "feat-a", filepath.Join(".worktrees", "feat-a"), "HEAD"): "fatal: 'feat-a' is already used by worktree at '/repo/.worktrees/feat-a'\n",
		},
	}

	_, err := CreateWorktree(context.Background(), runner, "feat-a")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "git worktree add") {
		t.Fatalf("expected wrapped git worktree add error, got %v", err)
	}
	if !strings.Contains(err.Error(), "already used by worktree") {
		t.Fatalf("expected git stderr in error, got %v", err)
	}
}

type createFakeRunner struct {
	outputs  map[string]string
	stderr   map[string]string
	errors   map[string]error
	commands map[string]struct{}
}

func (f *createFakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	if f.commands == nil {
		f.commands = map[string]struct{}{}
	}
	k := key(append([]string{name}, args...)...)
	f.commands[k] = struct{}{}
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
