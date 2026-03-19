package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"wt/internal/git"
	"wt/internal/state"
	"wt/internal/ui"
	"wt/internal/worktree"
)

type Deps interface {
	CurrentRepoKey(ctx context.Context) (string, error)
	ListWorktrees(ctx context.Context) (string, []worktree.Worktree, error)
	SelectWorktreeWithFzf(ctx context.Context, items []worktree.Worktree) (worktree.Worktree, error)
	SelectWorktreeWithTUI(in io.Reader, out io.Writer, items []worktree.Worktree) (worktree.Worktree, error)
	CreateWorktree(ctx context.Context, name string) (string, error)
	LoadWorktreeState(ctx context.Context, repoKey string) (map[string]int64, error)
	TouchWorktreeState(ctx context.Context, repoKey, path string) error
}

type RealDeps struct{}

var defaultStateStore struct {
	once  sync.Once
	store *state.Store
	err   error
}

func ensureStore() (*state.Store, error) {
	defaultStateStore.once.Do(func() {
		defaultStateStore.store, defaultStateStore.err = state.NewStore()
	})
	return defaultStateStore.store, defaultStateStore.err
}

func (d RealDeps) ListWorktrees(ctx context.Context) (string, []worktree.Worktree, error) {
	return git.ListWorktrees(ctx, git.ExecRunner{})
}

func (d RealDeps) CurrentRepoKey(ctx context.Context) (string, error) {
	return git.CurrentRepoKey(ctx, git.ExecRunner{})
}

func (d RealDeps) SelectWorktreeWithFzf(ctx context.Context, items []worktree.Worktree) (worktree.Worktree, error) {
	return ui.SelectWorktreeWithFzf(ctx, items, ui.ExecRunner{})
}

func (d RealDeps) SelectWorktreeWithTUI(in io.Reader, out io.Writer, items []worktree.Worktree) (worktree.Worktree, error) {
	return ui.SelectWorktreeWithTUI(in, out, items, ui.OSRawMode{})
}

func (d RealDeps) CreateWorktree(ctx context.Context, name string) (string, error) {
	return git.CreateWorktree(ctx, git.ExecRunner{}, name)
}

func (d RealDeps) LoadWorktreeState(_ context.Context, repoKey string) (map[string]int64, error) {
	store, err := ensureStore()
	if err != nil {
		return nil, err
	}
	return store.Load(repoKey)
}

func (d RealDeps) TouchWorktreeState(_ context.Context, repoKey, path string) error {
	store, err := ensureStore()
	if err != nil {
		return err
	}
	return store.Touch(repoKey, path)
}

func Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
	if deps == nil {
		deps = &RealDeps{}
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

		repoKey, items, warn, err := orderedWorktrees(ctx, deps)
		if err != nil {
			return writeWorktreeError(errOut, err)
		}
		warnStateIssue(errOut, warn)

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
		warnStateIssue(errOut, touchWorktreeStateBestEffort(ctx, deps, repoKey, selected.Path))
		return 0
	}

	if len(args) == 0 {
		repoKey, items, warn, err := orderedWorktrees(ctx, deps)
		if err != nil {
			return writeWorktreeError(errOut, err)
		}
		warnStateIssue(errOut, warn)

		selected, err := selectInteractiveWorktree(ctx, in, errOut, items, deps, false)
		if err != nil {
			return writeSelectionError(errOut, err)
		}
		fmt.Fprintln(out, selected.Path)
		warnStateIssue(errOut, touchWorktreeStateBestEffort(ctx, deps, repoKey, selected.Path))
		return 0
	}

	if len(args) > 1 {
		fmt.Fprintf(errOut, "unexpected extra arguments: %s\n", strings.Join(args[1:], " "))
		return 2
	}

	repoKey, items, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeWorktreeError(errOut, err)
	}
	warnStateIssue(errOut, warn)

	index, err := strconv.Atoi(args[0])
	if err == nil {
		if index <= 0 {
			fmt.Fprintf(errOut, "invalid worktree index: %q\n", args[0])
			return 2
		}
		selected, ok := selectByIndex(items, index)
		if !ok {
			fmt.Fprintf(errOut, "worktree index %d out of range\n", index)
			return 2
		}

		fmt.Fprintln(out, selected.Path)
		warnStateIssue(errOut, touchWorktreeStateBestEffort(ctx, deps, repoKey, selected.Path))
		return 0
	}

	selected, err := worktree.Match(items, args[0])
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}

	fmt.Fprintln(out, selected.Path)
	warnStateIssue(errOut, touchWorktreeStateBestEffort(ctx, deps, repoKey, selected.Path))
	return 0
}

