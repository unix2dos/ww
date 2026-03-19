# ww

`ww` is a shell-first Git worktree tool for the current repository, with `fzf`-powered interactive switching and a built-in selector fallback.

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
- `ww` uses `fzf` automatically when available and falls back to the built-in arrow-key selector otherwise.

### Interactive Pick

```bash
ww
```

Without `fzf`, this opens the built-in selector like:

```text
> [1] * main /path/to/repo
  [2]   feat-a /path/to/repo/.worktrees/feat-a

Use Up/Down (or j/k). Enter to confirm. Esc/Ctrl-C to cancel.
```

Move with arrow keys and press Enter to switch.

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

This prints the current, MRU-sorted worktree table without changing your shell directory.

### New

```bash
ww new feat-a
```

This creates branch `feat-a` from the current `HEAD` in `./.worktrees/feat-a`, then switches into it.

### Typical Flow

```bash
cd /path/to/repo
ww
ww switch feat-a
ww list
ww new feat-b
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
ww 1
ww switch feat-a
ww list
ww new feat-b
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

GitHub release publishing is wired through `.github/workflows/release.yml` and runs on tags matching `v*`.
