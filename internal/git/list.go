package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"ww/internal/worktree"
)

var ErrNotGitRepository = errors.New("not a git repository")

type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout []byte, stderr []byte, err error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func ListWorktrees(ctx context.Context, runner Runner) (string, []worktree.Worktree, error) {
	currentPath, repoKey, err := currentRepoContext(ctx, runner)
	if err != nil {
		return "", nil, err
	}

	worktreeOut, worktreeErr, err := runner.Run(ctx, "git", "-C", currentPath, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		if isNotGitRepository(err, worktreeOut, worktreeErr) {
			return "", nil, ErrNotGitRepository
		}
		return "", nil, fmt.Errorf("git worktree list: %w", err)
	}

	items, err := worktree.ParsePorcelainZ(string(worktreeOut))
	if err != nil {
		return "", nil, err
	}

	for i := range items {
		items[i].IsCurrent = filepath.Clean(items[i].Path) == filepath.Clean(currentPath)
	}
	annotateCreationTimes(items)

	return repoKey, items, nil
}

func CurrentRepoKey(ctx context.Context, runner Runner) (string, error) {
	_, repoKey, err := currentRepoContext(ctx, runner)
	if err != nil {
		return "", err
	}
	return repoKey, nil
}

func currentRepoContext(ctx context.Context, runner Runner) (string, string, error) {
	rootOut, rootErr, err := runner.Run(ctx, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		if isNotGitRepository(err, rootOut, rootErr) {
			return "", "", ErrNotGitRepository
		}
		return "", "", fmt.Errorf("git rev-parse --show-toplevel: %w", err)
	}

	currentPath := strings.TrimSpace(string(rootOut))
	commonDirOut, commonDirErr, err := runner.Run(ctx, "git", "-C", currentPath, "rev-parse", "--git-common-dir")
	if err != nil {
		if isNotGitRepository(err, commonDirOut, commonDirErr) {
			return "", "", ErrNotGitRepository
		}
		return "", "", fmt.Errorf("git rev-parse --git-common-dir: %w", err)
	}

	return currentPath, cleanPath(currentPath, commonDirOut), nil
}

func isNotGitRepository(err error, stdout []byte, stderr []byte) bool {
	combined := strings.ToLower(string(stdout) + " " + string(stderr) + " " + err.Error())
	return strings.Contains(combined, "not a git repository")
}

func cleanPath(base string, raw []byte) string {
	return cleanPathString(base, strings.TrimSpace(string(raw)))
}

func cleanPathString(base, raw string) string {
	if raw == "" {
		return ""
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw)
	}
	return filepath.Clean(filepath.Join(base, raw))
}
