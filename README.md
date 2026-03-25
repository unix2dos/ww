# w+w

Fast worktree switching for safer parallel work.

`ww` is a shell-first Git worktree workflow for the current repository. It keeps the fast switch/create/remove loop, then adds a human-readable safety check and an interactive cleanup path so parallel work stays manageable.

## Demo

[![ww demo](docs/assets/ww-demo.svg)](https://unix2dos.github.io/ww/)

The demo still shows the core loop in under half a minute:

- switch into an existing worktree with the `fzf` fast path
- create a fresh branch workspace with `ww new feat-demo`
- remove that temporary workspace with safe `ww rm`

## Why ww

- `ww` changes the current shell directory, so switching worktrees feels like changing folders, not launching a side tool.
- `ww new <name>` creates a fresh branch workspace and moves your shell into it immediately.
- `ww check` prints the current path, branch, changes, and saved workspace context when available.
- `ww rm` explains what will be removed, what will be kept, and what looks risky before you confirm.
- `ww rm --cleanup` lets you review old worktrees and delete the ones you no longer need.

## Quick Start

Install the latest release for your shell:

```bash
curl -fsSL https://github.com/unix2dos/ww/releases/latest/download/install-release.sh | bash
source ~/.zshrc
```

Then try the boundary-safe loop inside any Git repository:

```bash
ww
ww new feat-demo
ww list
ww check
ww rm feat-demo
ww rm --cleanup
```

## Selector Behavior

For the fastest path, install `fzf`. If `fzf` is not available, `ww` automatically falls back to the built-in selector, so the workflow still works without extra setup.

## For AI Agents

Use `ww-helper` for programmatic calls. `ww` stays shell-first for humans and still changes your current shell directory for `switch` and `new`.

Current machine-readable commands:

```bash
ww-helper list --json
ww-helper new-path --json --label agent:claude-code --ttl 24h feat-demo
ww-helper gc --ttl-expired --dry-run --json
ww-helper rm --json --non-interactive feat-demo
```

`ww` does not install platform-specific skills. The shared contract for coding agents is the machine-readable `ww-helper` interface plus repository instructions such as `AGENTS.md`. If your agent platform supports installable skills, this repository also ships an optional template at `skills/using-ww-worktrees/SKILL.md`.

Human-facing safety flow:

```bash
ww new feat-a
ww check
ww rm --cleanup
```

`ww-helper rm --json` uses the same JSON envelope shape as the other machine-readable commands. For humans, bulk cleanup lives under `ww rm --cleanup`. For automation, `ww-helper gc` still requires at least one explicit selector such as `--ttl-expired`, `--idle 7d`, or `--merged`.

If you want a local skill for Codex or another skill-capable agent, see [Optional Agent Skill Setup](docs/agent-skills.md).

## Reference

`README.md` stays in landing-page mode. Detailed install, usage, release, and command reference live in:

- [Reference Guide](docs/reference.md)
- [Demo Script Notes](docs/demo-script.md)
