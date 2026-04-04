package ui

import (
	"fmt"
	"strings"

	"ww/internal/worktree"
)

func StatusTags(item worktree.Worktree) []string {
	tags := make([]string, 0, 2)
	if item.IsCurrent {
		tags = append(tags, "[CURRENT]")
	}
	if item.IsMerged {
		tags = append(tags, "[MERGED]")
	}
	return tags
}

func StatusText(item worktree.Worktree) string {
	return strings.Join(StatusTags(item), " ")
}

func StatusLabel(item worktree.Worktree) string {
	return StatusText(item)
}

// FormatFileChanges returns colored "+N ~N ?N" string.
// Omits categories with zero count. Returns "" if all zero.
func FormatFileChanges(staged, unstaged, untracked int) string {
	parts := make([]string, 0, 3)
	if staged > 0 {
		parts = append(parts, Green(fmt.Sprintf("+%d", staged)))
	}
	if unstaged > 0 {
		parts = append(parts, Yellow(fmt.Sprintf("~%d", unstaged)))
	}
	if untracked > 0 {
		parts = append(parts, Dim(fmt.Sprintf("?%d", untracked)))
	}
	return strings.Join(parts, " ")
}

// FormatAheadBehind returns colored "↑N ↓N" string.
// Omits directions with zero count. Returns "" if both zero.
func FormatAheadBehind(ahead, behind int) string {
	parts := make([]string, 0, 2)
	if ahead > 0 {
		parts = append(parts, Green(fmt.Sprintf("↑%d", ahead)))
	}
	if behind > 0 {
		parts = append(parts, Red(fmt.Sprintf("↓%d", behind)))
	}
	return strings.Join(parts, " ")
}
