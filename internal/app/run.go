package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"wt/internal/git"
	"wt/internal/ui"
	"wt/internal/worktree"
)

type Deps interface {
	ListWorktrees(ctx context.Context) ([]worktree.Worktree, error)
	SelectWorktreeWithFzf(ctx context.Context, items []worktree.Worktree) (worktree.Worktree, error)
}

type RealDeps struct{}

func (RealDeps) ListWorktrees(ctx context.Context) ([]worktree.Worktree, error) {
	return git.ListWorktrees(ctx, git.ExecRunner{})
}

func (RealDeps) SelectWorktreeWithFzf(ctx context.Context, items []worktree.Worktree) (worktree.Worktree, error) {
	return ui.SelectWorktreeWithFzf(ctx, items, ui.ExecRunner{})
}

func Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
	if deps == nil {
		deps = RealDeps{}
	}

	if len(args) > 0 && args[0] == "--help" {
		fmt.Fprintln(out, "Usage: wt [--fzf] [index]")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Print the selected git worktree path.")
		return 0
	}

	if len(args) > 0 && args[0] == "--fzf" {
		if len(args) > 1 {
			fmt.Fprintf(errOut, "unexpected extra arguments: %s\n", strings.Join(args[1:], " "))
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
		if len(items) == 0 {
			fmt.Fprintln(errOut, "no worktrees available")
			return 1
		}

		selected, err := deps.SelectWorktreeWithFzf(ctx, items)
		if err != nil {
			switch {
			case errors.Is(err, ui.ErrFzfNotInstalled):
				fmt.Fprintln(errOut, "fzf is not installed")
				return 3
			case errors.Is(err, ui.ErrSelectionCanceled):
				return 130
			default:
				fmt.Fprintln(errOut, err)
				return 1
			}
		}

		fmt.Fprintln(out, selected.Path)
		return 0
	}

	if len(args) == 0 {
		items, err := deps.ListWorktrees(ctx)
		if err != nil {
			if errors.Is(err, git.ErrNotGitRepository) {
				fmt.Fprintln(errOut, "not a git repository")
				return 3
			}
			fmt.Fprintln(errOut, err)
			return 1
		}
		if len(items) == 0 {
			fmt.Fprintln(errOut, "no worktrees available")
			return 1
		}

		ui.RenderMenu(errOut, items)
		index, err := ui.ReadSelection(in, errOut, len(items))
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 1
			}
			fmt.Fprintln(errOut, err)
			return 1
		}
		for i := range items {
			if items[i].Index == index {
				fmt.Fprintln(out, items[i].Path)
				return 0
			}
		}
		fmt.Fprintf(errOut, "worktree index %d out of range\n", index)
		return 2
	}

	if len(args) > 1 {
		fmt.Fprintf(errOut, "unexpected extra arguments: %s\n", strings.Join(args[1:], " "))
		return 2
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
