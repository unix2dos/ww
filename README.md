# wt

`wt` is a small Git worktree switcher for the current repository.

## Install

### Install From Source

```bash
cd /Users/liuwei/workspace/wt
bash install.sh
source ~/.zshrc
```

If you use Bash, reload with `source ~/.bashrc` instead.

The installer builds `wt` into `~/.local/bin/wt` and appends a managed shell block that sources `shell/cwt.sh`.

Source installs require a working Go toolchain.

### Install From A Release Bundle

```bash
tar -xzf wt-v0.1.0-darwin-arm64.tar.gz
cd wt-v0.1.0-darwin-arm64
bash install.sh
source ~/.zshrc
```

Release bundle installs copy the prebuilt `bin/wt` binary and do not require Go.

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

`wt` 只处理“当前仓库”的 worktree。你需要在某个 Git 仓库或它的 worktree 目录里运行它。

### Interactive Pick

```bash
wt
```

This prints a numbered menu like:

```text
[1] * main /path/to/repo
[2]   feat-a /path/to/repo/.worktrees/feat-a
Select a worktree [number]:
```

Enter a number and `wt` prints only the selected path to `stdout`.

### Direct Index

```bash
wt 2
```

Useful for scripting:

```bash
target="$(wt 2)"
cd "$target"
```

### Fzf Mode

```bash
wt --fzf
```

This opens `fzf`, searches by the non-index columns, and prints the selected path.

### Switch Current Shell

```bash
cwt
cwt 2
cwt --fzf
```

`cwt` is the shell wrapper. It calls `wt`, reads the returned path, and runs `cd` in your current shell session.

### Typical Flow

```bash
cd /path/to/repo
git worktree list
wt
cwt --fzf
```

### Exit Behavior

- `0`: success
- `2`: invalid user input such as a bad index or extra args
- `3`: environment problem such as not being in a Git repo or missing `fzf`
- `130`: `fzf` selection canceled

## Smoke Test Matrix

```bash
wt --help
wt 1
printf '2\n' | wt
wt --fzf
cwt 1
```

Installer checks:

```bash
bash install.sh
bash install.sh
```

## Release

Build release archives locally:

```bash
bash scripts/release.sh v0.1.0
```

Artifacts are written to `dist/`:

- `wt-v0.1.0-darwin-arm64.tar.gz`
- `wt-v0.1.0-darwin-amd64.tar.gz`
- `wt-v0.1.0-linux-arm64.tar.gz`
- `wt-v0.1.0-linux-amd64.tar.gz`
- `checksums.txt`

GitHub release publishing is wired through `.github/workflows/release.yml` and runs on tags matching `v*`.
