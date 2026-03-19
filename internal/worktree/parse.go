package worktree

import (
	"errors"
	"fmt"
	"strings"
)

func ParsePorcelainZ(raw string) ([]Worktree, error) {
	records := strings.Split(raw, "\x00")
	items := make([]Worktree, 0)

	var current *Worktree
	for _, token := range records {
		if token == "" {
			if current != nil {
				items = append(items, *current)
				current = nil
			}
			continue
		}

		key, value, ok := strings.Cut(token, " ")
		if !ok {
			switch token {
			case "detached":
				if current == nil {
					return nil, errors.New("detached token before worktree")
				}
				current.IsDetached = true
				current.BranchLabel = "(detached)"
				current.BranchRef = ""
				continue
			case "bare":
				continue
			}
			return nil, fmt.Errorf("malformed token: %q", token)
		}

		switch key {
		case "worktree":
			if current != nil {
				items = append(items, *current)
			}
			current = &Worktree{Path: value}
		case "branch":
			if current == nil {
				return nil, errors.New("branch token before worktree")
			}
			current.BranchRef = value
			current.BranchLabel = branchLabel(value)
		case "HEAD":
		case "locked", "prunable", "bare":
		default:
			return nil, fmt.Errorf("unsupported token: %q", key)
		}
	}

	if current != nil {
		items = append(items, *current)
	}

	return items, nil
}

func branchLabel(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}
