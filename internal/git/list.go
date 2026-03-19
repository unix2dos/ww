package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"wt/internal/worktree"
)

var ErrNotGitRepository = errors.New("not a git repository")

type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

func ListWorktrees(ctx context.Context, runner Runner) ([]worktree.Worktree, error) {
	rootOut, err := runner.Run(ctx, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		if isNotGitRepository(err, rootOut) {
			return nil, ErrNotGitRepository
		}
		return nil, fmt.Errorf("git rev-parse --show-toplevel: %w", err)
	}

	currentPath := strings.TrimSpace(string(rootOut))
	worktreeOut, err := runner.Run(ctx, "git", "-C", currentPath, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		if isNotGitRepository(err, worktreeOut) {
			return nil, ErrNotGitRepository
		}
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	items, err := worktree.ParsePorcelainZ(string(worktreeOut))
	if err != nil {
		return nil, err
	}

	return worktree.Normalize(items, currentPath), nil
}

func isNotGitRepository(err error, output []byte) bool {
	combined := strings.ToLower(string(output) + " " + err.Error())
	return strings.Contains(combined, "not a git repository")
}
