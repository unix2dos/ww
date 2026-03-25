# Agent Workflow

This repository ships two entrypoints:

- `ww`: human shell function that can change the current shell directory
- `ww-helper`: machine-readable CLI for automation and coding agents

## Required Behavior

- Use `ww-helper` for automation. Do not script `ww` unless you are testing the human shell UX.
- Prefer `--json` whenever a `ww-helper` command supports it.
- Do not call raw `git worktree add`, `git worktree remove`, or `git worktree list` when an equivalent `ww-helper` command exists.
- Keep repository-managed worktrees under `./.worktrees/`. This repository ignores paths below that directory.
- When you need to verify the ignore rule, check a path under the directory, for example: `git check-ignore -q .worktrees/probe`

## Standard Commands

- Create: `ww-helper new-path --json --label agent:codex --ttl 24h <branch>`
- List: `ww-helper list --json`
- Resolve a target path: `ww-helper switch-path [<target>]`
- Remove non-interactively: `ww-helper rm --json --non-interactive <target>`
- Review cleanup candidates: `ww-helper gc --ttl-expired --idle 7d --dry-run --json`

## Naming Conventions

- Use a short task slug for branch names: `feat-...`, `fix-...`, `docs-...`, `chore-...`
- Include the agent or runtime in `--label`, for example: `agent:codex`
- Add task context to the label when useful, for example: `agent:codex task:docs`
- Set a TTL for agent-created worktrees unless the task is expected to stay open

## Baseline Verification

After creating a new worktree for this repository:

```bash
go test ./...
```

If the baseline is red, report that before making feature changes.

## Safety Rules

- Treat `ww check` as human-readable output, not a stable machine interface.
- Do not run `ww-helper gc` without at least one explicit selector such as `--ttl-expired`, `--idle 7d`, or `--merged`.
- Do not remove dirty worktrees without explicit human approval.
- Do not remove the active worktree.
- If `ww-helper` does not cover the workflow you need, document the gap before falling back to raw Git commands.

## Optional Repo Skill

If your agent platform supports installable skills, this repository includes a portable template at `skills/using-ww-worktrees/SKILL.md`. The skill is optional. The repository contract is `ww-helper` plus the rules in this file. For installation guidance, see `docs/agent-skills.md`.
