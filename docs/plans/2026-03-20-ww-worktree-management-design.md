# WW Worktree Management Design

**Goal:** Make worktree discovery stable and add safe worktree cleanup without breaking the shell-first workflow.

## Product Decisions

- Worktree ordering is stable and sorted by `BranchLabel` ascending.
- The current worktree is not moved to the top. It is rendered as `ACTIVE` in every view.
- MRU state is no longer used for display order or index assignment.
- `ww rm` lists every non-current worktree, shows safety status, and requires confirmation before execution.
- Dirty worktrees are refused by default; `--force` only applies to worktree removal.
- Branch deletion is attempted only when the branch is merged into the effective base branch.
- The effective base branch defaults to the repository default branch at command execution time and can be overridden with `--base` for `ww rm`.

## Default Branch Resolution

`ww` needs a deterministic way to find the repository default branch:

1. Resolve `refs/remotes/origin/HEAD` if present.
2. Fall back to a local `main` branch if present.
3. Fall back to a local `master` branch if present.
4. If none of the above exist, return a descriptive error and ask the caller to specify `--base`.

This keeps the behavior aligned with hosted Git workflows without introducing new metadata files.

## Architecture

### App Layer

`internal/app/run.go` remains the command router. It will:

- add the `rm` command entry point
- stop loading/touching MRU state for display ordering
- route new confirmation and output formatting flows

### Git Layer

New Git helpers will live under `internal/git` to keep shell commands out of the app layer:

- default branch detection
- branch cleanliness / merged checks
- worktree removal
- optional branch deletion

### UI Layer

`internal/ui` keeps all presentation rules:

- stable `ACTIVE` status rendering in list, fallback menu, TUI, and fzf input
- confirmation prompt helpers for `rm`

## Error Handling

- Attempting to remove the current worktree is a user error.
- Dirty worktrees without `--force` are rejected before any destructive action.
- Unmerged branches are preserved, and the result explicitly states that only the worktree was removed.

## Testing Strategy

- Update normalization tests to verify stable branch-name ordering and index assignment.
- Update UI tests to verify `ACTIVE` replaces the current-worktree `*`.
- Add Git unit tests for default branch resolution and removal safety checks.
- Add app-layer tests for `ww rm` argument handling and output.
- Add e2e coverage for stable list ordering and safe removal.
