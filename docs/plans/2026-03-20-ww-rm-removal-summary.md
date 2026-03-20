# WW Remove Summary Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace `ww rm`'s raw safety labels with a grouped picker and a post-selection summary card that explains deletion impact in plain language.

**Architecture:** Keep all removal semantics in `internal/git` unchanged. Add presentation-only helpers in `internal/app` to classify candidates, render grouped lists, render summary cards, and short-circuit unsafe dirty removals before confirmation.

**Tech Stack:** Go, standard library, Git CLI

---

### Task 1: Lock the new UX with failing tests

**Files:**
- Modify: `internal/app/run_test.go`
- Modify: `test/e2e/e2e_test.go`

**Step 1: Write the failing test**

Add tests that prove:
- interactive `ww rm` renders grouped headings and plain-language reasons
- merged worktrees show a safe summary card before confirmation
- dirty worktrees without `--force` show a stop card and do not prompt
- dirty worktrees with `--force` show a review card and still prompt

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app ./test/e2e`

Expected: FAIL because `ww rm` still prints raw Git status labels and still prompts before the dirty refusal.

### Task 2: Implement the new removal presentation flow

**Files:**
- Modify: `internal/app/run.go`

**Step 1: Write minimal implementation**

Add helpers for:
- risk classification
- grouped candidate rendering
- plain-language summary-card rendering
- early stop behavior for dirty worktrees without `--force`

**Step 2: Run test to verify it passes**

Run: `go test ./internal/app ./test/e2e`

Expected: PASS

### Task 3: Update docs and demo expectations

**Files:**
- Modify: `docs/reference.md`
- Modify: `scripts/demo-record.exp`
- Modify: `docs/assets/ww-demo.cast`
- Modify: `docs/assets/ww-demo.svg`

**Step 1: Update static expectations**

Align docs and demo capture expectations with the new summary-card flow.

**Step 2: Run verification**

Run: `go test ./test/docs`

Expected: PASS

### Task 4: Full verification

**Files:**
- Verify only

**Step 1: Run the full test suite**

Run: `go test ./...`

Expected: PASS
