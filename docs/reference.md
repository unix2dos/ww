# ww Reference

Use [the README](../README.md) for the product overview and demo. This page keeps the full install, usage, command, and release reference.

## Install

For the best interactive workflow, install `fzf`. If `fzf` is not available, `ww` falls back to the built-in arrow-key selector automatically.

### Homebrew Tap

Homebrew installs the helper and shell library, but leaves shell activation to you.

```bash
brew tap unix2dos/ww https://github.com/unix2dos/ww
brew install ww
```

For Zsh:

```bash
printf 'eval "$("%s/bin/ww-helper" init zsh)"\n' "$(brew --prefix ww)" >> ~/.zshrc
source ~/.zshrc
```

For Bash:

```bash
printf 'eval "$("%s/bin/ww-helper" init bash)"\n' "$(brew --prefix ww)" >> ~/.bashrc
source ~/.bashrc
```

`ww-helper init zsh` and `ww-helper init bash` print the activation snippet if you want to inspect it before adding it to your shell rc file.

### One-Line Install

Install the latest release for your current platform:

```bash
curl -fsSL https://github.com/unix2dos/ww/releases/latest/download/install-release.sh | bash
source ~/.zshrc
```

For Bash:

```bash
curl -fsSL https://github.com/unix2dos/ww/releases/latest/download/install-release.sh | bash -s -- --shell bash --rc-file ~/.bashrc
source ~/.bashrc
```

Install a specific version:

```bash
curl -fsSL https://github.com/unix2dos/ww/releases/latest/download/install-release.sh | WT_VERSION=vX.Y.Z bash
```

This path does not require Go. It downloads the installer script from the latest GitHub Release, then fetches the matching release archive for your platform and runs the bundled installer.

### Install From Source

```bash
git clone https://github.com/unix2dos/ww.git
cd ww
bash install.sh
source ~/.zshrc
```

If you use Bash, reload with `source ~/.bashrc` instead.

The installer puts `ww-helper` and `ww.sh` into your target bin directory, then appends a managed shell block that exposes `ww`.

Source installs require a working Go toolchain.

### Install From A Release Bundle

```bash
tar -xzf ww-vX.Y.Z-darwin-arm64.tar.gz
cd ww-vX.Y.Z-darwin-arm64
bash install.sh
source ~/.zshrc
```

Release bundle installs copy the prebuilt `bin/ww-helper` binary and `ww.sh`, and do not require Go.

### Installer Options

```bash
bash install.sh --shell zsh
bash install.sh --shell bash --rc-file ~/.bashrc
bash install.sh --bin-dir ~/.local/bin
```

### Uninstall

```bash
bash uninstall.sh
source ~/.zshrc
```

If you installed into Bash, reload `~/.bashrc` instead.

## Usage

`ww` only works for the current repository. Run it inside a Git repository or one of that repository's worktrees.

`ww` is a shell function that switches worktrees and changes your current shell directory.

- `ww` or `ww switch` selects a worktree and switches into it.
- `ww list` prints worktrees without changing directory. `ww list --verbose` adds labels, intent, and metadata.
- `ww new <name>` creates a new worktree under `./.worktrees/<name>` and switches into it.
- `ww rm [<name>]` removes a worktree and deletes its branch only when that branch is already merged into the effective base branch. Without a target, `ww rm` opens an interactive selector for review-and-remove.
- `ww version` (or `ww --version`) prints the binary and protocol version. Local dev installs include the embedded Git commit when available.
- `ww help` or `ww --help` prints the command summary.
- `ww` uses `fzf` automatically when available and falls back to the built-in arrow-key selector otherwise.

### For AI Agents

Two integration paths — pick whichever fits the agent. Both are backed by the same v1.1 wire protocol; see [`protocol.md`](protocol.md) for the formal contract.

**Over MCP** (Claude Code, Cursor, Zed, Continue, Cline, Codex, …):

