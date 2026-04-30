# ww-helper Protocol (v1.0-draft)

> **Status:** DRAFT — not yet frozen. Sections marked `[DECIDE]` need a call before v1.0 is published.
>
> **Audience:** AI agents, orchestrators, IDE plugins, and any tool that wants to script `ww` programmatically.
>
> **Promise (once frozen):** the shape of every JSON envelope, the spelling of every field, the meaning of every error code, and every exit code documented here are stable across the v1.x line. New fields may be added; existing fields will not be renamed, retyped, or removed within a major version.

---

## 1. Why this document exists

`ww-helper` is the machine-readable interface to `ww`. Without a written contract, no orchestrator or plugin can safely depend on it — every release is a guessing game. This document freezes the contract so that:

- AI agents and orchestrators (e.g. ccmanager, Rift, custom MCP servers) can call `ww-helper` and rely on the output shape.
- Downstream tools can parse stderr/stdout/exit-codes deterministically.
- `ww-helper`'s human counterpart (`ww`) is free to evolve UX without breaking machines.

The human-facing `ww` command is **not** covered by this contract. Its output may change at any time. Always use `ww-helper` for scripting.

---

## 2. Versioning

- The protocol uses **semantic versioning** independent of the `ww` binary version.
- Current protocol version: **`1.0-draft`** → target stable: **`1.0`**.
- Within a major version: additive changes only (new commands, new optional fields, new error codes documented as additive).
- A major version bump (e.g. `2.0`) is required to remove or change the meaning of an existing field.
- Every JSON envelope carries a top-level `"protocol"` field whose value is the semver string of the contract that envelope conforms to (e.g. `"1.0"`). Clients should branch on the **major** component only; minor bumps are guaranteed additive.
- A `ww-helper version --json` command will return the binary version and the protocol version it implements. **`[TODO]` ship this command before v1.0.**

---

## 3. Envelope

Every JSON output from `ww-helper --json` is a single line of UTF-8 JSON written to **stdout** and terminated by `\n`. No other text is written to stdout.

### 3.1 Success envelope

```json
{
  "protocol": "1.0",
  "ok": true,
  "command": "<command-name>",
  "data": <command-specific payload>,
  "warnings": []
}
```

### 3.2 Error envelope

```json
{
  "protocol": "1.0",
  "ok": false,
  "command": "<command-name>",
  "error": {
    "code": "<domain.subcode>",
    "message": "<human-readable message>",
    "context": {}
  }
}
```

The process exit code carries the same severity information as the error (see §5). Clients should rely on the process exit code at the OS layer and on `error.code` inside the envelope; the two are always consistent. There is **no** `exit_code` field inside the envelope — that would be redundant and a source of skew.

### 3.3 Stable invariants

| Invariant | Guarantee |
|-----------|-----------|
| `protocol` is always present and a valid semver string | ✅ |
| `ok` is always present and boolean | ✅ |
| Exactly one of `data` (when `ok=true`) or `error` (when `ok=false`) is present | ✅ |
| `command` is always the canonical command name (lowercase, hyphenated) | ✅ |
| Output is exactly one JSON line per invocation | ✅ |
| Side-channel logs go to **stderr**, never stdout | ✅ |
| Field order in JSON is **not** stable; clients must parse, not pattern-match | ✅ |

### 3.4 Warnings (success-with-info channel)

`warnings` is an **optional array** on the success envelope. It carries non-fatal signals that a machine consumer needs but a human reader can ignore. Examples: a partial sync result, a deprecated flag was used, a state file was missing and recreated.

```json
{"code": "sync.copied", "context": {"file": ".env"}}
```

- `code` follows the same `domain.subcode` convention as error codes (see §5).
- `message` is human-readable and **not** stability-covered. Code-driven branching only on `code`.
- `context` is an open-ended object whose **keys** are stability-covered per code (i.e., once `sync.skipped` documents a `file` key, that key is preserved within v1.x). New keys may be added; existing keys won't be removed or repurposed.
- An empty array `[]` and a missing field MUST be treated as equivalent. Clients should default to `[]` when absent for forward compatibility.

**Known warning codes (v1.0):**

| Code | Emitted by | Context keys |
|------|-----------|--------------|
| `sync.copied` | `new-path --json` | `file` (string), `dry_run` (bool, only when true) |
| `sync.skipped` | `new-path --json` | `file` (string), `reason` (string: `blacklisted`, `too_large`, `not_regular`, `read_error`), `size` (int, when known) |
| `sync.failed` | `new-path --json` | — (`message` carries the cause) |
| `sync.config_error` | `new-path --json` | — (`message` carries the parse error; sync continues with defaults) |

