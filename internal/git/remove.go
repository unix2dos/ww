package git

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"ww/internal/worktree"
)

type RemovalPreview struct {
	Worktree     worktree.Worktree
	BaseBranch   string
	Dirty        bool
	BranchMerged bool
	DeleteBranch bool
}

type RemoveOptions struct {
	BaseBranch string
	Force      bool
}

type RemoveResult struct {
	WorktreePath     string
	Branch           string
	BaseBranch       string
	RemovedWorktree  bool
	DeletedBranch    bool
	KeptBranchReason string
}

func PreviewRemoval(ctx context.Context, runner Runner, item worktree.Worktree, baseBranch string) (RemovalPreview, error) {
	if baseBranch == "" && item.BranchRef != "" {
		return RemovalPreview{}, fmt.Errorf("base branch is required for removal preview")
	}

	dirty, err := worktreeDirty(ctx, runner, item.Path)
	if err != nil {
		return RemovalPreview{}, err
	}

	preview := RemovalPreview{
		Worktree:   item,
		BaseBranch: baseBranch,
		Dirty:      dirty,
	}
	if item.BranchRef == "" {
		return preview, nil
	}

	merged, err := branchMergedInto(ctx, runner, item.Path, item.BranchLabel, baseBranch)
	if err != nil {
		return RemovalPreview{}, err
	}
	preview.BranchMerged = merged
	preview.DeleteBranch = merged && item.BranchLabel != baseBranch
	return preview, nil
}

func RemoveWorktree(ctx context.Context, runner Runner, item worktree.Worktree, opts RemoveOptions) (RemoveResult, error) {
	if item.IsCurrent {
		return RemoveResult{}, fmt.Errorf("cannot remove the active worktree")
	}
	if opts.BaseBranch == "" && item.BranchRef != "" {
		return RemoveResult{}, fmt.Errorf("base branch is required for removal")
	}

	preview, err := PreviewRemoval(ctx, runner, item, opts.BaseBranch)
	if err != nil {
		return RemoveResult{}, err
	}
	if preview.Dirty && !opts.Force {
		return RemoveResult{}, fmt.Errorf("worktree has uncommitted changes; rerun with --force")
	}

	currentPath, repoKey, err := currentRepoContext(ctx, runner)
	if err != nil {
		return RemoveResult{}, err
	}
	root := repositoryRoot(currentPath, repoKey)

	args := []string{"-C", root, "worktree", "remove"}
	if opts.Force {
		args = append(args, "-f")
	}
	args = append(args, item.Path)

	removeOut, removeErr, err := runner.Run(ctx, "git", args...)
	if err != nil {
		if isNotGitRepository(err, removeOut, removeErr) {
			return RemoveResult{}, ErrNotGitRepository
		}
		return RemoveResult{}, commandError("git worktree remove", err, removeErr)
	}

	result := RemoveResult{
		WorktreePath:    item.Path,
		Branch:          item.BranchLabel,
		BaseBranch:      opts.BaseBranch,
		RemovedWorktree: true,
	}

	if item.BranchRef == "" {
		result.KeptBranchReason = "detached"
		return result, nil
	}
	if !preview.DeleteBranch {
		if item.BranchLabel == opts.BaseBranch {
			result.KeptBranchReason = "base branch"
			return result, nil
		}
	}
	if !preview.BranchMerged {
		result.KeptBranchReason = "not merged"
		return result, nil
	}

	deleteOut, deleteErr, err := runner.Run(ctx, "git", "-C", root, "branch", "-d", item.BranchLabel)
	if err != nil {
		if isNotGitRepository(err, deleteOut, deleteErr) {
			return result, ErrNotGitRepository
		}
		return result, commandError("git branch -d", err, deleteErr)
	}

	result.DeletedBranch = true
	return result, nil
}

func worktreeDirty(ctx context.Context, runner Runner, path string) (bool, error) {
	out, errOut, err := runner.Run(ctx, "git", "-C", path, "status", "--porcelain", "--", ".", ":(exclude).worktrees")
	if err != nil {
		if isNotGitRepository(err, out, errOut) {
			return false, ErrNotGitRepository
		}
		return false, commandError("git status --porcelain -- . :(exclude).worktrees", err, errOut)
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

func branchMergedInto(ctx context.Context, runner Runner, path, branch, baseBranch string) (bool, error) {
	out, errOut, err := runner.Run(ctx, "git", "-C", path, "branch", "--format=%(refname:short)", "--merged", baseBranch)
	if err != nil {
		if isNotGitRepository(err, out, errOut) {
			return false, ErrNotGitRepository
		}
		return false, commandError("git branch --merged", err, errOut)
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == branch {
			return true, nil
		}
	}
	return false, nil
}
