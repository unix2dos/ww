package git

import (
	"context"
	"strings"
	"testing"
)

func TestDefaultBranchResolvesOriginHeadFirst(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                   "/repo/.worktrees/feat-a\n",
			key("git", "-C", "/repo/.worktrees/feat-a", "rev-parse", "--git-common-dir"): "/repo/.git\n",
			key("git", "-C", "/repo", "symbolic-ref", "refs/remotes/origin/HEAD"):        "refs/remotes/origin/main\n",
		},
	}

	got, err := DefaultBranch(context.Background(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "main" {
		t.Fatalf("expected main, got %q", got)
	}
}

func TestDefaultBranchFallsBackToLocalMain(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                           "/repo\n",
			key("git", "-C", "/repo", "rev-parse", "--git-common-dir"):                           "/repo/.git\n",
			key("git", "-C", "/repo", "branch", "--list", "--format=%(refname:short)", "main"):   "main\n",
			key("git", "-C", "/repo", "branch", "--list", "--format=%(refname:short)", "master"): "",
		},
		errors: map[string]error{
			key("git", "-C", "/repo", "symbolic-ref", "refs/remotes/origin/HEAD"): errCommand("exit status 1"),
		},
	}

	got, err := DefaultBranch(context.Background(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "main" {
		t.Fatalf("expected main, got %q", got)
	}
}

func TestDefaultBranchFallsBackToLocalMaster(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                           "/repo\n",
			key("git", "-C", "/repo", "rev-parse", "--git-common-dir"):                           "/repo/.git\n",
			key("git", "-C", "/repo", "branch", "--list", "--format=%(refname:short)", "main"):   "",
			key("git", "-C", "/repo", "branch", "--list", "--format=%(refname:short)", "master"): "master\n",
		},
		errors: map[string]error{
			key("git", "-C", "/repo", "symbolic-ref", "refs/remotes/origin/HEAD"): errCommand("exit status 1"),
		},
	}

	got, err := DefaultBranch(context.Background(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "master" {
		t.Fatalf("expected master, got %q", got)
	}
}

func TestDefaultBranchReturnsHelpfulErrorWhenNoDefaultBranchCanBeResolved(t *testing.T) {
	runner := &recordingRunner{
		outputs: map[string]string{
			key("git", "rev-parse", "--show-toplevel"):                                           "/repo\n",
			key("git", "-C", "/repo", "rev-parse", "--git-common-dir"):                           "/repo/.git\n",
			key("git", "-C", "/repo", "branch", "--list", "--format=%(refname:short)", "main"):   "",
			key("git", "-C", "/repo", "branch", "--list", "--format=%(refname:short)", "master"): "",
		},
		errors: map[string]error{
			key("git", "-C", "/repo", "symbolic-ref", "refs/remotes/origin/HEAD"): errCommand("exit status 1"),
		},
	}

	_, err := DefaultBranch(context.Background(), runner)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "default branch") {
		t.Fatalf("expected helpful error, got %v", err)
	}
}
