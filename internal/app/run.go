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
	CreateWorktree(ctx context.Context, name string) (string, error)
}

type RealDeps struct{}

func (RealDeps) ListWorktrees(ctx context.Context) ([]worktree.Worktree, error) {
	return git.ListWorktrees(ctx, git.ExecRunner{})
}

func (RealDeps) SelectWorktreeWithFzf(ctx context.Context, items []worktree.Worktree) (worktree.Worktree, error) {
	return ui.SelectWorktreeWithFzf(ctx, items, ui.ExecRunner{})
}

func (RealDeps) CreateWorktree(ctx context.Context, name string) (string, error) {
	return git.CreateWorktree(ctx, git.ExecRunner{}, name)
}

func Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
	if deps == nil {
		deps = RealDeps{}
	}

	if len(args) == 0 {
		return runSwitchPath(ctx, args, in, out, errOut, deps)
	}

	switch args[0] {
	case "--help", "-h", "help":
		printHelperHelp(out)
		return 0
	case "switch-path":
		return runSwitchPath(ctx, args[1:], in, out, errOut, deps)
	case "new-path":
		return runNewPath(ctx, args[1:], out, errOut, deps)
	case "list":
		return runList(ctx, out, errOut, deps)
	default:
		return runSwitchPath(ctx, args, in, out, errOut, deps)
	}
}

func runSwitchPath(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
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

func runList(ctx context.Context, out io.Writer, errOut io.Writer, deps Deps) int {
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

	for _, item := range items {
		marker := " "
		if item.IsCurrent {
			marker = "*"
		}
		fmt.Fprintf(out, "[%d] %s %s %s\n", item.Index, marker, item.BranchLabel, item.Path)
	}
	return 0
}

func runNewPath(ctx context.Context, args []string, out io.Writer, errOut io.Writer, deps Deps) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "missing worktree name")
		return 2
	}
	if len(args) > 1 {
		fmt.Fprintf(errOut, "unexpected extra arguments: %s\n", strings.Join(args[1:], " "))
		return 2
	}

	path, err := deps.CreateWorktree(ctx, args[0])
	if err != nil {
		if errors.Is(err, git.ErrNotGitRepository) {
			fmt.Fprintln(errOut, "not a git repository")
			return 3
		}
		fmt.Fprintln(errOut, err)
		return 1
	}

	fmt.Fprintln(out, path)
	return 0
}

func printHelperHelp(out io.Writer) {
	fmt.Fprintln(out, "Usage: ww-helper [switch-path|list|new-path|--help]")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "switch-path prints the selected git worktree path.")
	fmt.Fprintln(out, "list prints the current worktree table.")
	fmt.Fprintln(out, "new-path creates a worktree and prints its path.")
}
