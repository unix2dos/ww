package git

import (
	"context"
	"strings"
	"testing"
)

func TestDiffSummaryReturnsAheadBehindCommitsAndFileStats(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                              "/repo/.worktrees/feat-current\n",
			key("git", "-C", "/repo/.worktrees/feat-current", "rev-parse", "--git-common-dir"):      "/repo/.git\n",
			key("git", "-C", "/repo/.worktrees/feat-current", "branch", "--show-current"):           "feat-current\n",
			key("git", "-C", "/repo", "rev-list", "--left-right", "--count", "main...feat-current"): "3\t5\n",
			key("git", "-C", "/repo", "log", "--format=%H%x09%s", "main..feat-current"):             "aaa111\tfeat: add list\nbbb222\tfix: clean output\n",
			key("git", "-C", "/repo", "diff", "--numstat", "main...feat-current"):                   "10\t2\tinternal/app/run.go\n-\t-\tdocs/assets/demo.png\n",
		},
	}

	got, err := DiffSummary(context.Background(), runner, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.CurrentBranch != "feat-current" || got.TargetBranch != "main" {
		t.Fatalf("unexpected branches: %#v", got)
	}
	if got.Behind != 3 || got.Ahead != 5 {
		t.Fatalf("expected ahead/behind counts, got %#v", got)
	}
	if got.ChangedFiles != 2 || got.Insertions != 10 || got.Deletions != 2 {
		t.Fatalf("expected numstat totals, got %#v", got)
	}
	if len(got.Commits) != 2 || got.Commits[0].Subject != "feat: add list" {
		t.Fatalf("expected commit summaries, got %#v", got.Commits)
	}
	if len(got.Files) != 2 || got.Files[1].Path != "docs/assets/demo.png" {
		t.Fatalf("expected file stats, got %#v", got.Files)
	}
}

func TestDiffPatchReturnsMergeBasePatch(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                         "/repo/.worktrees/feat-current\n",
			key("git", "-C", "/repo/.worktrees/feat-current", "rev-parse", "--git-common-dir"): "/repo/.git\n",
			key("git", "-C", "/repo/.worktrees/feat-current", "branch", "--show-current"):      "feat-current\n",
			key("git", "-C", "/repo", "diff", "main...feat-current"):                           "diff --git a/file b/file\n",
		},
	}

	got, err := DiffPatch(context.Background(), runner, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "diff --git") {
		t.Fatalf("expected patch output, got %q", got)
	}
}

func TestDiffSummaryReturnsHelpfulErrorWhenDetached(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                         "/repo/.worktrees/feat-current\n",
			key("git", "-C", "/repo/.worktrees/feat-current", "rev-parse", "--git-common-dir"): "/repo/.git\n",
			key("git", "-C", "/repo/.worktrees/feat-current", "branch", "--show-current"):      "\n",
		},
	}

	_, err := DiffSummary(context.Background(), runner, "main")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "current branch") {
		t.Fatalf("expected detached branch error, got %v", err)
	}
}

type recordingRunner struct {
	outputs  map[string]string
	stderr   map[string]string
	errors   map[string]error
	commands map[string]struct{}
}

func (f *recordingRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
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
