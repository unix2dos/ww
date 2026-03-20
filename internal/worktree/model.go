package worktree

type Worktree struct {
	Path        string
	BranchRef   string
	BranchLabel string
	IsDetached  bool
	IsCurrent   bool
	CreatedAt   int64
	LastUsedAt  int64
	Index       int
}
