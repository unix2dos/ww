package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"ww/internal/worktree"
)

var ErrFzfNotInstalled = errors.New("fzf not installed")
var ErrSelectionCanceled = errors.New("selection canceled")

type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, name string, stdin []byte, args ...string) (stdout []byte, stderr []byte, err error)
}

type ExecRunner struct{}

func (ExecRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (ExecRunner) Run(ctx context.Context, name string, stdin []byte, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func SelectWorktreeWithFzf(ctx context.Context, items []worktree.Worktree, runner Runner) (worktree.Worktree, error) {
	if _, err := runner.LookPath("fzf"); err != nil {
		return worktree.Worktree{}, ErrFzfNotInstalled
	}

	input := formatFzfCandidates(items)
	stdout, _, err := runner.Run(ctx, "fzf", input,
		"--ansi",
		"--no-sort",
		"--tac",
		"--delimiter=\t",
		"--nth=2..",
		"--pointer=*",
		fmt.Sprintf("--bind=load:pos(%d)", initialFzfPosition(items)),
		fmt.Sprintf("--prompt=Select [1-%d]> ", len(items)),
	)
	if err != nil {
		if isCanceled(err) {
			return worktree.Worktree{}, ErrSelectionCanceled
		}
		return worktree.Worktree{}, fmt.Errorf("fzf: %w", err)
	}

	selection := strings.TrimSpace(string(stdout))
	if selection == "" {
		return worktree.Worktree{}, ErrSelectionCanceled
	}

	index, err := parseFzfSelection(selection)
	if err != nil {
		return worktree.Worktree{}, err
	}

	for i := range items {
		if items[i].Index == index {
			return items[i], nil
		}
	}

	return worktree.Worktree{}, fmt.Errorf("selected worktree index %d not found", index)
}

func formatFzfCandidates(items []worktree.Worktree) []byte {
	var buf strings.Builder
	branchWidth := HumanBranchWidth(items)
	abWidth := aheadBehindWidth(items)
	fcWidth := fileChangesWidth(items)
	for _, item := range items {
		marker := " "
		if item.IsCurrent {
			marker = Bold("★")
		}

		statusStr := colorizeStatus(item)

		branchStr := item.BranchLabel
		if item.IsMerged && !item.IsCurrent {
			branchStr = Dim(branchStr)
		}

		abStr := FormatAheadBehind(item.Ahead, item.Behind)
		fcStr := FormatFileChanges(item.Staged, item.Unstaged, item.Untracked)

		pathStr := item.Path
		if item.IsMerged && !item.IsCurrent {
			pathStr = Dim(pathStr)
		}

		fmt.Fprintf(&buf, "%d\t%s %s\t%s\t%s\t%s\t%s\n",
			item.Index,
			marker,
			PadRight(statusStr, humanStatusWidth),
			PadRight(branchStr, branchWidth),
			PadRight(abStr, abWidth),
			PadRight(fcStr, fcWidth),
			pathStr,
		)
	}
	return []byte(buf.String())
}

func initialFzfPosition(items []worktree.Worktree) int {
	for i, item := range items {
		if item.IsCurrent {
			return len(items) - i
		}
	}
	return 1
}

func parseFzfSelection(selection string) (int, error) {
	fields := strings.SplitN(selection, "\t", 2)
	if len(fields) == 0 || fields[0] == "" {
		return 0, fmt.Errorf("invalid fzf selection: %q", selection)
	}

	index, err := strconv.Atoi(strings.TrimSpace(fields[0]))
	if err != nil || index <= 0 {
		return 0, fmt.Errorf("invalid fzf selection: %q", selection)
	}
	return index, nil
}

func isCanceled(err error) bool {
	type exitCoder interface {
		ExitCode() int
	}

	var code exitCoder
	if errors.As(err, &code) {
		return code.ExitCode() == 130
	}
	return false
}
