# WW Worktree Management Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement stable worktree ordering, safe cleanup, and PR-style diff workflows for `ww`.

**Architecture:** Keep CLI routing in `internal/app`, put Git command orchestration in `internal/git`, and keep rendering/selection details inside `internal/ui`. Remove MRU from display semantics while preserving the existing shell-first command flow.

**Tech Stack:** Go, standard library, Git CLI, `golang.org/x/term`

---

### Task 1: Document the product contract

**Files:**
- Create: `docs/plans/2026-03-20-ww-worktree-management-design.md`
- Create: `docs/plans/2026-03-20-ww-worktree-management.md`

**Step 1: Capture the approved design**

Write the design decisions for ordering, `ACTIVE`, `ww rm`, `ww diff`, and default branch detection.

**Step 2: Save the execution plan**

Write this implementation plan with concrete file targets and verification steps.

### Task 2: Stabilize ordering and status rendering

**Files:**
- Modify: `internal/worktree/normalize.go`
- Modify: `internal/worktree/normalize_test.go`
- Modify: `internal/ui/menu.go`
- Modify: `internal/ui/fzf.go`
- Modify: `internal/ui/menu_test.go`
- Modify: `internal/ui/tui_test.go`
- Modify: `internal/ui/fzf_test.go`
- Modify: `internal/app/run_test.go`

**Step 1: Write failing tests**

Add tests that prove:
- order is branch-name ascending regardless of `LastUsedAt`
- current worktree keeps its sorted position
- current worktree renders as `ACTIVE`

**Step 2: Run targeted tests to verify failure**

Run:
`go test ./internal/worktree ./internal/ui ./internal/app`

Expected: failures around MRU ordering and `*` markers.

**Step 3: Write the minimal implementation**

Remove MRU ordering from normalization and update UI renderers to use `ACTIVE`.

**Step 4: Run targeted tests to verify pass**

Run:
`go test ./internal/worktree ./internal/ui ./internal/app`

Expected: PASS

### Task 3: Add Git primitives for default branch, removal, and diff

**Files:**
- Create: `internal/git/default_branch.go`
- Create: `internal/git/default_branch_test.go`
- Create: `internal/git/remove.go`
- Create: `internal/git/remove_test.go`
- Create: `internal/git/diff.go`
- Create: `internal/git/diff_test.go`

**Step 1: Write failing tests**

Add tests for:
- default branch resolution via `origin/HEAD`, then `main`, then `master`
- safe worktree removal decisions and branch deletion behavior
- merge-base diff summary and patch command construction

**Step 2: Run targeted tests to verify failure**

Run:
`go test ./internal/git`

Expected: failures for missing files and commands.

**Step 3: Write the minimal implementation**

Implement helpers for default branch lookup, worktree inspection/removal, and diff summary generation.

**Step 4: Run targeted tests to verify pass**

Run:
`go test ./internal/git`

Expected: PASS

### Task 4: Add `ww rm` and `ww diff` command handling

**Files:**
- Modify: `internal/app/run.go`
- Modify: `internal/app/run_test.go`
- Modify: `shell/ww.sh`

**Step 1: Write failing tests**

Add app tests for:
- `ww rm` candidate listing, confirmation, `--force`, `--base`, and result text / JSON
- `ww diff` default target selection, explicit target selection, summary output, and `--patch`

**Step 2: Run targeted tests to verify failure**

Run:
`go test ./internal/app`

Expected: failures for unknown commands / missing behavior.

**Step 3: Write the minimal implementation**

Wire new subcommands into the existing dependency interfaces and shell wrapper.

**Step 4: Run targeted tests to verify pass**

Run:
`go test ./internal/app`

Expected: PASS

### Task 5: Update user-facing docs and end-to-end coverage

**Files:**
- Modify: `README.md`
- Modify: `test/e2e/e2e_test.go`
- Modify: `test/e2e/testrepo.go`
- Modify: `docs/assets/ww-demo.cast` (only if behavior snapshots require refresh)
- Modify: `docs/assets/ww-demo.svg` (only if behavior snapshots require refresh)

**Step 1: Write failing tests**

Extend e2e coverage for:
- stable ordering after switch/new
- `ww rm` removal rules
- `ww diff` summary and patch behavior

**Step 2: Run targeted tests to verify failure**

Run:
`go test ./test/e2e ./test/docs`

Expected: failures until CLI and docs are updated.

**Step 3: Write the minimal implementation**

Update README/help text and any demo artifacts needed to match the new UX.

**Step 4: Run targeted tests to verify pass**

Run:
`go test ./test/e2e ./test/docs`

Expected: PASS

### Task 6: Full verification

**Files:**
- Verify only

**Step 1: Run the full test suite**

Run:
`go test ./...`

Expected: PASS

**Step 2: Review the diff against the requirements**

Check that ordering, `ACTIVE`, `ww rm`, `ww diff`, help text, and README all match the approved Definition of Done.
