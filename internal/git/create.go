package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CreateWorktree(ctx context.Context, runner Runner, name string) (string, error) {
	currentPath, repoKey, err := currentRepoContext(ctx, runner)
	if err != nil {
		return "", err
	}

	root := repositoryRoot(currentPath, repoKey)
	target := filepath.Join(root, ".worktrees", name)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", fmt.Errorf("mkdir worktree parent: %w", err)
	}

	createOut, createErr, err := runner.Run(ctx, "git", "-C", root, "worktree", "add", "-b", name, filepath.Join(".worktrees", name), "HEAD")
	if err != nil {
		if isNotGitRepository(err, createOut, createErr) {
			return "", ErrNotGitRepository
		}
		return "", commandError("git worktree add", err, createErr)
	}

	return target, nil
}

func repositoryRoot(currentPath, repoKey string) string {
	if filepath.Base(repoKey) == ".git" {
		return filepath.Dir(repoKey)
	}
	return filepath.Clean(currentPath)
}

func commandError(prefix string, err error, stderr []byte) error {
	msg := strings.TrimSpace(string(stderr))
	if msg == "" {
		return fmt.Errorf("%s: %w", prefix, err)
	}
	return fmt.Errorf("%s: %w: %s", prefix, err, msg)
}
