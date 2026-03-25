# Optional Agent Skill Setup

`ww` does not install platform-specific skills automatically.

The integration model is split into two layers:

- **Shared contract:** this repository's `AGENTS.md` plus the machine-readable `ww-helper` commands
- **Optional local enhancement:** a platform-specific skill that wraps `ww-helper`

That means a coding agent can work with this repository even if no local skill is installed. The optional skill only makes the workflow more discoverable and repeatable in platforms that support installable skills.

## How Agents Discover The Workflow

### Baseline: Repository Instructions

Agents that read repository guidance should follow:

- `AGENTS.md` for the local workflow rules
- `README.md` and `docs/reference.md` for command examples

This is the portable path. It does not depend on any local skill system.

### Optional: Local Skill

If your agent platform supports local skills, install the template from:

`skills/using-ww-worktrees/SKILL.md`

That template tells the agent to prefer `ww-helper --json` over raw `git worktree` commands.

## Codex Example

Codex looks for installed skills under `$CODEX_HOME/skills`.

The fastest path is the helper script shipped in this repository:

```bash
bash scripts/install-codex-skill.sh
```

If you want to copy the template manually, use:

```bash
mkdir -p "${CODEX_HOME:-$HOME/.codex}/skills/using-ww-worktrees"
cp "$(git rev-parse --show-toplevel)/skills/using-ww-worktrees/SKILL.md" \
  "${CODEX_HOME:-$HOME/.codex}/skills/using-ww-worktrees/SKILL.md"
```

Then start a new Codex session in the target repository.

In practice, the agent should now have two sources of truth:

- the installed local skill for reusable workflow guidance
- the repository's `AGENTS.md` for repository-specific rules

## Recommended Usage

Use this model in order:

1. Rely on `AGENTS.md` and `ww-helper` as the default integration path.
2. Install `skills/using-ww-worktrees/SKILL.md` only if your platform benefits from local skills.
3. Keep repository rules authoritative when a local skill and the repository diverge.

## Why There Is No Automatic Skill Install

Automatic skill installation is intentionally out of scope for `ww`:

- skill directories are platform-specific
- skill formats are platform-specific
- many users want `ww` without any agent integration

`ww` therefore ships the reusable template in-repo and keeps the actual product contract at the CLI and repository-documentation layer.
