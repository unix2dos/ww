# ww Reference

Use [the README](../README.md) for the product overview and demo. This page keeps the full install, usage, and release reference.

## Install

For the best interactive workflow, install `fzf`. If `fzf` is not available, `ww` falls back to the built-in arrow-key selector automatically.

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
- `ww list` prints worktrees without changing directory.
- `ww new <name>` creates a new worktree under `./.worktrees/<name>` and switches into it.
- `ww rm [<name>]` removes a worktree and deletes its branch only when that branch is already merged into the effective base branch.
- `ww help` or `ww --help` prints the command summary.
- `ww` uses `fzf` automatically when available and falls back to the built-in arrow-key selector otherwise.

### Interactive Pick

```bash
ww
```

Without `fzf`, this opens the built-in selector like:

```text
* [1] ACTIVE main /path/to/repo
  [2]        feat-a /path/to/repo/.worktrees/feat-a

Use Up/Down (or j/k). Enter to confirm. Esc/Ctrl-C to cancel.
```

Move with arrow keys and press Enter to switch. The active shell worktree is labeled `ACTIVE`, and the selector starts on it by default.

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
```

This prints the current worktree table without changing your shell directory.

Worktrees are shown from oldest to newest by worktree creation time. Smaller indices refer to older worktrees, and the current one is labeled `ACTIVE`.

### New

```bash
ww new feat-a
```

This creates branch `feat-a` from the current `HEAD` in `./.worktrees/feat-a`, then switches into it.

### Remove

```bash
ww rm
ww rm feat-a
ww rm --force feat-a
ww rm --base release/1.0 feat-a
```

`ww rm` lists removable worktrees, shows whether each one is dirty or merged, asks for confirmation, removes the worktree, and only deletes the branch when it is already merged into the effective base branch.

### Typical Flow

```bash
cd /path/to/repo
ww
ww switch feat-a
ww list
ww new feat-b
ww rm feat-a
```

`ww`, `ww 2`, and `ww switch feat-a` all switch the current shell into the target worktree.

### Exit Behavior

- `0`: success
- `2`: invalid user input such as a bad index, bad name match, or extra args
- `3`: environment problem such as not being in a Git repo
- `130`: interactive selection canceled

## Smoke Test Matrix

```bash
ww --help
ww help
ww 1
ww switch feat-a
ww list
ww new feat-b
ww rm feat-a
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

To publish a GitHub Release, create and push a tag matching `v*`:

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

GitHub release publishing is wired through `.github/workflows/release.yml` and only publishes when the workflow runs for `refs/tags/v*`.

Manual `workflow_dispatch` runs still build the `dist/` artifacts, but they do not publish a GitHub Release.
