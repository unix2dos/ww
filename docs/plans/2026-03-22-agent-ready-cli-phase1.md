# Agent-Ready CLI Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a stable machine-readable Phase 1 interface for AI agents by formalizing `ww-helper` as the programmatic entrypoint and adding JSON support to `list`, `new-path`, and `rm`.

**Architecture:** Keep the shell-first human UX unchanged: `ww` remains a shell function that consumes path output and performs `cd`. Put all machine-readable behavior inside `internal/app`, emitted by `ww-helper`, with a shared JSON envelope and shared error mapping so `list`, `new-path`, and `rm` behave consistently.

**Tech Stack:** Go, standard library, Git CLI, existing `internal/app` test harness, install/docs tests.

---

## File Map

- Modify: `internal/app/run.go`
  - Add `--json` parsing for `list` and `new-path`
  - Add shared JSON envelope helpers
  - Add shared app-error-to-exit-code / JSON-error mapping
  - Add `--non-interactive` handling to `rm`
- Modify: `internal/app/run_test.go`
  - Add TDD coverage for success/error envelopes and non-interactive removal
- Modify: `shell/ww.sh`
  - Preserve human shell behavior while allowing pass-through machine commands where needed
- Modify: `test/install/install_test.go`
  - Lock in shell-wrapper behavior so `ww` never tries to `cd` into JSON
- Modify: `README.md`
  - Add a short "For AI agents" section
- Modify: `docs/reference.md`
  - Document `ww-helper` JSON contract, `--non-interactive`, exit codes, and breaking change note

## Scope Guardrails

- Do not introduce `ww switch --json` in this phase.
- Do not change `state.json` format in this phase.
- Do not add `gc`, `label`, or `ttl` in this phase.
- Do not add a separate agent binary beyond the existing `ww-helper`.

### Task 1: Lock the JSON contract in tests

**Files:**
- Modify: `internal/app/run_test.go`

- [ ] **Step 1: Write failing tests for `list --json` success**

Add a test that calls `Run(..., []string{"list", "--json"}, ...)` and asserts:
- exit code `0`
- stdout contains `{"ok":true,"command":"list","data":...}`
- each item includes `path`, `branch`, `dirty`, `active`, `created_at`

- [ ] **Step 2: Write failing tests for `new-path --json` success**

Add a test that calls `Run(..., []string{"new-path", "--json", "alpha"}, ...)` and asserts:
- exit code `0`
- stdout contains `{"ok":true,"command":"new-path","data":...}`
- `data.worktree_path` and `data.branch` match the created worktree

- [ ] **Step 3: Write failing tests for JSON error envelopes**

Add tests covering:
- `list --json` outside a Git repo -> `NOT_GIT_REPO`, exit `3`
- `new-path --json` with missing name -> invalid-input code, exit `2`
- `rm --json --non-interactive` without a target and multiple candidates -> `AMBIGUOUS_MATCH`, exit `2`
- `rm --json beta` on a dirty worktree without `--force` -> `WORKTREE_DIRTY`, exit `1`

- [ ] **Step 4: Write failing tests for `rm --non-interactive`**

Add tests that assert:
- confirmation prompt is skipped
- target is required for ambiguous cases
- successful removal still calls `RemoveWorktree`

- [ ] **Step 5: Run targeted tests to verify failure**

Run: `go test ./internal/app`
Expected: FAIL on missing flag parsing / missing JSON envelope behavior

### Task 2: Implement the minimal app-layer contract

**Files:**
- Modify: `internal/app/run.go`

- [ ] **Step 1: Add shared JSON response helpers**

Implement small helpers for:
- success envelope: `ok`, `command`, `data`
- error envelope: `ok`, `command`, `error.code`, `error.message`, `error.exit_code`

Keep this logic in `internal/app/run.go`; do not create new packages in this phase.

- [ ] **Step 2: Add shared error classification**

Map existing app errors to stable codes and exit codes:
- `git.ErrNotGitRepository` -> `NOT_GIT_REPO`, `3`
- bad index / bad target / extra args / missing args -> invalid-input family, `2`
- ambiguous name match -> `AMBIGUOUS_MATCH`, `2`
- dirty removal stop -> `WORKTREE_DIRTY`, `1`
- interactive cancel -> `CANCELLED`, `130`
- Git command failures -> `GIT_ERROR`, `1`

- [ ] **Step 3: Add `--json` support to `list`**

Parse `list --json`, reuse `orderedWorktrees`, and emit structured items without changing non-JSON human output.

- [ ] **Step 4: Add `--json` support to `new-path`**

Parse `new-path --json <name>`, keep the existing creation path and state touch, and emit JSON without changing the non-JSON path output contract.

- [ ] **Step 5: Upgrade `rm --json` to the shared envelope**

Keep existing removal semantics and result fields, but wrap them in the shared success envelope.

- [ ] **Step 6: Add `rm --non-interactive`**

Skip confirmation when the flag is present, but preserve all safety rules:
- no deleting current worktree
- dirty worktree still requires `--force`
- ambiguous target without a name is rejected

- [ ] **Step 7: Run targeted tests to verify pass**

Run: `go test ./internal/app`
Expected: PASS

### Task 3: Protect the shell-first human UX

**Files:**
- Modify: `shell/ww.sh`
- Modify: `test/install/install_test.go`

- [ ] **Step 1: Write a failing install/shell test**

Add a shell-wrapper test that proves:
- `ww list --json` prints JSON and does not `cd`
- `ww new <name>` still expects a path and changes directory
- `ww switch <name>` still expects a path and changes directory

- [ ] **Step 2: Make the smallest shell change that preserves behavior**

Prefer pass-through logic over a redesign. If no shell change is needed after app-layer design is finalized, keep the diff in tests only and document why.

- [ ] **Step 3: Run targeted shell/install tests**

Run: `go test ./test/install`
Expected: PASS

### Task 4: Update docs for humans and agents

**Files:**
- Modify: `README.md`
- Modify: `docs/reference.md`
- Test: `test/docs/docs_test.go`

- [ ] **Step 1: Add a short "For AI agents" section to `README.md`**

Document:
- use `ww-helper` for programmatic calls
- `ww` remains the interactive shell-first command
- `rm --json` response format changed in this release

- [ ] **Step 2: Expand `docs/reference.md`**

Document:
- `ww-helper list --json`
- `ww-helper new-path --json <name>`
- `ww-helper rm --json --non-interactive <target>`
- exit codes
- JSON envelope format

- [ ] **Step 3: Add or update docs tests if wording is asserted**

Run: `go test ./test/docs`
Expected: PASS

### Task 5: Full verification and release prep

**Files:**
- Verify only

- [ ] **Step 1: Run the full test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 2: Check the diff against the Phase 1 contract**

Verify all of the following are true:
- `ww-helper` is the documented agent entrypoint
- `list`, `new-path`, and `rm` support JSON
- `rm --json` uses the envelope
- `rm --non-interactive` works without confirmation
- shell-first `ww` behavior remains intact
- docs mention the breaking change

- [ ] **Step 3: Prepare release notes**

Include:
- new machine-readable `ww-helper` commands
- `rm --json` breaking-format change
- no `gc` / metadata / MCP yet in this release
