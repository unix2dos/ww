# ww

`ww` is a small Git worktree switcher for the current repository.

## Install

### One-Line Install

Install the latest release for your current platform:

```bash
curl -fsSL https://github.com/unix2dos/wt/releases/latest/download/install-release.sh | bash
source ~/.zshrc
```

For Bash:

```bash
curl -fsSL https://github.com/unix2dos/wt/releases/latest/download/install-release.sh | bash -s -- --shell bash --rc-file ~/.bashrc
source ~/.bashrc
```

Install a specific version:

```bash
curl -fsSL https://github.com/unix2dos/wt/releases/latest/download/install-release.sh | WT_VERSION=vX.Y.Z bash
```

This path does not require Go. It downloads the installer script from the latest GitHub Release, then fetches the matching release archive for your platform and runs the bundled installer.

### Install From Source

```bash
git clone https://github.com/unix2dos/wt.git
cd wt
bash install.sh
source ~/.zshrc
```

If you use Bash, reload with `source ~/.bashrc` instead.

The installer puts the helper binary into your target bin directory, then appends a managed shell block that exposes `ww`.

Source installs require a working Go toolchain.

### Install From A Release Bundle

```bash
tar -xzf ww-vX.Y.Z-darwin-arm64.tar.gz
cd ww-vX.Y.Z-darwin-arm64
bash install.sh
source ~/.zshrc
```

Release bundle installs copy the prebuilt `bin/ww` binary and do not require Go.

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

- `ww` selects a worktree and switches into it.
- `fzf` is opt-in. Use `ww --fzf` when you want fuzzy search.

### Interactive Pick

```bash
ww
```

This prints a numbered menu like:

```text
[1] * main /path/to/repo
[2]   feat-a /path/to/repo/.worktrees/feat-a
Select a worktree [number]:
```

Enter a number and `ww` switches your current shell into that worktree.

### Direct Index

```bash
ww 2
```

Use this for a direct jump to a known worktree.

### Fzf Mode

```bash
ww --fzf
```

This opens `fzf`, searches by the non-index columns, and switches into the selected worktree.

### Switch Current Shell

```bash
ww
ww 2
ww --fzf
```

### Typical Flow

```bash
cd /path/to/repo
git worktree list
ww
```

`ww` and `ww 2` both switch the current shell into the selected worktree.

### Exit Behavior

- `0`: success
- `2`: invalid user input such as a bad index or extra args
- `3`: environment problem such as not being in a Git repo or missing `fzf`
- `130`: `fzf` selection canceled

## Smoke Test Matrix

```bash
ww --help
ww 1
ww --fzf
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
