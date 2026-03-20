package git

import (
	"context"
	"fmt"
	"strings"
)

func DefaultBranch(ctx context.Context, runner Runner) (string, error) {
	currentPath, repoKey, err := currentRepoContext(ctx, runner)
	if err != nil {
		return "", err
	}

	root := repositoryRoot(currentPath, repoKey)
	refOut, refErr, err := runner.Run(ctx, "git", "-C", root, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		if branch := remoteHeadBranch(string(refOut)); branch != "" {
			return branch, nil
		}
	} else if !canFallbackFromSymbolicRef(err, refErr) {
		if isNotGitRepository(err, refOut, refErr) {
			return "", ErrNotGitRepository
		}
		return "", commandError("git symbolic-ref refs/remotes/origin/HEAD", err, refErr)
	}

	for _, branch := range []string{"main", "master"} {
		exists, err := localBranchExists(ctx, runner, root, branch)
		if err != nil {
			return "", err
		}
		if exists {
			return branch, nil
		}
	}

	return "", fmt.Errorf("default branch could not be resolved; use --base <branch>")
}

func localBranchExists(ctx context.Context, runner Runner, root, branch string) (bool, error) {
	out, errOut, err := runner.Run(ctx, "git", "-C", root, "branch", "--list", "--format=%(refname:short)", branch)
	if err != nil {
		if isNotGitRepository(err, out, errOut) {
			return false, ErrNotGitRepository
		}
		return false, commandError("git branch --list", err, errOut)
	}
	return strings.TrimSpace(string(out)) == branch, nil
}

func remoteHeadBranch(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "refs/remotes/")
	if trimmed == "" {
		return ""
	}
	_, branch, ok := strings.Cut(trimmed, "/")
	if !ok {
		return ""
	}
	return branch
}

func canFallbackFromSymbolicRef(err error, stderr []byte) bool {
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "exit status 1") {
		return true
	}
	msg := strings.ToLower(string(stderr))
	return strings.Contains(msg, "not a symbolic ref") || strings.Contains(msg, "no such ref")
}
