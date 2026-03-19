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
		if out[i].IsCurrent != out[j].IsCurrent {
			return out[i].IsCurrent
		}
		iHasMRU := out[i].LastUsedAt > 0
		jHasMRU := out[j].LastUsedAt > 0
		if iHasMRU != jHasMRU {
			return iHasMRU
		}
		if iHasMRU && out[i].LastUsedAt != out[j].LastUsedAt {
			return out[i].LastUsedAt > out[j].LastUsedAt
		}
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
