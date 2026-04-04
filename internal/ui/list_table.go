package ui

import (
	"fmt"
	"strings"

	"ww/internal/worktree"
)

const listIndexWidth = len(humanIndexHeader)
const listPathWidth = 48
const listABHeader = "AHEAD/BEHIND"
const listChangesHeader = "CHANGES"
const listABWidth = len(listABHeader)     // 12
const listChangesWidth = len(listChangesHeader) // 7

type ListTableEntry struct {
	Worktree worktree.Worktree
	Detail   string
}

func FormatListTable(entries []ListTableEntry) string {
	if len(entries) == 0 {
		return ""
	}

	branchWidth := listBranchWidth(entries)
	var buf strings.Builder

	buf.WriteString(listTableBorder("┌", "┬", "┐", branchWidth))
	buf.WriteByte('\n')
	buf.WriteString(listTableRow(humanIndexHeader, humanStatusHeader, humanBranchHeader, listABHeader, listChangesHeader, humanPathHeader, branchWidth))
	buf.WriteByte('\n')
	buf.WriteString(listTableBorder("├", "┼", "┤", branchWidth))
	buf.WriteByte('\n')

	for i, entry := range entries {
		for _, row := range listTableRows(entry, branchWidth) {
			buf.WriteString(row)
			buf.WriteByte('\n')
		}
		if i == len(entries)-1 {
			buf.WriteString(listTableBorder("└", "┴", "┘", branchWidth))
		} else {
			buf.WriteString(listTableBorder("├", "┼", "┤", branchWidth))
			buf.WriteByte('\n')
		}
	}

	return buf.String()
}

func listBranchWidth(entries []ListTableEntry) int {
	items := make([]worktree.Worktree, 0, len(entries))
	for _, entry := range entries {
		items = append(items, entry.Worktree)
	}
	return normalizedBranchWidth(HumanBranchWidth(items))
}

func listTableRows(entry ListTableEntry, branchWidth int) []string {
	pathContent := entry.Worktree.Path
	if entry.Detail != "" {
		pathContent += "\n" + entry.Detail
	}

	pathLines := wrapCell(pathContent, listPathWidth)
	rows := make([]string, 0, len(pathLines))
	for i, pathLine := range pathLines {
		index := ""
		status := ""
		branch := ""
		ab := ""
		changes := ""
		if i == 0 {
			index = fmt.Sprintf("%d", entry.Worktree.Index)
			status = StatusText(entry.Worktree)
			branch = entry.Worktree.BranchLabel
			ab = FormatAheadBehind(entry.Worktree.Ahead, entry.Worktree.Behind)
			changes = FormatFileChanges(entry.Worktree.Staged, entry.Worktree.Unstaged, entry.Worktree.Untracked)
		}
		rows = append(rows, listTableRow(index, status, branch, ab, changes, pathLine, branchWidth))
	}
	return rows
}

func listTableRow(index, status, branch, ab, changes, path string, branchWidth int) string {
	return fmt.Sprintf("│ %-*s │ %-*s │ %-*s │ %s │ %s │ %-*s │",
		listIndexWidth, index,
		humanStatusWidth, status,
		branchWidth, branch,
		PadRight(ab, listABWidth),
		PadRight(changes, listChangesWidth),
		listPathWidth, path,
	)
}

func listTableBorder(left, mid, right string, branchWidth int) string {
	return left +
		strings.Repeat("─", listIndexWidth+2) + mid +
		strings.Repeat("─", humanStatusWidth+2) + mid +
		strings.Repeat("─", branchWidth+2) + mid +
		strings.Repeat("─", listABWidth+2) + mid +
		strings.Repeat("─", listChangesWidth+2) + mid +
		strings.Repeat("─", listPathWidth+2) +
		right
}

func wrapCell(text string, width int) []string {
	if text == "" {
		return []string{""}
	}

	var lines []string
	for _, rawLine := range strings.Split(text, "\n") {
		if rawLine == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, wrapLine(rawLine, width)...)
	}
	return lines
}

func wrapLine(text string, width int) []string {
	if width <= 0 || len(text) <= width {
		return []string{text}
	}

	var lines []string
	remaining := text
	for len(remaining) > width {
		cut, trimLeading := findWrapPoint(remaining, width)
		lines = append(lines, remaining[:cut])
		remaining = remaining[cut:]
		if trimLeading {
			remaining = strings.TrimLeft(remaining, " ")
		}
	}
	lines = append(lines, remaining)
	return lines
}

func findWrapPoint(text string, width int) (int, bool) {
	if len(text) <= width {
		return len(text), false
	}

	for i := width - 1; i >= 0; i-- {
		switch text[i] {
		case '/':
			return i + 1, false
		case '-', '_':
			return i + 1, false
		case ' ':
			return i, true
		}
	}

	return width, false
}