The error envelope's `error.context` follows the same rules: optional, code-keyed, additive.

---

## 4. Commands

> Pass `--json` to opt into machine-readable mode. Without `--json`, `ww-helper` prints human-friendly text and is **not** covered by this contract.

| Command | Purpose | Stable since |
|---------|---------|--------------|
| `list` | Enumerate worktrees with status | 1.0 |
| `new-path` | Create a worktree, return its path | 1.0 |
| `gc` | Evaluate cleanup rules; optionally remove | 1.0 |
| `rm` | Remove a specific worktree | 1.0 |
| `init` | Print shell activation snippet (text only, no JSON) | 1.0 |
| `switch-path` | Resolve a worktree path. **Raw stdout** — see §4.3 | 1.0 |
| `version` | Report binary + protocol version | 1.0 |

### 4.1 `list --json`

**Purpose:** enumerate every worktree in the current repository with status fields needed for orchestration decisions.

**Flags:** `--filter <expr>` (see §6 — currently unstable), `--verbose` (no schema impact in JSON mode).

**Data shape:** array of worktree objects. Empty array if no worktrees match.

```json
{
  "protocol": "1.0",
  "ok": true,
  "command": "list",
  "data": [
    {
      "path": "/abs/path/to/worktree",
      "branch": "feat-demo",
      "dirty": false,
      "active": true,
      "created_at": 1714329600000,
      "last_used_at": 1714330000000,
      "label": "agent:codex",
      "ttl": "24h",
      "merged": false,
      "ahead": 2,
      "behind": 0,
      "staged": 0,
      "unstaged": 1,
      "untracked": 3
    }
  ],
  "warnings": []
}
```

**Field contract:**

| Field | Type | Meaning | Stability |
|-------|------|---------|-----------|
| `path` | string (abs) | Filesystem path to the worktree | stable |
| `branch` | string | Branch label as reported by git | stable |
| `dirty` | bool | Has uncommitted changes (any of staged/unstaged/untracked > 0) | stable |
| `active` | bool | This worktree is the caller's current directory | stable |
| `created_at` | int (unix milliseconds) | When `ww` first recorded this worktree. `0` = unknown | stable |
| `last_used_at` | int (unix milliseconds) | Last time this worktree was switched into via `ww`. `0` = never | stable |
| `label` | string | Free-form metadata label set at creation; `""` if none | stable |
| `ttl` | string | Duration string (`"24h"`, `"7d"`); `""` if none | stable |
| `merged` | bool | Branch is merged into the base branch | stable |
| `ahead` | int | Commits ahead of base branch | stable |
| `behind` | int | Commits behind base branch | stable |
| `staged` | int | Count of staged changes | stable |
| `unstaged` | int | Count of unstaged changes | stable |
| `untracked` | int | Count of untracked files | stable |

> **Implementation status:** as of `feat/protocol-v1.0`, the binary emits both timestamps in unix milliseconds via the `nanosToMillis` helper. Internal storage stays in nanoseconds; only the JSON output is converted.

### 4.2 `new-path --json`

**Purpose:** create a new worktree from a branch name and return its absolute path so the caller can `cd` into it (or hand it off to an agent).

**Flags:**

| Flag | Meaning |
|------|---------|
| `--label <str>` | Free-form metadata stored alongside the worktree |
| `--ttl <duration>` | Duration string for `gc --ttl-expired` |
| `-m <message>` | Task-note message stored on creation when `--label` is set |
| `--no-sync` | Skip ignored-file sync (only relevant in human mode; see note) |
| `--sync-dry-run` | Same as above |

**Data shape:**

```json
{
  "protocol": "1.0",
  "ok": true,
  "command": "new-path",
  "data": {
    "worktree_path": "/abs/path/to/new-worktree",
    "branch": "feat-demo"
  },
  "warnings": []
}
```

**Field contract:**

| Field | Type | Meaning | Stability |
|-------|------|---------|-----------|
| `worktree_path` | string (abs) | Path of the newly created worktree | stable |
| `branch` | string | The branch the worktree is checked out to (echoes the requested name) | stable |

**Sync behavior:** `new-path --json` copies the same git-ignored files (`.env`, local config) that `ww new` copies, by default. Sync is best-effort: failures and per-file outcomes are reported via the envelope's `warnings` array (codes `sync.copied`, `sync.skipped`, `sync.failed`, `sync.config_error`) and never fail the operation. Pass `--no-sync` to opt out, or `--sync-dry-run` to report what would be copied without writing files. This makes the JSON surface behaviorally consistent with the human surface and with `ww_new` over MCP.

