package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"wt/internal/git"
	"wt/internal/worktree"
)

type Deps interface {
	ListWorktrees(ctx context.Context) ([]worktree.Worktree, error)
}

type RealDeps struct{}

func (RealDeps) ListWorktrees(ctx context.Context) ([]worktree.Worktree, error) {
	return git.ListWorktrees(ctx, git.ExecRunner{})
}

func Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
	_ = in
	if deps == nil {
		deps = RealDeps{}
	}

	if len(args) > 0 && args[0] == "--help" {
		fmt.Fprintln(out, "Usage: wt [--fzf] [index]")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Print the selected git worktree path.")
		return 0
	}

	if len(args) == 0 || args[0] == "--fzf" {
		fmt.Fprintln(errOut, "interactive selection not implemented")
		return 1
	}

	index, err := strconv.Atoi(args[0])
	if err != nil || index <= 0 {
		fmt.Fprintf(errOut, "invalid worktree index: %q\n", args[0])
		return 2
	}

	items, err := deps.ListWorktrees(ctx)
	if err != nil {
		if errors.Is(err, git.ErrNotGitRepository) {
			fmt.Fprintln(errOut, "not a git repository")
			return 3
		}
		fmt.Fprintln(errOut, err)
		return 1
	}

	var selected *worktree.Worktree
	for i := range items {
		if items[i].Index == index {
			selected = &items[i]
			break
		}
	}
	if selected == nil {
		fmt.Fprintf(errOut, "worktree index %d out of range\n", index)
		return 2
	}

	fmt.Fprintln(out, selected.Path)
	return 0
}
