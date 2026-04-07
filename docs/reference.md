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
- `ww rm [<name>]` removes a worktree and deletes its branch only when that branch is already merged into the effective base branch.
- `ww rm --cleanup` opens an interactive cleanup review for old worktrees.
- `ww help` or `ww --help` prints the command summary.
- `ww` uses `fzf` automatically when available and falls back to the built-in arrow-key selector otherwise.

### For AI Agents

Use `ww-helper` for machine-readable workflows. `ww` remains the human shell entrypoint and still treats `switch` / `new` as directory-changing commands.

Current programmatic commands:

```bash
ww-helper list --json
ww-helper new-path --json --label agent:claude-code --ttl 24h -m "Fix login redirect" feat-a
ww-helper gc --ttl-expired --idle 7d --dry-run --json
ww-helper rm --json --non-interactive feat-a
```

The shared integration contract is `AGENTS.md` plus the machine-readable `ww-helper` commands. When `ww-helper` covers a workflow, agents should use it instead of scripting raw `git worktree` commands.

`ww-helper switch-path` remains a path-printing helper; agents should keep using `switch-path` and the JSON subcommands above for machine-readable flows.

#### JSON Envelope

Successful `--json` responses use:

```json
{
  "ok": true,
  "command": "list",
  "data": { ... }
}
```

Error responses use:

```json
{
  "ok": false,
  "command": "rm",
  "error": {
    "code": "WORKTREE_DIRTY",
    "message": "worktree has uncommitted changes; rerun with --force",
    "exit_code": 1
  }
}
```

#### `ww-helper list --json`

Returns an array of worktrees with:

- `path`
- `branch`
- `dirty`
- `active`
- `created_at`
- `last_used_at`
- `label`
- `ttl`

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

`gc requires at least one explicit selector`; a bare `ww-helper gc --json` returns `GC_RULE_REQUIRED` with exit code `2`.

Dry-run responses use the same envelope and return:

- `summary.matched`
- `summary.removed`
- `summary.skipped`
- `items[].matched_rules`
- `items[].action`
- `items[].reason` when skipped

#### `ww-helper rm --json --non-interactive <target>`

Removes the target without prompting, while still enforcing the normal safety rules:

- dirty worktrees still require `--force`
- the active worktree cannot be removed
- if you omit `<target>` and more than one removable worktree exists, the command returns `AMBIGUOUS_MATCH`

#### Breaking Change

`ww-helper rm --json` used to return a flat JSON object. It now returns the same JSON envelope format as the other Phase 1 machine-readable commands.

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

Available helper-driven filters:

- `--filter dirty`
- `--filter label=agent:claude-code`
- `--filter label~agent`
- `--filter stale=7d`

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
ww rm --base release/1.0 feat-a
ww rm --cleanup
```

`ww rm` groups removable worktrees by deletion risk, prints a plain-language summary card after selection, removes the worktree, and only deletes the branch when it is already merged into the effective base branch. Dirty worktrees stop before confirmation unless you explicitly rerun with `--force`.

`ww rm --cleanup` opens a repeated review flow for non-current worktrees so you can clean up multiple stale workspaces without learning helper-only cleanup rules.

When saved workspace context exists, the summary card also includes that context and weak-boundary warnings such as detached state or missing context.

### Typical Flow

```bash
cd /path/to/repo
ww
ww switch feat-a
ww list
ww new feat-b
ww rm feat-a
ww rm --cleanup
```

`ww`, `ww 2`, and `ww switch feat-a` all switch the current shell into the target worktree.

### Exit Behavior

- `0`: success
- `2`: invalid user input such as a bad index, bad name match, or extra args
- `3`: environment problem such as not being in a Git repo
- `130`: interactive selection canceled

For `ww-helper ... --json`, the envelope `error.exit_code` matches the process exit code.

## Smoke Test Matrix

```bash
ww --help
ww help
ww 1
ww switch feat-a
ww list
ww new feat-b
ww rm feat-a
ww rm --cleanup
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