### 4.3 `switch-path` — RAW STDOUT, OUT OF CONTRACT

`switch-path` is **deliberately not covered by the JSON envelope contract.**

It exists to support the shell idiom `cd "$(ww-helper switch-path <name>)"`. Its only output is a single line: the absolute path of the resolved worktree. There is no `--json` flag and there will not be one in v1.0 — wrapping the path in an envelope would break every shell-eval caller.

**Stable contract for `switch-path` (narrow, not-an-envelope):**

- On success (exit 0): exactly one line on stdout, the absolute path, terminated by `\n`. Nothing else on stdout.
- On failure (non-zero exit): nothing on stdout. A human-readable message on stderr. Exit codes follow §5.1.
- Argument forms (all stable):
  - `switch-path` (no args): interactive selector (fzf or built-in TUI). Not for scripts.
  - `switch-path --fzf`: force fzf. Returns exit 3 with `fzf is not installed` if fzf is missing.
  - `switch-path <name-or-substring>`: resolve by name match.
  - `switch-path <index>`: resolve by 1-based list index.

If you need structured output for resolution, use `list --json` and resolve client-side.

### 4.4 `gc --json`

**Purpose:** evaluate cleanup rules against existing worktrees. With `--dry-run`, only report matches; without, perform removals and report results.

**Required selectors (at least one):** `--ttl-expired`, `--idle <duration>`, `--merged`. Calling `gc` without a selector is rejected with `input.missing_selector`.

**Data shape:**

```json
{
  "protocol": "1.0",
  "ok": true,
  "command": "gc",
  "data": {
    "summary": { "matched": 3, "removed": 2, "skipped": 1 },
    "items": [
      {
        "path": "/abs/path",
        "branch": "stale-feat",
        "matched_rules": ["ttl-expired", "merged"],
        "action": "removed",
        "reason": ""
      }
    ]
  },
  "warnings": []
}
```

**Field contract:**

| Field | Type | Meaning | Stability |
|-------|------|---------|-----------|
| `summary.matched` | int | Total candidates evaluated | stable |
| `summary.removed` | int | Successful removals | stable |
| `summary.skipped` | int | Skipped (e.g., dirty without `--force`) | stable |
| `items[].path` | string | Worktree path | stable |
| `items[].branch` | string | Branch label | stable |
| `items[].matched_rules` | string[] | Which selector(s) matched (`"ttl-expired"`, `"idle"`, `"merged"`) | stable |
| `items[].action` | enum | `"removed"`, `"skipped"`, `"dry_run"` | stable |
| `items[].reason` | string | Human-readable reason when skipped; `""` otherwise | code-stable per `action` |

**`action` enum:** `"removed"` \| `"skipped"` \| `"dry_run"`. New enum values require a major version bump.

### 4.5 `version --json`

**Purpose:** report the running binary's build version and the protocol version it speaks. Use this when you want to feature-detect against a specific binary release or confirm protocol compatibility before issuing other commands.

**Flags:** `--json` (no other options).

**Data shape:**

```json
{
  "protocol": "1.0",
  "ok": true,
  "command": "version",
  "data": {
    "binary": "v0.11.0"
  },
  "warnings": []
}
```

**Field contract:**

| Field | Type | Meaning | Stability |
|-------|------|---------|-----------|
| `binary` | string | Build version of the binary. `"dev"` for unreleased builds; otherwise the release tag (e.g. `"v0.11.0"`). Injected at build time via ldflags | stable |

The protocol version is intentionally **not** duplicated inside `data` — it already lives at the envelope's top-level `protocol` field. Pulling it from `data` would invite skew between the two.

**Without `--json`:** prints a single line `ww-helper <binary> (protocol <protocol>)\n` to stdout. This text form is **not** covered by the contract.

### 4.6 `rm --json`

**Purpose:** remove one named worktree (and optionally its merged branch).

**Flags:**

