package worktree

type Worktree struct {
	Path        string
	BranchRef   string
	BranchLabel string
	IsDetached  bool
	IsCurrent   bool
	LastUsedAt  int64
	Index       int
}
