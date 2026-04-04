package git

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

// FileChangeCounts returns staged, unstaged, and untracked file counts.
func FileChangeCounts(ctx context.Context, runner Runner, path string) (staged, unstaged, untracked int, err error) {
	out, errOut, err := runner.Run(ctx, "git", "-C", path, "status", "--porcelain", "--", ".", ":(exclude).worktrees")
	if err != nil {
		if isNotGitRepository(err, out, errOut) {
			return 0, 0, 0, ErrNotGitRepository
		}
		return 0, 0, 0, commandError("git status --porcelain", err, errOut)
	}

	for _, line := range strings.Split(string(bytes.TrimSpace(out)), "\n") {
		if len(line) < 2 {
			continue
		}
		x, y := line[0], line[1]
		if x == '?' && y == '?' {
			untracked++
			continue
		}
		if x != ' ' && x != '?' {
			staged++
		}
		if y != ' ' && y != '?' {
			unstaged++
		}
	}
	return staged, unstaged, untracked, nil
}

// AheadBehind returns how many commits branch is ahead of and behind baseBranch.
func AheadBehind(ctx context.Context, runner Runner, path, branch, baseBranch string) (ahead, behind int, err error) {
	out, errOut, err := runner.Run(ctx, "git", "-C", path, "rev-list", "--left-right", "--count", branch+"..."+baseBranch)
	if err != nil {
		if isNotGitRepository(err, out, errOut) {
			return 0, 0, ErrNotGitRepository
		}
		return 0, 0, commandError("git rev-list --left-right --count", err, errOut)
	}

	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %q", string(out))
	}

	ahead, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse ahead count: %w", err)
	}
	behind, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse behind count: %w", err)
	}
	return ahead, behind, nil
}

// BranchMergedIntoBase checks if branch has been fully merged into baseBranch.
func BranchMergedIntoBase(ctx context.Context, runner Runner, path, branch, baseBranch string) (bool, error) {
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
