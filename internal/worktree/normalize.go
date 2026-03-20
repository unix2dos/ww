package worktree

import (
	"sort"
)

func Normalize(items []Worktree) []Worktree {
	if len(items) == 0 {
		return nil
	}

	out := make([]Worktree, len(items))
	copy(out, items)

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].BranchLabel != out[j].BranchLabel {
			return out[i].BranchLabel < out[j].BranchLabel
		}
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Index < out[j].Index
	})

	for i := range out {
		out[i].Index = i + 1
	}

	return out
}