| Flag | Meaning |
|------|---------|
| `--force` | Remove despite uncommitted changes (data loss risk — caller's responsibility) |

**Positional:** `<name|path|index>` — same resolution rules as `switch-path`. When omitted in `--json` mode, `rm` returns `worktree.not_found` rather than prompting (the JSON path never blocks on a TTY).

**Data shape:**

```json
{
  "protocol": "1.0",
  "ok": true,
  "command": "rm",
  "data": {
    "worktree_path": "/abs/path",
    "branch": "feat-demo",
    "base_branch": "main",
    "removed_worktree": true,
    "deleted_branch": true,
    "kept_branch_reason": ""
  },
  "warnings": []
}
```

**Field contract:**

| Field | Type | Meaning | Stability |
|-------|------|---------|-----------|
| `worktree_path` | string (abs) | The removed worktree's path | stable |
| `branch` | string | The branch the worktree was on (`""` if detached) | stable |
| `base_branch` | string | Base branch used for merged-detection (e.g. `"main"`) | stable |
| `removed_worktree` | bool | The worktree directory was removed (always `true` on success) | stable |
| `deleted_branch` | bool | The branch was also deleted (only when merged into base) | stable |
| `kept_branch_reason` | string | If `deleted_branch=false`, the reason. `""` otherwise. Reasons are stable strings (see below) | code-stable |

**`kept_branch_reason` known values (stable enumeration):**

- `""` — branch was deleted (only when `deleted_branch=true`)
- `"not merged"` — branch has commits not in base
- `"detached"` — worktree was detached HEAD, no branch to delete
- `"protected"` — branch matched a protected pattern (e.g., `main`, `master`)

`[TODO]` audit `git.RemoveResult.KeptBranchReason` callsites to confirm these are the complete set before freeze.

---

## 5. Error codes

Every `ok:false` response carries a `code` from this table. New codes may be added in a minor version; existing codes will not change meaning.

Codes use a `domain.subcode` convention so clients can do prefix-matching (`code.startsWith("worktree.")`) when handling families of failures.

| Code | When | Process exit code |
|------|------|-------------------|
| `git.repo_missing` | Current directory is not inside a git repository | 3 |
| `git.command_failed` | Underlying `git` invocation failed | 1 |
| `worktree.not_found` | The requested worktree name/path doesn't match any | 2 |
| `worktree.ambiguous_match` | The name matches multiple worktrees | 2 |
| `worktree.dirty` | Operation refused because the worktree has uncommitted changes | 1 |
| `worktree.remove_current` | Refused to remove the worktree the caller is currently inside | 1 |
| `selector.fzf_not_installed` | `--fzf` requested but `fzf` is not on PATH | 3 |
| `selector.cancelled` | User cancelled an interactive prompt | 130 |
| `input.missing_selector` | `gc` called without any selector flag | 2 |
| `input.invalid_argument` | Argument parsing failed (extra args, bad index, missing values, unknown flags) | 2 |
| `input.invalid_duration` | A duration value (`--ttl`, `--idle`) failed to parse | 2 |
| `input.invalid_filter` | A `list --filter` expression failed to parse | 2 |

> **Implementation status:** as of `feat/protocol-v1.0`, the binary emits the `domain.subcode` codes above and `selector.fzf_not_installed` is split out from generic git failures. The legacy `UPPER_SNAKE` codes are gone and will not return.

### 5.1 Exit code summary

| Exit | Meaning |
|------|---------|
| 0 | Success |
| 1 | Recoverable failure (dirty worktree, generic git error) |
| 2 | Invalid input (not found, ambiguous, missing selector, bad args) |
| 3 | Environment failure (not a git repo, fzf missing) |
| 130 | User cancelled |

The process exit code is authoritative. The error envelope's `error.code` provides finer-grained classification than the exit code can carry.

---

## 6. Filter expression grammar (`list --filter`)

**Status: out-of-contract for v1.0.** The `--filter` flag works but its grammar is not frozen and may change. Clients that need stability should fetch full `list --json` output and filter client-side.

`[TODO]` either freeze the grammar before v1.1 or formalize a simpler subset. Tracked in §11.

---

## 7. What is NOT covered by this contract

The following can change at any time without a major version bump:

- Human-readable output of `ww` (the human binary)
- Human-readable output of `ww-helper` without `--json`
- Stderr text content (only stdout JSON envelopes are covered)
- Internal state file format on disk (`~/.local/state/ww/...`)
- Specific wording of `error.message` and `warnings[].message` (only `code` keys are stable)
- The order of items in `list.data` (callers must sort if they need a specific order)
- The order of fields inside any JSON object
- Performance characteristics
- The grammar of `list --filter` (see §6)

---

## 8. Adding to the protocol

Process for evolving this contract:

1. **New command:** propose in a PR; add a section under §4; bump minor (1.0 → 1.1).
2. **New optional field on existing command:** PR with rationale; add to the field table marked "since 1.x"; bump minor.
3. **New error code or warning code:** PR; add to §5 / §3.4; bump minor.
4. **Breaking change:** requires a `2.0` cycle. No exceptions within `1.x`.

A change is "breaking" if it would cause an existing well-formed client to misinterpret the response. Renaming a field, narrowing a type, removing an enum value, or repurposing an exit code are all breaking. Adding a field, adding an enum value, or relaxing a type (e.g. `string` → `string|null`) is **also** breaking unless explicitly carved out at v1.0 (it isn't).

---

## 9. Reference clients

`[TODO]` once published:

- A minimal Go client showing envelope parsing (`examples/client.go`)
- A `ww-helper mcp serve` mode (planned, see §10)

## 10. `ww-helper mcp serve`

Runs an MCP server over stdio so any MCP-aware agent can call ww-helper natively. Six tools, one per v1.0 command:

| MCP tool | Maps to | Notes |
|----------|---------|-------|
| `ww_list` | `list --json` | Filter expressions accepted but currently dropped (out-of-contract per §6) |
| `ww_new` | `new-path --json` | Defaults to sync; pass `no_sync: true` to skip |
| `ww_remove` | `rm --json` | Refuses current worktree; refuses dirty without `force: true` |
| `ww_gc` | `gc --json` | At least one of `ttl_expired`, `idle`, `merged` is required |
| `ww_switch_path` | `switch-path` (raw) | **Returns the path inside an MCP envelope** — the §4.3 raw-stdout carve-out only applied to shell-eval; MCP wraps it normally |
| `ww_version` | `version --json` | |

Tool schemas are generated from the same Go structs the CLI uses to marshal `--json` output, so the field names and types match this document one-for-one.

**Errors:** when a tool fails, the MCP `CallToolResult.isError` is `true` and `content[0].text` is a single JSON line `{"code": "...", "message": "...", "context": {...}}` mirroring the CLI envelope's `error` object. Agents should branch on `code` exactly as they would for a subprocess invocation.

**Warnings:** for `ww_new`, sync results (`sync.copied`, `sync.skipped`, etc. — see §3.4) are emitted as additional `TextContent` blocks alongside the structured result. Agents that need them parse each text block as JSON `{"warning": {...}}`.

**Stdio hygiene:** while `mcp serve` is running, stdout is reserved for the MCP JSON-RPC transport. Any human-readable diagnostic goes to stderr. Do not pipe the server's stdout anywhere except into an MCP client.

**Server identity:**

```json
{
  "name": "ww",
  "version": "<binaryVersion>"
}
```

`name` is intentionally `"ww"` (the brand) rather than the binary name `"ww-helper"`. Agents locate the binary via the config block's `command` field — the server name is for display only.

**Implementation:** `internal/mcp/` (server, tools, translation). The `mcp serve` subcommand is wired in `internal/app/run.go`; `MCPServe` is injected by `cmd/ww-helper/main.go` to break the import cycle (the MCP package depends on app for the `*Data` functions).

---

## 11. Open decisions before freezing v1.0

Resolved:

- ✅ §2: every envelope carries a `"protocol"` field
- ✅ §3.2: error envelope does not carry `exit_code` (relies on process exit code)
- ✅ §3.4: `warnings` array on success envelope, with `domain.subcode` codes
- ✅ §4.1: timestamps in unix milliseconds (implementation migration pending)
- ✅ §4.3: `switch-path` is deliberately raw-output, out of envelope contract
- ✅ §5: error codes use `domain.subcode` convention; `fzf-not-installed` split out

Still open:

- [x] Add `protocol` field to `writeJSONSuccess` / `writeJSONError` *(done in `feat/protocol-v1.0`)*
- [x] Migrate timestamps in `list --json` from nanoseconds to milliseconds *(done)*
- [x] Rewrite `classifyError` to emit `domain.subcode` codes *(done)*
- [x] Drop `exit_code` from error envelope payload *(done)*
- [x] Add `warnings` array to all success envelopes (default `[]`) *(done)*
- [x] Add `ww-helper version --json` command *(done in `feat/protocol-v1.0`)*
- [ ] Wire `binaryVersion` ldflags injection into `scripts/release.sh` (separate PR; default `"dev"` works in the meantime)
- [ ] Audit `kept_branch_reason` enumeration completeness
- [x] §4.2: `new-path --json` defaults to sync, results surface via `warnings` *(landed; see §3.4 / §4.2)*
- [ ] §6: freeze `list --filter` grammar or keep out-of-contract for 1.0?
- [x] §10: `ww-helper mcp serve` shipped *(landed in v0.12.0; see §10 for the tool list)*
