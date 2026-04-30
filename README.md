# w+w

**The worktree primitive your AI agents and you share.**

`ww` is a Git worktree workflow with two interfaces over one set of worktrees: `ww` for humans (fzf-driven, changes your shell directory) and [`ww-helper --json`](docs/protocol.md) for AI agents and orchestrators — backed by a versioned wire protocol you can depend on.

## Demo

[![ww demo](docs/assets/ww-demo.svg)](https://unix2dos.github.io/ww/)

A one-minute workflow overview ending with a short `ww-helper --json` tail:

- switch into an existing worktree with the `fzf` fast path
- inspect the current workspace set with `ww list`
- create a fresh branch workspace with `ww new feat-demo`
- remove a workspace with safe `ww rm`
- end with a quick machine-readable `ww-helper --json` pass

## Why ww

**One mental model for you and your agent.** Both `ww` and `ww-helper` operate on the same worktrees, with the same metadata (labels, TTL, last-used). When a Claude / Codex / Cursor agent creates a worktree, your `ww list` sees it immediately. When you create one, the agent sees it too.

**Shell-first for humans, contract-first for agents.** `ww` changes your current shell directory — switching worktrees feels like `cd`-ing, not launching a side tool. `ww-helper --json` emits a versioned, [stable JSON envelope](docs/protocol.md) so an MCP server, orchestrator, or shell script can depend on the wire format without guessing.

**Safe by default.** `ww rm` shows what will be removed, what will be kept, and what looks risky before you confirm. `ww new` copies your git-ignored config files (`.env`, local configs) into the new worktree so it's runnable on first `cd`.

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

Then try the loop inside any Git repository:

```bash
ww                # interactive switch
ww new feat-demo  # create + cd into a new worktree
ww list           # see all worktrees with status
ww rm feat-demo   # remove with safety preview
```

For the fastest interactive switch, install `fzf`. If `fzf` is not on PATH, `ww` falls back to a built-in selector — the workflow still works without extra setup.

## For AI agents and orchestrators

Two ways to call `ww-helper` from an agent — pick whichever fits the agent's plumbing.

### Over MCP (recommended for Claude Code, Cursor, Zed, …)

Add one block to your MCP config and every worktree command becomes a typed tool. No subprocess marshalling, no JSON parsing in the agent.

```json
{
  "mcpServers": {
    "ww": {
      "command": "ww-helper",
      "args": ["mcp", "serve"]
    }
  }
}
```

The server exposes six tools backed by the same v1.0 wire protocol the CLI uses: `ww_list`, `ww_new`, `ww_remove`, `ww_gc`, `ww_switch_path`, `ww_version`. Schemas are generated from the same Go structs the CLI marshals, so the data shape is identical across both transports.

### As a subprocess (any agent / shell script)

Every `--json` command emits a single-line envelope conforming to the [versioned wire protocol](docs/protocol.md):

```bash
ww-helper version --json
ww-helper list --json
ww-helper new-path --json --label agent:codex --ttl 24h -m "Fix login redirect" feat-demo
ww-helper gc --ttl-expired --dry-run --json
ww-helper rm --json feat-demo
```

Envelope shape:

```json
{"protocol":"1.0","ok":true,"command":"list","data":[...],"warnings":[]}
{"protocol":"1.0","ok":false,"command":"list","error":{"code":"git.repo_missing","message":"...","context":{}}}
```

The `protocol` field, the field names inside `data`, and the `domain.subcode` error codes (`worktree.dirty`, `git.repo_missing`, `selector.fzf_not_installed`, …) are stable within v1.x — additive changes only. See [`docs/protocol.md`](docs/protocol.md) for the complete contract, including per-command schemas, exit codes, and what is explicitly **not** covered (`switch-path` is raw stdout for shell-eval; `list --filter` grammar is not yet frozen).

Repository-level instructions for coding agents live in [`AGENTS.md`](AGENTS.md).

## Reference

`README.md` stays in landing-page mode. Detailed install, usage, release, and command reference live in:

- [Wire Protocol](docs/protocol.md) — for anyone scripting `ww-helper`
- [Reference Guide](docs/reference.md)
- [Demo Script Notes](docs/demo-script.md)
