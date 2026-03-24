package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"ww/internal/state"
	"ww/internal/tasknote"
	"ww/internal/worktree"
)

func runCheck(ctx context.Context, args []string, out io.Writer, errOut io.Writer, deps Deps) int {
	if len(args) > 0 {
		return writeCommandError("check", out, errOut, false, appError{
			Code:     "INVALID_ARGUMENTS",
			Message:  fmt.Sprintf("unexpected extra arguments: %v", args),
			ExitCode: 2,
		})
	}

	_, items, metadata, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeCommandError("check", out, errOut, false, err)
	}
	warnStateIssue(errOut, warn)

	entry, ok := currentListEntry(items, metadata)
	if !ok {
		return writeCommandError("check", out, errOut, false, appError{
			Code:     "WORKTREE_NOT_FOUND",
			Message:  "current worktree not found",
			ExitCode: 1,
		})
	}

	branch := entry.item.BranchLabel
	if entry.item.BranchRef == "" {
		branch = "DETACHED"
	}

	taskLabel := entry.meta.Label
	if taskLabel == "" {
		taskLabel = "unlabeled"
	}

	dirtyState := "clean"
	if entry.item.IsDirty {
		dirtyState = "dirty"
	}

	fmt.Fprintf(out, "Path: %s\n", entry.item.Path)
	fmt.Fprintf(out, "Branch: %s\n", branch)
	fmt.Fprintf(out, "Task: %s\n", taskLabel)
	fmt.Fprintf(out, "Dirty: %s\n", dirtyState)

	note, warnings := loadCheckNote(ctx, deps, entry)
	if note != nil && note.Intent != "" {
		fmt.Fprintf(out, "Intent: %s\n", note.Intent)
	}
	for _, warning := range warnings {
		fmt.Fprintln(errOut, warning)
	}
	return 0
}

func currentListEntry(items []worktree.Worktree, metadata map[string]state.WorktreeMetadata) (listEntry, bool) {
	for _, entry := range decorateListEntries(items, metadata) {
		if entry.item.IsCurrent {
			return entry, true
		}
	}
	return listEntry{}, false
}

func loadCheckNote(ctx context.Context, deps Deps, entry listEntry) (*tasknote.Note, []string) {
	var warnings []string
	if entry.item.BranchRef == "" {
		warnings = append(warnings, "warning: current worktree is detached")
	}
	if entry.meta.Label == "" {
		warnings = append(warnings, "warning: current worktree is unlabeled")
		return nil, warnings
	}

	note, err := readTaskNote(ctx, deps, entry.item.Path, entry.meta.Label)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			warnings = append(warnings, "warning: task note missing for labeled worktree")
			return nil, warnings
		}
		warnings = append(warnings, fmt.Sprintf("warning: task note unreadable: %v", err))
		return nil, warnings
	}

	return &note, warnings
}

func readTaskNote(ctx context.Context, deps Deps, worktreePath, label string) (tasknote.Note, error) {
	if label == "" {
		return tasknote.Note{}, fmt.Errorf("task label is required")
	}
	notePath, err := deps.WorktreeGitPath(ctx, worktreePath, "ww/task-note.md")
	if err != nil {
		return tasknote.Note{}, err
	}
	return tasknote.ReadFile(notePath)
}
