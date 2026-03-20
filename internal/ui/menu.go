package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"ww/internal/worktree"
)

func RenderMenu(w io.Writer, items []worktree.Worktree) {
	for _, item := range items {
		fmt.Fprintln(w, formatMenuRow(item))
	}
	fmt.Fprint(w, "Select a worktree [number]: ")
}

func formatMenuRow(item worktree.Worktree) string {
	return fmt.Sprintf("[%d] %-6s %s %s", item.Index, worktreeStatus(item), item.BranchLabel, item.Path)
}

func formatTUIRow(item worktree.Worktree, active bool) string {
	prefix := " "
	if active {
		prefix = "*"
	}
	return fmt.Sprintf("%s %s", prefix, formatMenuRow(item))
}

func worktreeStatus(item worktree.Worktree) string {
	if item.IsCurrent {
		return "ACTIVE"
	}
	return ""
}

func ReadSelection(in io.Reader, errOut io.Writer, max int) (int, error) {
	reader := bufio.NewReader(in)

	for {
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if errors.Is(err, io.EOF) {
				return 0, io.EOF
			}
			fmt.Fprintln(errOut, "empty selection")
			continue
		}

		index, convErr := strconv.Atoi(trimmed)
		if convErr != nil || index <= 0 || index > max {
			fmt.Fprintf(errOut, "invalid worktree selection: %q\n", trimmed)
			if errors.Is(err, io.EOF) {
				return 0, io.EOF
			}
			continue
		}

		return index, nil
	}
}
