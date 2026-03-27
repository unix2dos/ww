# w+w

Fast worktree switching for safer parallel work.

`ww` is a shell-first Git worktree workflow for the current repository. It keeps the fast switch/create/remove loop, then adds an interactive cleanup path so parallel work stays manageable.

## Demo

[![ww demo](docs/assets/ww-demo.svg)](https://unix2dos.github.io/ww/)

The demo is now a workflow overview in about a minute, with a short `ww-helper --json` tail:

- switch into an existing worktree with the `fzf` fast path
- inspect the current workspace set with `ww list`
- create a fresh branch workspace with `ww new feat-demo`
- remove the temporary workspace with safe `ww rm`
- review stale workspaces with `ww rm --cleanup`
- end with a quick machine-readable `ww-helper --json` pass

## Why ww

- `ww` changes the current shell directory, so switching worktrees feels like changing folders, not launching a side tool.
- `ww new <name>` creates a fresh branch workspace and moves your shell into it immediately.
- `ww list` shows all worktrees at a glance; `ww list --verbose` adds labels, intent, and metadata.
- `ww rm` explains what will be removed, what will be kept, and what looks risky before you confirm.
- `ww rm --cleanup` lets you review old worktrees and delete the ones you no longer need.

## Quick Start

Install with Homebrew tap:

```bash
brew tap unix2dos/ww https://github.com/unix2dos/ww
brew install ww
printf 'eval "$("%s/bin/ww-helper" init zsh)"\n' "$(brew --prefix ww)" >> ~/.zshrc
source ~/.zshrc
```

`ww-helper init zsh` prints the activation snippet if you want to inspect it before adding it to your shell rc file.

Or install the latest release for your shell:

```bash
curl -fsSL https://github.com/unix2dos/ww/releases/latest/download/install-release.sh | bash
source ~/.zshrc
```

Then try the boundary-safe loop inside any Git repository:

```bash
ww
ww new feat-demo
ww list
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
ww-helper new-path --json --label agent:codex --ttl 24h -m "Fix login redirect" feat-demo
ww-helper gc --ttl-expired --dry-run --json
ww-helper rm --json --non-interactive feat-demo
```

`ww` does not install platform-specific skills. The shared contract for coding agents is the machine-readable `ww-helper` interface plus repository instructions such as `AGENTS.md`. If your agent platform supports installable skills, this repository also ships an optional template at `skills/using-ww-worktrees/SKILL.md`.

Human-facing safety flow:

```bash
ww new feat-a
ww rm --cleanup
```

`ww-helper rm --json` uses the same JSON envelope shape as the other machine-readable commands. For humans, bulk cleanup lives under `ww rm --cleanup`. For automation, `ww-helper gc` still requires at least one explicit selector such as `--ttl-expired`, `--idle 7d`, or `--merged`.

If you want a local skill for Codex or another skill-capable agent, see [Optional Agent Skill Setup](docs/agent-skills.md).

## Reference

`README.md` stays in landing-page mode. Detailed install, usage, release, and command reference live in:

- [Reference Guide](docs/reference.md)
- [Demo Script Notes](docs/demo-script.md)