```json
{"mcpServers": {"ww": {"command": "ww-helper", "args": ["mcp", "serve"]}}}
```

Six tools become available: `ww_list`, `ww_new`, `ww_remove`, `ww_gc`, `ww_switch_path`, `ww_version`.

**As a subprocess**:

```bash
ww-helper version --json
ww-helper list --json
ww-helper new-path --json --label agent:claude-code --ttl 24h -m "Fix login redirect" feat-a
ww-helper gc --ttl-expired --idle 7d --dry-run --json
ww-helper rm --json feat-a
```

The shared integration contract is `AGENTS.md` plus the machine-readable `ww-helper` commands. When `ww-helper` covers a workflow, agents should use it instead of scripting raw `git worktree` commands.

`ww-helper switch-path` is a path-printing helper for shell-eval (`cd "$(ww-helper switch-path X)"`) and is intentionally **out of the JSON envelope contract**; over MCP, the equivalent `ww_switch_path` tool wraps the path normally.

#### JSON Envelope

Successful `--json` responses:

```json
{
  "protocol": "1.1",
  "ok": true,
  "command": "list",
  "data": { ... },
  "warnings": []
}
```

Error responses:

```json
{
  "protocol": "1.1",
  "ok": false,
  "command": "rm",
  "error": {
    "code": "worktree.dirty",
    "message": "worktree has uncommitted changes; rerun with --force",
    "context": {}
  }
}
```

The envelope no longer carries `exit_code` — the process exit code is the single source of truth. Error codes follow `domain.subcode` (`worktree.dirty`, `git.repo_missing`, `selector.fzf_not_installed`, `input.missing_selector`, …); see `protocol.md` §5 for the full table.

#### `ww-helper list --json`

Returns an array of worktrees. Each entry has:

- `path` — absolute filesystem path
- `branch` — branch label
- `dirty` — boolean; any uncommitted changes
- `active` — boolean; this is the caller's current worktree
- `created_at` — unix milliseconds; `0` if unknown
- `last_used_at` — unix milliseconds; `0` if never
- `label` — free-form metadata string; `""` if none
- `ttl` — duration string (`"24h"`, `"7d"`); `""` if none
- `merged` — branch is merged into the display status base ref
- `ahead` / `behind` — commits ahead/behind the display status base ref
- `status_base_ref` — ref used for `merged`, `ahead`, and `behind`; usually `origin/HEAD`, then `origin/<default>`, then the local default branch when remote refs are unavailable
- `staged` / `unstaged` / `untracked` — change counts

Status display never runs `git fetch`; remote refs are local cached tracking refs. Cleanup and removal safety still use their own conservative base checks.

#### `ww-helper new-path --json --label agent:claude-code --ttl 24h -m "Fix login redirect" feat-a`

Returns:

- `worktree_path`
- `branch`

`label` is stored as a single free-text string. `ttl` is fixed from creation time; this release does not include a metadata editing command. `-m` sets a one-line intent describing what this worktree is for; it appears in `ww list --verbose` and `ww rm` safety output.

When `label` is present, `ww-helper` also stores extra workspace context for later human summaries. That context is kept in Git's per-worktree admin area, not in tracked files.

#### `ww-helper gc --ttl-expired --idle 7d --dry-run --json`

`gc` requires at least one explicit selector. Supported selectors are:

- `--ttl-expired`
- `--idle <duration>`
- `--merged`

A bare `ww-helper gc --json` (no selector) returns `input.missing_selector` with exit code `2`.

Dry-run responses use the same envelope and return:

- `summary.matched`
- `summary.removed`
- `summary.skipped`
- `items[].matched_rules`
- `items[].action`
- `items[].reason` when skipped

#### `ww-helper rm --json <target>`

The JSON path never prompts. Safety rules:

- dirty worktrees require `--force`
- the active worktree cannot be removed (returns `worktree.remove_current`)
- if you omit `<target>` and more than one removable worktree exists, the command returns `worktree.ambiguous_match`