func runList(ctx context.Context, out io.Writer, errOut io.Writer, deps Deps) int {
	_, items, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeWorktreeError(errOut, err)
	}
	warnStateIssue(errOut, warn)
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

	repoKey, err := deps.CurrentRepoKey(ctx)
	if err != nil {
		return writeWorktreeError(errOut, err)
	}

	path, err := deps.CreateWorktree(ctx, args[0])
	if err != nil {
		return writeWorktreeError(errOut, err)
	}

	fmt.Fprintln(out, path)
	warnStateIssue(errOut, touchWorktreeStateBestEffort(ctx, deps, repoKey, path))
	return 0
}

func orderedWorktrees(ctx context.Context, deps Deps) (string, []worktree.Worktree, error, error) {
	repoKey, items, err := deps.ListWorktrees(ctx)
	if err != nil {
		return "", nil, nil, err
	}
	mru, err := deps.LoadWorktreeState(ctx, repoKey)
	if err != nil {
		normalized := worktree.Normalize(items)
		return repoKey, normalized, fmt.Errorf("state load unavailable: %w", err), nil
	}
	for i := range items {
		items[i].LastUsedAt = mru[items[i].Path]
	}
	return repoKey, worktree.Normalize(items), nil, nil
}

func selectInteractiveWorktree(ctx context.Context, in io.Reader, errOut io.Writer, items []worktree.Worktree, deps Deps, forceFzf bool) (worktree.Worktree, error) {
	if forceFzf {
		return deps.SelectWorktreeWithFzf(ctx, items)
	}

	selected, err := deps.SelectWorktreeWithFzf(ctx, items)
	switch {
	case err == nil:
		return selected, nil
	case errors.Is(err, ui.ErrFzfNotInstalled):
		return deps.SelectWorktreeWithTUI(in, errOut, items)
	default:
		return worktree.Worktree{}, err
	}
}

func selectByIndex(items []worktree.Worktree, index int) (worktree.Worktree, bool) {
	for i := range items {
		if items[i].Index == index {
			return items[i], true
		}
	}
	return worktree.Worktree{}, false
}

func writeWorktreeError(errOut io.Writer, err error) int {
	if errors.Is(err, git.ErrNotGitRepository) {
		fmt.Fprintln(errOut, "not a git repository")
		return 3
	}
	fmt.Fprintln(errOut, err)
	return 1
}

func writeSelectionError(errOut io.Writer, err error) int {
	switch {
	case errors.Is(err, ui.ErrFzfNotInstalled):
		fmt.Fprintln(errOut, "fzf is not installed")
		return 3
	case errors.Is(err, ui.ErrSelectionCanceled):
		return 130
	case errors.Is(err, io.EOF):
		return 1
	default:
		fmt.Fprintln(errOut, err)
		return 1
	}
}

func touchWorktreeStateBestEffort(ctx context.Context, deps Deps, repoKey, path string) error {
	if err := deps.TouchWorktreeState(ctx, repoKey, path); err != nil {
		return fmt.Errorf("state update skipped: %w", err)
	}
	return nil
}

func warnStateIssue(errOut io.Writer, err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(errOut, err)
}

func printHelperHelp(out io.Writer) {
	fmt.Fprintln(out, "Usage: ww-helper [switch-path|list|new-path|--help]")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "switch-path prints the selected git worktree path.")
	fmt.Fprintln(out, "Interactive switch uses fzf when available, otherwise the built-in selector.")
	fmt.Fprintln(out, "list prints the current worktree table.")
	fmt.Fprintln(out, "new-path creates a worktree and prints its path.")
}
