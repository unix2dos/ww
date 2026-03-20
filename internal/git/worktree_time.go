package git

import (
	"os"

	"ww/internal/worktree"
)

func annotateCreationTimes(items []worktree.Worktree) {
	for i := range items {
		items[i].CreatedAt = worktreeCreatedAt(items[i].Path)
	}
}

func worktreeCreatedAt(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}

	if createdAt, ok := birthTimeUnixNano(info); ok {
		return createdAt
	}
	return info.ModTime().UnixNano()
}
