# ww

One command to switch, create, and clean up worktrees.

`ww` is a shell-first Git worktree workflow for the current repository. Use the fast picker to jump into an existing branch, open a fresh worktree from where you already are, and clean up temporary branches without breaking shell flow.

## Demo

[![ww demo](docs/assets/ww-demo.svg)](https://unix2dos.github.io/ww/)

The demo follows the core loop in under half a minute:

- switch into an existing worktree with the `fzf` fast path
- create a fresh branch workspace with `ww new feat-demo`
- remove that temporary workspace with safe `ww rm`

## Why ww

- `ww` changes the current shell directory, so switching worktrees feels like changing folders, not launching a side tool.
- `ww` keeps the everyday loop in one command family: `ww`, `ww new`, `ww rm`.
- `ww rm` only deletes the branch when it is already merged into the effective base branch.

## Quick Start

Install the latest release for your shell:

```bash
curl -fsSL https://github.com/unix2dos/ww/releases/latest/download/install-release.sh | bash
source ~/.zshrc
```

Then try the core flow inside any Git repository:

```bash
ww
ww new feat-demo
ww rm feat-demo
```

## Selector Behavior

For the fastest path, install `fzf`. If `fzf` is not available, `ww` automatically falls back to the built-in selector, so the workflow still works without extra setup.

## For AI Agents

Use `ww-helper` for programmatic calls. `ww` stays shell-first for humans and still changes your current shell directory for `switch` and `new`.

Phase 1 machine-readable commands:

```bash
ww-helper list --json
ww-helper new-path --json feat-demo
ww-helper rm --json --non-interactive feat-demo
```

`ww-helper rm --json` now returns a JSON envelope with `ok`, `command`, and `data`/`error`. This is a breaking change from the older flat JSON object.

## Reference

`README.md` is the landing page. Detailed install, usage, release, and command reference now live in:

- [Reference Guide](docs/reference.md)
- [Demo Script Notes](docs/demo-script.md)
