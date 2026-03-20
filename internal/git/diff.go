package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type DiffReport struct {
	CurrentBranch string
	TargetBranch  string
	Ahead         int
	Behind        int
	ChangedFiles  int
	Insertions    int
	Deletions     int
	Commits       []DiffCommit
	Files         []DiffFile
}

type DiffCommit struct {
	Hash    string
	Subject string
}

type DiffFile struct {
	Path       string
	Insertions int
	Deletions  int
}

func DiffSummary(ctx context.Context, runner Runner, targetBranch string) (DiffReport, error) {
	root, currentBranch, err := currentBranchContext(ctx, runner)
	if err != nil {
		return DiffReport{}, err
	}
	if targetBranch == "" {
		return DiffReport{}, fmt.Errorf("target branch is required")
	}

	behind, ahead, err := aheadBehind(ctx, runner, root, targetBranch, currentBranch)
	if err != nil {
		return DiffReport{}, err
	}
	commits, err := diffCommits(ctx, runner, root, targetBranch, currentBranch)
	if err != nil {
		return DiffReport{}, err
	}
	files, insertions, deletions, err := diffFiles(ctx, runner, root, targetBranch, currentBranch)
	if err != nil {
		return DiffReport{}, err
	}

	return DiffReport{
		CurrentBranch: currentBranch,
		TargetBranch:  targetBranch,
		Ahead:         ahead,
		Behind:        behind,
		ChangedFiles:  len(files),
		Insertions:    insertions,
		Deletions:     deletions,
		Commits:       commits,
		Files:         files,
	}, nil
}

func DiffPatch(ctx context.Context, runner Runner, targetBranch string) (string, error) {
	root, currentBranch, err := currentBranchContext(ctx, runner)
	if err != nil {
		return "", err
	}
	if targetBranch == "" {
		return "", fmt.Errorf("target branch is required")
	}

	out, errOut, err := runner.Run(ctx, "git", "-C", root, "diff", fmt.Sprintf("%s...%s", targetBranch, currentBranch))
	if err != nil {
		if isNotGitRepository(err, out, errOut) {
			return "", ErrNotGitRepository
		}
		return "", commandError("git diff", err, errOut)
	}
	return string(out), nil
}

func currentBranchContext(ctx context.Context, runner Runner) (string, string, error) {
	currentPath, repoKey, err := currentRepoContext(ctx, runner)
	if err != nil {
		return "", "", err
	}

	out, errOut, err := runner.Run(ctx, "git", "-C", currentPath, "branch", "--show-current")
	if err != nil {
		if isNotGitRepository(err, out, errOut) {
			return "", "", ErrNotGitRepository
		}
		return "", "", commandError("git branch --show-current", err, errOut)
	}

	currentBranch := strings.TrimSpace(string(out))
	if currentBranch == "" {
		return "", "", fmt.Errorf("current branch is unavailable; detached HEAD is not supported")
	}

	return repositoryRoot(currentPath, repoKey), currentBranch, nil
}

func aheadBehind(ctx context.Context, runner Runner, root, targetBranch, currentBranch string) (int, int, error) {
	out, errOut, err := runner.Run(ctx, "git", "-C", root, "rev-list", "--left-right", "--count", fmt.Sprintf("%s...%s", targetBranch, currentBranch))
	if err != nil {
		if isNotGitRepository(err, out, errOut) {
			return 0, 0, ErrNotGitRepository
		}
		return 0, 0, commandError("git rev-list --left-right --count", err, errOut)
	}

	fields := strings.Fields(string(out))
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected git rev-list output: %q", strings.TrimSpace(string(out)))
	}
	behind, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse behind count: %w", err)
	}
	ahead, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse ahead count: %w", err)
	}
	return behind, ahead, nil
}

func diffCommits(ctx context.Context, runner Runner, root, targetBranch, currentBranch string) ([]DiffCommit, error) {
	out, errOut, err := runner.Run(ctx, "git", "-C", root, "log", "--format=%H%x09%s", fmt.Sprintf("%s..%s", targetBranch, currentBranch))
	if err != nil {
		if isNotGitRepository(err, out, errOut) {
			return nil, ErrNotGitRepository
		}
		return nil, commandError("git log", err, errOut)
	}

	if strings.TrimSpace(string(out)) == "" {
		return nil, nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	commits := make([]DiffCommit, 0, len(lines))
	for _, line := range lines {
		hash, subject, ok := strings.Cut(line, "\t")
		if !ok {
			return nil, fmt.Errorf("unexpected git log output: %q", line)
		}
		commits = append(commits, DiffCommit{Hash: hash, Subject: subject})
	}
	return commits, nil
}

func diffFiles(ctx context.Context, runner Runner, root, targetBranch, currentBranch string) ([]DiffFile, int, int, error) {
	out, errOut, err := runner.Run(ctx, "git", "-C", root, "diff", "--numstat", fmt.Sprintf("%s...%s", targetBranch, currentBranch))
	if err != nil {
		if isNotGitRepository(err, out, errOut) {
			return nil, 0, 0, ErrNotGitRepository
		}
		return nil, 0, 0, commandError("git diff --numstat", err, errOut)
	}

	if strings.TrimSpace(string(out)) == "" {
		return nil, 0, 0, nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	files := make([]DiffFile, 0, len(lines))
	totalInsertions := 0
	totalDeletions := 0
	for _, line := range lines {
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 {
			return nil, 0, 0, fmt.Errorf("unexpected git diff --numstat output: %q", line)
		}

		insertions := parseNumstat(fields[0])
		deletions := parseNumstat(fields[1])
		totalInsertions += insertions
		totalDeletions += deletions
		files = append(files, DiffFile{
			Path:       fields[2],
			Insertions: insertions,
			Deletions:  deletions,
		})
	}

	return files, totalInsertions, totalDeletions, nil
}

func parseNumstat(raw string) int {
	if raw == "-" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}
