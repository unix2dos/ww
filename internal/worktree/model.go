package worktree

type Worktree struct {
	Path        string
	BranchRef   string
	BranchLabel string
	IsDetached  bool
	IsCurrent   bool
	IsDirty     bool
	CreatedAt   int64
	LastUsedAt  int64
	Index       int

	// Branch-level status
	IsMerged bool
	Ahead    int
	Behind   int

	// Worktree-level file changes
	Staged    int
	Unstaged  int
	Untracked int
}
