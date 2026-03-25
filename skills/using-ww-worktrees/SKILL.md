---
name: using-ww-worktrees
description: Use in repositories that ship ww-helper to create, inspect, and clean up isolated worktrees through the machine-readable CLI instead of raw git worktree commands
---

# Using ww Worktrees

Use this skill when the current repository ships `ww-helper` and you need an isolated workspace for implementation or review work.

**Announce at start:** "I'm using the using-ww-worktrees skill to manage an isolated workspace through ww-helper."

## Core Rule

`ww` is the human shell entrypoint. `ww-helper` is the stable machine-readable interface for agents. When `ww-helper` covers a workflow, use it instead of raw `git worktree` commands.

## Preconditions

1. Confirm that `ww-helper` is available in `PATH`.
2. Read the repository `AGENTS.md` if present.
3. If `ww-helper` is not available, fall back to a generic git-worktree skill or direct Git commands.

## Standard Flow

### 1. Create a Worktree

```bash
ww-helper new-path --json --label agent:codex --ttl 24h feat-demo
```

Read `data.worktree_path` from the JSON response, then change into that directory.

### 2. Inspect Existing Worktrees

```bash
ww-helper list --json
ww-helper switch-path feat-demo
```

Use `switch-path` when you need a path without the human shell behavior of `ww`.

### 3. Verify the Baseline

Run the repository's setup and tests after creation.

For the `ww` repository:

```bash
go test ./...
```

If the baseline fails, report it before making feature changes.

### 4. Remove a Worktree

```bash
ww-helper rm --json --non-interactive feat-demo
```

### 5. Review Cleanup Candidates

```bash
ww-helper gc --ttl-expired --idle 7d --dry-run --json
```

Never run `gc` without at least one explicit selector.

## Safety Rules

- Prefer `--json` over human-readable output.
- Do not script `ww`; it exists for humans and may change the current shell directory.
- Do not call `git worktree add`, `git worktree remove`, or `git worktree list` unless you are debugging `ww-helper` itself or implementing a missing `ww-helper` capability.
- Do not remove dirty worktrees without explicit approval.
- Do not remove the active worktree.
- For repository-local storage changes, verify the ignore rule with a path under `.worktrees/`, for example: `git check-ignore -q .worktrees/probe`

## Recommended Conventions

- Use short branch slugs such as `feat-demo`, `fix-list-json`, or `docs-agent-guide`.
- Set `--label` to the agent or runtime, and append task context when helpful.
- Set a TTL on temporary agent worktrees so cleanup can stay explicit and low-risk.
