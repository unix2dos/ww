package git

import (
	"context"
	"strings"
	"testing"

	"ww/internal/worktree"
)

func TestPreviewRemovalMarksDirtyAndUnmergedWorktree(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "-C", "/repo/.worktrees/feat-a", "status", "--porcelain", "--", ".", ":(exclude).worktrees"): " M README.md\n",
			key("git", "-C", "/repo/.worktrees/feat-a", "branch", "--format=%(refname:short)", "--merged", "main"):  "main\n",
		},
	}

	got, err := PreviewRemoval(context.Background(), runner, worktree.Worktree{
		Path:        "/repo/.worktrees/feat-a",
		BranchLabel: "feat-a",
		BranchRef:   "refs/heads/feat-a",
	}, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Dirty {
		t.Fatalf("expected dirty preview, got %#v", got)
	}
	if got.BranchMerged {
		t.Fatalf("expected branch to be marked unmerged, got %#v", got)
	}
	if got.DeleteBranch {
		t.Fatalf("expected branch deletion to be skipped, got %#v", got)
	}
}

func TestPreviewRemovalKeepsBaseBranchEvenWhenMerged(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "-C", "/repo/.worktrees/main", "status", "--porcelain"):                                   "",
			key("git", "-C", "/repo/.worktrees/main", "branch", "--format=%(refname:short)", "--merged", "main"): "main\n",
		},
	}

	got, err := PreviewRemoval(context.Background(), runner, worktree.Worktree{
		Path:        "/repo/.worktrees/main",
		BranchLabel: "main",
		BranchRef:   "refs/heads/main",
	}, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.BranchMerged {
		t.Fatalf("expected base branch to still count as merged, got %#v", got)
	}
	if got.DeleteBranch {
		t.Fatalf("expected base branch deletion to be skipped, got %#v", got)
	}
}

func TestRemoveWorktreeDeletesMergedBranch(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                                              "/repo/.worktrees/current\n",
			key("git", "-C", "/repo/.worktrees/current", "rev-parse", "--git-common-dir"):                           "/repo/.git\n",
			key("git", "-C", "/repo/.worktrees/feat-a", "status", "--porcelain", "--", ".", ":(exclude).worktrees"): "",
			key("git", "-C", "/repo/.worktrees/feat-a", "branch", "--format=%(refname:short)", "--merged", "main"):  "main\nfeat-a\n",
		},
	}

	got, err := RemoveWorktree(context.Background(), runner, worktree.Worktree{
		Path:        "/repo/.worktrees/feat-a",
		BranchLabel: "feat-a",
		BranchRef:   "refs/heads/feat-a",
	}, RemoveOptions{BaseBranch: "main"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.RemovedWorktree || !got.DeletedBranch {
		t.Fatalf("expected worktree and branch removal, got %#v", got)
	}

	removeCmd := key("git", "-C", "/repo", "worktree", "remove", "/repo/.worktrees/feat-a")
	if _, ok := runner.commands[removeCmd]; !ok {
		t.Fatalf("expected remove command %q, got %#v", removeCmd, runner.commands)
	}
	deleteCmd := key("git", "-C", "/repo", "branch", "-d", "feat-a")
	if _, ok := runner.commands[deleteCmd]; !ok {
		t.Fatalf("expected delete command %q, got %#v", deleteCmd, runner.commands)
	}
}

func TestRemoveWorktreeKeepsUnmergedBranch(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                                              "/repo/.worktrees/current\n",
			key("git", "-C", "/repo/.worktrees/current", "rev-parse", "--git-common-dir"):                           "/repo/.git\n",
			key("git", "-C", "/repo/.worktrees/feat-a", "status", "--porcelain", "--", ".", ":(exclude).worktrees"): "",
			key("git", "-C", "/repo/.worktrees/feat-a", "branch", "--format=%(refname:short)", "--merged", "main"):  "main\n",
		},
	}

	got, err := RemoveWorktree(context.Background(), runner, worktree.Worktree{
		Path:        "/repo/.worktrees/feat-a",
		BranchLabel: "feat-a",
		BranchRef:   "refs/heads/feat-a",
	}, RemoveOptions{BaseBranch: "main"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.RemovedWorktree || got.DeletedBranch {
		t.Fatalf("expected only worktree removal, got %#v", got)
	}
	if got.KeptBranchReason != "not merged" {
		t.Fatalf("expected keep reason, got %#v", got)
	}
}

func TestRemoveWorktreeKeepsBaseBranch(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                                           "/repo/.worktrees/current\n",
			key("git", "-C", "/repo/.worktrees/current", "rev-parse", "--git-common-dir"):                        "/repo/.git\n",
			key("git", "-C", "/repo/.worktrees/main", "status", "--porcelain"):                                   "",
			key("git", "-C", "/repo/.worktrees/main", "branch", "--format=%(refname:short)", "--merged", "main"): "main\n",
		},
	}

	got, err := RemoveWorktree(context.Background(), runner, worktree.Worktree{
		Path:        "/repo/.worktrees/main",
		BranchLabel: "main",
		BranchRef:   "refs/heads/main",
	}, RemoveOptions{BaseBranch: "main"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.RemovedWorktree || got.DeletedBranch {
		t.Fatalf("expected only worktree removal, got %#v", got)
	}
	if got.KeptBranchReason != "base branch" {
		t.Fatalf("expected base-branch keep reason, got %#v", got)
	}
	deleteCmd := key("git", "-C", "/repo", "branch", "-d", "main")
	if _, ok := runner.commands[deleteCmd]; ok {
		t.Fatalf("did not expect delete command %q, got %#v", deleteCmd, runner.commands)
	}
}

func TestRemoveWorktreeRejectsDirtyWorktreeWithoutForce(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "-C", "/repo/.worktrees/feat-a", "status", "--porcelain", "--", ".", ":(exclude).worktrees"): " M README.md\n",
			key("git", "-C", "/repo/.worktrees/feat-a", "branch", "--format=%(refname:short)", "--merged", "main"):  "main\nfeat-a\n",
		},
	}

	_, err := RemoveWorktree(context.Background(), runner, worktree.Worktree{
		Path:        "/repo/.worktrees/feat-a",
		BranchLabel: "feat-a",
		BranchRef:   "refs/heads/feat-a",
	}, RemoveOptions{BaseBranch: "main"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected force hint, got %v", err)
	}
	for cmd := range runner.commands {
		if strings.Contains(cmd, "\x00worktree\x00remove\x00") {
			t.Fatalf("expected no removal command, got %#v", runner.commands)
		}
	}
}

func TestRemoveWorktreeRejectsCurrentWorktree(t *testing.T) {
	_, err := RemoveWorktree(context.Background(), &recordingRunner{}, worktree.Worktree{
		Path:        "/repo",
		BranchLabel: "main",
		IsCurrent:   true,
	}, RemoveOptions{BaseBranch: "main"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "active worktree") {
		t.Fatalf("expected active worktree error, got %v", err)
	}
}
