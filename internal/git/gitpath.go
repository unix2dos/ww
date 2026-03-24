package git

import (
	"context"
	"fmt"
)

func WorktreeGitPath(ctx context.Context, runner Runner, worktreePath string, rel string) (string, error) {
	out, errOut, err := runner.Run(ctx, "git", "-C", worktreePath, "rev-parse", "--git-path", rel)
	if err != nil {
		if isNotGitRepository(err, out, errOut) {
			return "", ErrNotGitRepository
		}
		return "", commandError("git rev-parse --git-path", err, errOut)
	}

	path := cleanPath(worktreePath, out)
	if path == "" {
		return "", fmt.Errorf("git rev-parse --git-path returned empty path")
	}
	return path, nil
}
