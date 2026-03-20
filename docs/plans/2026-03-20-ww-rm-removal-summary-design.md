# WW Remove Summary Design

**Goal:** Make `ww rm` understandable without requiring users to translate Git states such as merged, detached, and dirty into deletion risk.

## Product Decisions

- The interactive candidate list should optimize for picking a target, not explaining Git internals.
- Candidate rows should use human-language risk groupings instead of raw state tags such as `DELETE_BRANCH`, `KEEP_BRANCH`, `DIRTY`, and `DETACHED`.
- The list should be split into sections in this order: `Safe to delete`, `Review before deleting`, `Not safe to delete`.
- Section membership depends on the actual command mode:
  - clean + merged branch => safe
  - dirty without `--force` => not safe
  - everything else => review
- Emoji can be used as a visual accelerator, but every risk state must still be explained with plain text.

## Removal Summary Card

After the user chooses a candidate, `ww rm` should print a summary card before asking for confirmation.

The summary card should answer four questions in plain language:

1. What will be removed
2. What will be kept
3. What the risk is
4. What the next step is

The card should use three severity states:

- `✅ Safe to delete`
- `⚠️ Review before deleting`
- `🛑 Not safe to delete`

## Confirmation Rules

- Safe and review states continue to the normal confirmation prompt.
- Not-safe states should not prompt for confirmation when the command cannot proceed safely.
- Dirty worktrees without `--force` should stop after the summary card and tell the user to commit, stash, or rerun with `--force` to discard changes.

## Copy Rules

- Paths belong in the summary card, not the candidate list.
- Branch deletion should be described as an effect, not a status label.
- Detached worktrees should explain that no branch will be deleted and that the worktree is not on a branch.
- Unmerged branch worktrees should explain that the worktree will be removed but the branch will be kept.

## Testing Strategy

- Add app tests for grouped candidate rendering and summary-card copy.
- Add app tests for dirty worktrees stopping before confirmation unless `--force` is present.
- Update e2e coverage to assert the summary-card flow for merged and dirty worktrees.
- Refresh demo/doc artifacts that still assert the old raw status labels.