`new-path --json` automatically syncs git-ignored files (`.env` and similar) from the main worktree by default; results are surfaced through the envelope's `warnings` array (`sync.copied`, `sync.skipped`, …). Pass `--no-sync` to opt out, or `--sync-dry-run` to preview without writing files.

### Interactive Pick

```bash
ww
```

Without `fzf`, this opens the built-in selector like:

```text
* [1] [CURRENT]         main   /path/to/repo
  [2]                   feat-a /path/to/repo/.worktrees/feat-a

Use Up/Down (or j/k). Enter to confirm. Esc/Ctrl-C to cancel.
```

Move with arrow keys and press Enter to switch. The selector starts on the active shell worktree by default.

The status column can show:

- `[CURRENT]` for the current clean worktree
- `[CURRENT] [DIRTY]` for the current dirty worktree
- `[DIRTY]` for a non-current dirty worktree

`ww` ignores its own `.worktrees/` management directory when computing this status so the main worktree is not marked dirty just because linked worktrees exist.

### Direct Index Or Name

```bash
ww 2
ww switch feat-a
ww switch fea
```

Exact name matches win. If no exact match exists, `ww` falls back to a unique prefix match.

### List

```bash
ww list
ww list --verbose
```

This prints the current worktree table without changing your shell directory.
The human-readable `ww list` output uses a full Unicode box table. Interactive `ww` selection stays header-free so the picker remains compact.

Example:

```text
┌───────┬───────────────────┬────────┬──────────────────────────────────────────────────┐
│ INDEX │ STATUS            │ BRANCH │ PATH                                             │
├───────┼───────────────────┼────────┼──────────────────────────────────────────────────┤
│ 1     │ [CURRENT]         │ main   │ /path/to/repo                                    │
├───────┼───────────────────┼────────┼──────────────────────────────────────────────────┤
│ 2     │ [DIRTY]           │ feat-a │ /path/to/repo/.worktrees/very/long/path/that/    │
│       │                   │        │ wraps/inside/the/path/cell                       │
└───────┴───────────────────┴────────┴──────────────────────────────────────────────────┘
```

Worktrees are shown from oldest to newest by worktree creation time. Smaller indices refer to older worktrees, and the status column uses the same `[CURRENT]` / `[CURRENT] [DIRTY]` / `[DIRTY]` tags as the interactive selector.
Long `PATH` values are wrapped inside the `PATH` cell instead of being truncated.

Detached worktrees use plain human labels in `ww list`: clean worktrees with no commits of their own show branch `scratch` with detail `idle`; dirty scratch worktrees show `local changes`; detached worktrees with commits not reachable from the base branch show branch `unbranched`, the number of commits, and their last commit subject. Idle scratch worktrees do not show a last commit because it is usually just the base commit, not task context.

`--verbose` appends extra metadata such as stored workspace context and timestamps to the human-readable output.

### New

```bash
ww new feat-a
```

This creates branch `feat-a` from the current `HEAD` in `./.worktrees/feat-a`, copies git-ignored config files from the main worktree into the new one, then switches into it.

For metadata-aware creation, use `ww-helper new-path --json --label ... --ttl ... -m "intent"`. The `-m` flag sets a one-line intent that appears in `ww list --verbose` and `ww rm` safety output.

#### Ignored-File Sync

When a new worktree is created, `ww new` automatically copies git-ignored files from the main worktree — typically `.env`, local config files, and development certificates — so the new workspace is immediately usable.

**Flags:**

```bash
ww new feat-a                  # default: sync enabled
ww new feat-a --no-sync        # skip sync for this run
ww new feat-a --sync-dry-run   # preview what would be copied without writing files
```

**What gets skipped:**

Large dependency and build directories are excluded by default:

- JS/TS: `node_modules/`, `.next/`, `.nuxt/`, `dist/`, `build/`, `.vite/`, `.turbo/`, `coverage/`
- Python: `__pycache__/`, `.venv/`, `venv/`, `env/`, `.pytest_cache/`
- Go/Rust/Java: `vendor/`, `target/`, `.gradle/`
- General: `tmp/`, `temp/`, `logs/`, `.cache/`, `.DS_Store`

Any file at or above 1 MiB is also skipped as a safety net.

**Configuration (`~/.config/ww/config.json`):**

```json
{
  "version": 1,
  "sync": {
    "enabled": true,
    "max_file_size": 1048576,
    "blacklist_extra": ["my-secrets/", "local-certs/"],
    "blacklist_override": null
  }
}
```

- `enabled`: set to `false` to disable sync globally.
- `max_file_size`: per-file size cap in bytes (default 1 MiB).
- `blacklist_extra`: additional path segments appended to the built-in blacklist.
- `blacklist_override`: non-null value replaces the built-in blacklist entirely; an empty array `[]` disables the blacklist completely.

The config file is optional. A missing file uses all built-in defaults. `XDG_CONFIG_HOME` is honoured; the default path is `~/.config/ww/config.json`.

### Remove

```bash
ww rm
ww rm feat-a
ww rm --force feat-a
ww rm --cleanup
```

`ww rm` (no target) opens an interactive selector for review-and-remove. The selector marks clean merged worktrees as `safe` and marks dirty, unmerged, or branchless worktrees as `review` with a short reason. With a target, it removes that worktree directly after confirmation. The branch is deleted only when it is already merged into the effective base branch. Dirty worktrees stop before confirmation unless you explicitly rerun with `--force`.

`ww rm --cleanup` removes all clearly safe worktrees after one confirmation. Safe means clean files, already merged, and not the base branch. The prompt shows numbered names plus each last commit subject, and keeps base-branch, detached, dirty, or unmerged worktrees out of the deletion list.

### Typical Flow

```bash
cd /path/to/repo
ww
ww switch feat-a
ww list
ww new feat-b
ww rm feat-a
ww rm --cleanup  # delete all clearly safe worktrees after one confirmation
ww rm           # interactive picker for review-and-remove
```

`ww`, `ww 2`, and `ww switch feat-a` all switch the current shell into the target worktree.

### Exit Behavior

- `0`: success
- `2`: invalid user input such as a bad index, bad name match, or extra args
- `3`: environment problem such as not being in a Git repo
- `130`: interactive selection canceled

`ww-helper ... --json` envelopes do not carry `exit_code`; rely on the process exit code.

## Smoke Test Matrix

```bash
ww --help
ww help
ww --version
ww 1
ww switch feat-a
ww list
ww new feat-b
ww rm feat-a
ww rm
```

Installer checks:

```bash
bash install.sh
bash install.sh
```

## Release

Build release archives locally:

```bash
bash scripts/release.sh vX.Y.Z
```

Artifacts are written to `dist/`:

- `ww-vX.Y.Z-darwin-arm64.tar.gz`
- `ww-vX.Y.Z-darwin-amd64.tar.gz`
- `ww-vX.Y.Z-linux-arm64.tar.gz`
- `ww-vX.Y.Z-linux-amd64.tar.gz`
- `checksums.txt`
- `install-release.sh`
- `ww.rb`

Refresh the committed Homebrew formula after a release is published:

```bash
bash scripts/generate-homebrew-formula.sh vX.Y.Z Formula/ww.rb
git add Formula/ww.rb
git commit -m "chore: update Homebrew formula for vX.Y.Z"
git push origin main
```

To publish a GitHub Release, create and push a tag matching `v*`:

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

GitHub release publishing is wired through `.github/workflows/release.yml` and only publishes when the workflow runs for `refs/tags/v*`.

Manual `workflow_dispatch` runs still build the `dist/` artifacts, including `ww.rb`, but they do not publish a GitHub Release.
