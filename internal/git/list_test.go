package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ww/internal/worktree"
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

func TestListWorktreesAnnotatesDirtyState(t *testing.T) {
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
			key("git", "-C", "/repo/worktrees/current", "status", "--porcelain", "--", ".", ":(exclude).worktrees"): "",
			key("git", "-C", "/repo/.worktrees/feat-a", "status", "--porcelain", "--", ".", ":(exclude).worktrees"): "?? scratch.txt\n",
		},
	}

	_, got, err := ListWorktrees(context.Background(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].IsDirty {
		t.Fatalf("expected current worktree clean, got %#v", got[0])
	}
	if !got[1].IsDirty {
		t.Fatalf("expected linked worktree dirty, got %#v", got[1])
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

func TestAnnotateExtendedStatusPopulatesAllFields(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string]string{
			// FileChangeCounts for worktree 1 (main — skip merged/ahead-behind)
			key("git", "-C", "/repo", "status", "--porcelain", "--", ".", ":(exclude).worktrees"): "A  new.go\n M old.go\n",
			// FileChangeCounts for worktree 2
			key("git", "-C", "/repo/.worktrees/feat-a", "status", "--porcelain", "--", ".", ":(exclude).worktrees"): "?? scratch.txt\n",
			// AheadBehind for worktree 2
			key("git", "-C", "/repo/.worktrees/feat-a", "rev-list", "--left-right", "--count", "feat-a...main"): "3\t1\n",
			// BranchMergedIntoBase for worktree 2
			key("git", "-C", "/repo/.worktrees/feat-a", "branch", "--format=%(refname:short)", "--merged", "main"): "main\n",
		},
	}

	items := []worktree.Worktree{
		{Path: "/repo", BranchRef: "refs/heads/main", BranchLabel: "main", IsCurrent: true},
		{Path: "/repo/.worktrees/feat-a", BranchRef: "refs/heads/feat-a", BranchLabel: "feat-a"},
	}

	err := AnnotateExtendedStatus(context.Background(), runner, items, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// main: should have file changes but no merged/ahead/behind
	if items[0].Staged != 1 || items[0].Unstaged != 1 {
		t.Fatalf("expected main staged=1 unstaged=1, got %d %d", items[0].Staged, items[0].Unstaged)
	}
	if items[0].IsMerged || items[0].Ahead != 0 || items[0].Behind != 0 {
		t.Fatalf("expected main to skip branch-level checks, got merged=%v ahead=%d behind=%d", items[0].IsMerged, items[0].Ahead, items[0].Behind)
	}
	if !items[0].IsDirty {
		t.Fatal("expected main IsDirty=true")
	}

	// feat-a: should have file changes and NOT be merged (since "feat-a" is not in the merged list)
	if items[1].Untracked != 1 {
		t.Fatalf("expected feat-a untracked=1, got %d", items[1].Untracked)
	}
	if items[1].Ahead != 3 || items[1].Behind != 1 {
		t.Fatalf("expected feat-a ahead=3 behind=1, got %d %d", items[1].Ahead, items[1].Behind)
	}
	if items[1].IsDirty != true {
		t.Fatal("expected feat-a IsDirty=true")
	}
}

func TestAnnotateExtendedStatusSkipsDetachedHead(t *testing.T) {
	runner := fakeRunner{
		outputs: map[string]string{
			key("git", "-C", "/repo/.worktrees/detached", "status", "--porcelain", "--", ".", ":(exclude).worktrees"): "",
		},
	}

	items := []worktree.Worktree{
		{Path: "/repo/.worktrees/detached", BranchLabel: "(detached)", IsDetached: true},
	}

	err := AnnotateExtendedStatus(context.Background(), runner, items, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if items[0].IsMerged || items[0].Ahead != 0 || items[0].Behind != 0 {
		t.Fatalf("expected detached to skip branch checks, got merged=%v ahead=%d behind=%d", items[0].IsMerged, items[0].Ahead, items[0].Behind)
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
