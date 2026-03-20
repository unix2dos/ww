package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"ww/internal/git"
	"ww/internal/ui"
	"ww/internal/worktree"
)

func TestRunHelperHelpPrintsUsageAndExitsZero(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"--help"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("ww-helper")) {
		t.Fatalf("expected help to mention ww-helper, got %q", got)
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("switch-path")) {
		t.Fatalf("expected help to mention switch-path, got %q", got)
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("new-path")) {
		t.Fatalf("expected help to mention new-path, got %q", got)
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("rm")) {
		t.Fatalf("expected help to mention rm, got %q", got)
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("|help|--help")) {
		t.Fatalf("expected help to mention help subcommand, got %q", got)
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("create")) {
		t.Fatalf("expected help to describe creation behavior, got %q", got)
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("fzf when available")) {
		t.Fatalf("expected help to mention auto fzf routing, got %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunHelpSubcommandPrintsUsageAndExitsZero(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"help"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("Usage: ww-helper")) {
		t.Fatalf("expected help usage on stdout, got %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunSwitchPathPrintsSelectedPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		touched: &touchRecord{},
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true, CreatedAt: 10},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", CreatedAt: 20},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta", CreatedAt: 30},
		},
		state: map[string]map[string]int64{
			"/repo/.git": {
				"/repo/.worktrees/alpha": 10,
				"/repo/.worktrees/beta":  20,
			},
		},
	}

	code := Run(context.Background(), []string{"switch-path", "2"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "/repo/.worktrees/alpha\n" {
		t.Fatalf("expected selected path on stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if deps.touched.repoKey != "/repo/.git" || deps.touched.path != "/repo/.worktrees/alpha" {
		t.Fatalf("expected state touch after successful switch, got %#v", deps.touched)
	}
}

func TestRunSwitchPathMatchesNameAndPrintsSelectedPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		touched: &touchRecord{},
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta"},
		},
	}

	code := Run(context.Background(), []string{"switch-path", "alpha"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "/repo/.worktrees/alpha\n" {
		t.Fatalf("expected selected path on stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if deps.touched.repoKey != "/repo/.git" || deps.touched.path != "/repo/.worktrees/alpha" {
		t.Fatalf("expected state touch after successful named switch, got %#v", deps.touched)
	}
}

func TestRunSwitchPathRejectsAmbiguousName(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
			{Path: "/repo/.worktrees/alpine", BranchLabel: "alpine"},
		},
	}

	code := Run(context.Background(), []string{"switch-path", "alp"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("ambiguous worktree match")) {
		t.Fatalf("expected ambiguous-match message, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
}

func TestRunSwitchPathRejectsUnknownName(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
	}

	code := Run(context.Background(), []string{"switch-path", "gamma"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`no worktree matches "gamma"`)) {
		t.Fatalf("expected no-match message, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
}

func TestRunListPrintsMenuWithoutPrompt(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		touched: &touchRecord{},
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true, CreatedAt: 10},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", CreatedAt: 20},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta", CreatedAt: 30},
		},
		state: map[string]map[string]int64{
			"/repo/.git": {
				"/repo/.worktrees/alpha": 10,
				"/repo/.worktrees/beta":  20,
			},
		},
	}

	code := Run(context.Background(), []string{"list"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() == "" {
		t.Fatal("expected list output on stdout")
	}
	if strings.Index(stdout.String(), "ACTIVE main /repo") > strings.Index(stdout.String(), "/repo/.worktrees/alpha") {
		t.Fatalf("expected main before alpha in creation ordering, got %q", stdout.String())
	}
	if strings.Index(stdout.String(), "/repo/.worktrees/alpha") > strings.Index(stdout.String(), "/repo/.worktrees/beta") {
		t.Fatalf("expected alpha before beta in creation ordering, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "[1] ACTIVE main /repo") {
		t.Fatalf("expected ACTIVE status in list output, got %q", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte("Select a worktree")) {
		t.Fatalf("expected no prompt in list output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunListShowsDirtyStatuses(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		touched: &touchRecord{},
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true, IsDirty: true, CreatedAt: 10},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", IsDirty: true, CreatedAt: 20},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta", CreatedAt: 30},
		},
	}

	code := Run(context.Background(), []string{"list"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "[1] ACTIVE* main /repo") {
		t.Fatalf("expected dirty active status in list output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "[2] DIRTY  alpha /repo/.worktrees/alpha") {
		t.Fatalf("expected dirty non-current status in list output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunNewPathPrintsSelectedPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		createPath: "/repo/.worktrees/alpha",
		repoKey:    "/repo/.git",
		touched:    &touchRecord{},
	}

	code := Run(context.Background(), []string{"new-path", "alpha"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "/repo/.worktrees/alpha\n" {
		t.Fatalf("expected selected path on stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if deps.touched.repoKey != "/repo/.git" || deps.touched.path != "/repo/.worktrees/alpha" {
		t.Fatalf("expected state touch after successful create, got %#v", deps.touched)
	}
}

func TestRunNewPathUsesCanonicalRepoKeyForStateTouch(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey:    "/repo/.git",
		createPath: "/repo/.worktrees/current/.worktrees/alpha",
		touched:    &touchRecord{},
	}

	code := Run(context.Background(), []string{"new-path", "alpha"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if deps.touched.repoKey != "/repo/.git" {
		t.Fatalf("expected canonical repo key /repo/.git, got %#v", deps.touched)
	}
	if deps.touched.path != "/repo/.worktrees/current/.worktrees/alpha" {
		t.Fatalf("expected created path to be touched, got %#v", deps.touched)
	}
}

func TestRunNewPathUsesCanonicalRepoKeyWhenCreatedPathUsesAlias(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey:    "/real/repo/.git",
		createPath: "/alias/repo/.worktrees/alpha",
		touched:    &touchRecord{},
	}

	code := Run(context.Background(), []string{"new-path", "alpha"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if deps.touched.repoKey != "/real/repo/.git" {
		t.Fatalf("expected canonical repo key /real/repo/.git, got %#v", deps.touched)
	}
	if deps.touched.path != "/alias/repo/.worktrees/alpha" {
		t.Fatalf("expected created path to be touched, got %#v", deps.touched)
	}
}

func TestRunRmSelectsCandidateConfirmsAndPrintsHumanResult(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	removed := &removeCall{}
	deps := fakeDeps{
		repoKey:       "/repo/.git",
		defaultBranch: "main",
		removed:       removed,
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta", BranchRef: "refs/heads/beta"},
			{Path: "/repo/.worktrees/scratch", BranchLabel: "scratch"},
		},
		previews: map[string]git.RemovalPreview{
			"/repo/.worktrees/alpha": {
				Worktree:     worktree.Worktree{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
				BaseBranch:   "main",
				BranchMerged: true,
				DeleteBranch: true,
			},
			"/repo/.worktrees/beta": {
				Worktree:     worktree.Worktree{Path: "/repo/.worktrees/beta", BranchLabel: "beta", BranchRef: "refs/heads/beta"},
				BaseBranch:   "main",
				Dirty:        true,
				BranchMerged: false,
			},
			"/repo/.worktrees/scratch": {
				Worktree:   worktree.Worktree{Path: "/repo/.worktrees/scratch", BranchLabel: "scratch"},
				BaseBranch: "main",
			},
		},
		removeResult: git.RemoveResult{
			WorktreePath:    "/repo/.worktrees/alpha",
			Branch:          "alpha",
			BaseBranch:      "main",
			RemovedWorktree: true,
			DeletedBranch:   true,
		},
	}

	code := Run(context.Background(), []string{"rm"}, strings.NewReader("1\ny\n"), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Safe to delete") ||
		!strings.Contains(stderr.String(), "Review before deleting") ||
		!strings.Contains(stderr.String(), "Not safe to delete") {
		t.Fatalf("expected grouped removal list on stderr, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "✅ alpha") ||
		!strings.Contains(stderr.String(), "⚠️ scratch") ||
		!strings.Contains(stderr.String(), "🛑 beta") {
		t.Fatalf("expected human-readable candidate rows on stderr, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "\n    /repo/.worktrees/alpha\n") ||
		!strings.Contains(stderr.String(), "\n    /repo/.worktrees/scratch\n") ||
		!strings.Contains(stderr.String(), "\n    /repo/.worktrees/beta\n") {
		t.Fatalf("expected candidate paths on their own line, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "✅ Safe to delete") ||
		!strings.Contains(stderr.String(), "branch alpha (already merged into main)") ||
		!strings.Contains(stderr.String(), "Delete this worktree? [y/N]:") {
		t.Fatalf("expected safe summary card on stderr, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "removed worktree") || !strings.Contains(stdout.String(), "deleted branch") {
		t.Fatalf("expected human-readable removal output, got %q", stdout.String())
	}
	if removed.item.Path != "/repo/.worktrees/alpha" || removed.opts.BaseBranch != "main" || removed.opts.Force {
		t.Fatalf("expected selected removal call, got %#v", removed)
	}
}

func TestRunRmDirtyCandidateShowsStopCardWithoutConfirming(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	removed := &removeCall{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta", BranchRef: "refs/heads/beta"},
		},
		previews: map[string]git.RemovalPreview{
			"/repo/.worktrees/beta": {
				Worktree:     worktree.Worktree{Path: "/repo/.worktrees/beta", BranchLabel: "beta", BranchRef: "refs/heads/beta"},
				BaseBranch:   "main",
				Dirty:        true,
				BranchMerged: false,
			},
		},
		removed: removed,
	}

	code := Run(context.Background(), []string{"rm", "beta"}, strings.NewReader(""), stdout, stderr, deps)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "🛑 Not safe to delete") ||
		!strings.Contains(stderr.String(), "uncommitted changes detected") ||
		!strings.Contains(stderr.String(), "rerun with --force") {
		t.Fatalf("expected stop card with next steps, got %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "Delete this worktree? [y/N]:") {
		t.Fatalf("expected dirty removal to stop before confirmation, got %q", stderr.String())
	}
	if removed.item.Path != "" {
		t.Fatalf("expected removal not to be called, got %#v", removed)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
}

func TestRunRmOutputsJSONWhenRequested(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
		},
		previews: map[string]git.RemovalPreview{
			"/repo/.worktrees/alpha": {
				Worktree:     worktree.Worktree{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
				BaseBranch:   "release/1.0",
				BranchMerged: true,
				DeleteBranch: true,
			},
		},
		removeResult: git.RemoveResult{
			WorktreePath:    "/repo/.worktrees/alpha",
			Branch:          "alpha",
			BaseBranch:      "release/1.0",
			RemovedWorktree: true,
			DeletedBranch:   true,
		},
	}

	code := Run(context.Background(), []string{"rm", "--json", "--base", "release/1.0", "alpha"}, strings.NewReader("y\n"), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"deleted_branch":true`) || !strings.Contains(stdout.String(), `"base_branch":"release/1.0"`) {
		t.Fatalf("expected json output, got %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatalf("expected confirmation details on stderr")
	}
}

func TestRunRmPassesForceToRemoval(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	removed := &removeCall{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
		},
		previews: map[string]git.RemovalPreview{
			"/repo/.worktrees/alpha": {
				Worktree:     worktree.Worktree{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
				BaseBranch:   "main",
				Dirty:        true,
				BranchMerged: false,
			},
		},
		removeResult: git.RemoveResult{
			WorktreePath:     "/repo/.worktrees/alpha",
			Branch:           "alpha",
			BaseBranch:       "main",
			RemovedWorktree:  true,
			KeptBranchReason: "not merged",
		},
		removed: removed,
	}

	code := Run(context.Background(), []string{"rm", "--force", "alpha"}, strings.NewReader("y\n"), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stderr.String(), "⚠️ Review before deleting") ||
		!strings.Contains(stderr.String(), "uncommitted changes will be lost") ||
		!strings.Contains(stderr.String(), "Delete this worktree? [y/N]:") {
		t.Fatalf("expected force mode warning card, got %q", stderr.String())
	}
	if !removed.opts.Force {
		t.Fatalf("expected force to be forwarded, got %#v", removed)
	}
}

func TestRunRmTreatsBaseBranchWorktreeAsReview(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	removed := &removeCall{}
	deps := fakeDeps{
		repoKey:       "/repo/.git",
		defaultBranch: "main",
		removed:       removed,
		worktrees: []worktree.Worktree{
			{Path: "/repo/.worktrees/detached", BranchLabel: "(detached)"},
			{Path: "/repo/.worktrees/main", BranchLabel: "main", BranchRef: "refs/heads/main"},
			{Path: "/repo/.worktrees/topic", BranchLabel: "topic", BranchRef: "refs/heads/topic", IsCurrent: true},
		},
		previews: map[string]git.RemovalPreview{
			"/repo/.worktrees/main": {
				Worktree:     worktree.Worktree{Path: "/repo/.worktrees/main", BranchLabel: "main", BranchRef: "refs/heads/main"},
				BaseBranch:   "main",
				BranchMerged: true,
				DeleteBranch: false,
			},
			"/repo/.worktrees/detached": {
				Worktree:   worktree.Worktree{Path: "/repo/.worktrees/detached", BranchLabel: "(detached)"},
				BaseBranch: "main",
			},
		},
		removeResult: git.RemoveResult{
			WorktreePath:     "/repo/.worktrees/main",
			Branch:           "main",
			BaseBranch:       "main",
			RemovedWorktree:  true,
			KeptBranchReason: "base branch",
		},
	}

	code := Run(context.Background(), []string{"rm"}, strings.NewReader("2\ny\n"), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if strings.Contains(stderr.String(), "Safe to delete\n[1] ✅ main") {
		t.Fatalf("did not expect main in safe group, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Review before deleting") ||
		!strings.Contains(stderr.String(), "⚠️ main  Base branch will be kept") {
		t.Fatalf("expected main to be rendered as review, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "\n    /repo/.worktrees/main\n") ||
		!strings.Contains(stderr.String(), "\n    /repo/.worktrees/detached\n") {
		t.Fatalf("expected review candidates to include paths, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Will keep:\n- branch main (not deleted because it is the base branch)") {
		t.Fatalf("expected base-branch summary, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "kept branch main (base branch)") {
		t.Fatalf("expected human result to keep main, got %q", stdout.String())
	}
	if removed.item.Path != "/repo/.worktrees/main" {
		t.Fatalf("expected main worktree to be selected, got %#v", removed)
	}
}

func TestRunSwitchPathContinuesWhenStateLoadFails(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true, CreatedAt: 10},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", CreatedAt: 20},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta", CreatedAt: 30},
		},
		loadErr: errors.New("state unavailable"),
		touched: &touchRecord{},
	}

	code := Run(context.Background(), []string{"switch-path", "2"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "/repo/.worktrees/alpha\n" {
		t.Fatalf("expected fallback ordering path on stdout, got %q", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("state load unavailable")) {
		t.Fatalf("expected state load warning, got %q", stderr.String())
	}
}

func TestRunSwitchPathContinuesWhenStateTouchFails(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true, CreatedAt: 10},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", CreatedAt: 20},
		},
		touchErr: errors.New("permission denied"),
		touched:  &touchRecord{},
	}

	code := Run(context.Background(), []string{"switch-path", "2"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "/repo/.worktrees/alpha\n" {
		t.Fatalf("expected path on stdout, got %q", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("state update skipped")) {
		t.Fatalf("expected state touch warning, got %q", stderr.String())
	}
}

func TestRunNewPathContinuesWhenStateTouchFails(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey:    "/repo/.git",
		createPath: "/repo/.worktrees/alpha",
		touchErr:   errors.New("permission denied"),
		touched:    &touchRecord{},
	}

	code := Run(context.Background(), []string{"new-path", "alpha"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "/repo/.worktrees/alpha\n" {
		t.Fatalf("expected path on stdout, got %q", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("state update skipped")) {
		t.Fatalf("expected state touch warning, got %q", stderr.String())
	}
}

func TestRunNewPathRejectsMissingName(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"new-path"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("missing worktree name")) {
		t.Fatalf("expected missing-name message, got %q", stderr.String())
	}
}

func TestRunNewPathReturnsNonRepoWhenRepoKeyLookupFails(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"new-path", "alpha"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{
		repoKeyErr: git.ErrNotGitRepository,
		createPath: "/repo/.worktrees/alpha",
	})

	if code != 3 {
		t.Fatalf("expected exit code 3, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("not a git repository")) {
		t.Fatalf("expected non-repo message, got %q", stderr.String())
	}
}

func TestRunRejectsExtraArgsAfterSwitchPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		touched: &touchRecord{},
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
	}

	code := Run(context.Background(), []string{"switch-path", "2", "junk"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("unexpected extra arguments")) {
		t.Fatalf("expected extra-args message, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
}

func TestRunNonRepoReturnsExit3(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"switch-path", "1"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{
		err: git.ErrNotGitRepository,
	})

	if code != 3 {
		t.Fatalf("expected exit code 3, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("not a git repository")) {
		t.Fatalf("expected non-repo message, got %q", stderr.String())
	}
}

func TestRunRejectsInvalidIndex(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"switch-path", "0"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("invalid worktree index")) {
		t.Fatalf("expected invalid index message, got %q", stderr.String())
	}
}

func TestRunRejectsOutOfRangeIndex(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		touched: &touchRecord{},
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
		},
	}

	code := Run(context.Background(), []string{"switch-path", "2"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("out of range")) {
		t.Fatalf("expected out-of-range message, got %q", stderr.String())
	}
}

func TestRunSwitchPathFzfModePrintsSelectedPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		touched: &touchRecord{},
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
		fzfSelected: worktree.Worktree{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		state: map[string]map[string]int64{
			"/repo/.git": {
				"/repo/.worktrees/alpha": 10,
			},
		},
	}

	code := Run(context.Background(), []string{"switch-path", "--fzf"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "/repo/.worktrees/alpha\n" {
		t.Fatalf("expected selected path on stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if deps.touched.repoKey != "/repo/.git" || deps.touched.path != "/repo/.worktrees/alpha" {
		t.Fatalf("expected state touch after successful fzf switch, got %#v", deps.touched)
	}
}

func TestRunSwitchPathFzfModeReturnsExit3WhenFzfMissing(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		touched: &touchRecord{},
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
		},
		fzfErr: ui.ErrFzfNotInstalled,
	}

	code := Run(context.Background(), []string{"switch-path", "--fzf"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 3 {
		t.Fatalf("expected exit code 3, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("fzf is not installed")) {
		t.Fatalf("expected missing fzf message, got %q", stderr.String())
	}
}

func TestRunSwitchPathFzfModeReturns130WhenCanceled(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		touched: &touchRecord{},
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
		fzfErr: ui.ErrSelectionCanceled,
	}

	code := Run(context.Background(), []string{"switch-path", "--fzf"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 130 {
		t.Fatalf("expected exit code 130, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
}

func TestRunRejectsExtraArgsAfterSwitchPathFzf(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"switch-path", "--fzf", "junk"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("unexpected extra arguments")) {
		t.Fatalf("expected extra-args message, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
}

func TestRunSwitchPathInteractiveSelectionWritesTUIToStderrAndPathToStdout(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		touched: &touchRecord{},
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
		fzfErr: ui.ErrFzfNotInstalled,
		state: map[string]map[string]int64{
			"/repo/.git": {
				"/repo/.worktrees/alpha": 10,
			},
		},
	}

	code := Run(context.Background(), nil, strings.NewReader("\x1b[B\r"), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "/repo/.worktrees/alpha\n" {
		t.Fatalf("expected selected path on stdout, got %q", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Enter to confirm")) {
		t.Fatalf("expected tui instructions on stderr, got %q", stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("* [1]        alpha /repo/.worktrees/alpha")) {
		t.Fatalf("expected active row on stderr, got %q", stderr.String())
	}
	if deps.touched.repoKey != "/repo/.git" || deps.touched.path != "/repo/.worktrees/alpha" {
		t.Fatalf("expected state touch after interactive switch, got %#v", deps.touched)
	}
}

func TestRunSwitchPathInteractiveSelectionPrefersFzfWhenAvailable(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		touched: &touchRecord{},
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta"},
		},
		fzfSelected: worktree.Worktree{Path: "/repo/.worktrees/beta", BranchLabel: "beta"},
		tuiSelected: worktree.Worktree{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
	}

	code := Run(context.Background(), nil, strings.NewReader("\x1b[B\r"), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "/repo/.worktrees/beta\n" {
		t.Fatalf("expected fzf-selected path on stdout, got %q", stdout.String())
	}
	if deps.touched.repoKey != "/repo/.git" || deps.touched.path != "/repo/.worktrees/beta" {
		t.Fatalf("expected state touch after fzf switch, got %#v", deps.touched)
	}
}

func TestRunSwitchPathInteractiveSelectionFallsBackToTUIWhenFzfMissing(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		touched: &touchRecord{},
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
		fzfErr: ui.ErrFzfNotInstalled,
	}

	code := Run(context.Background(), nil, strings.NewReader("\x1b[B\r"), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != "/repo/.worktrees/alpha\n" {
		t.Fatalf("expected tui-selected path on stdout, got %q", stdout.String())
	}
}

func TestRunSwitchPathInteractiveSelectionReturns130WhenFzfCanceled(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
		fzfErr: ui.ErrSelectionCanceled,
	}

	code := Run(context.Background(), nil, strings.NewReader("\x1b[B\r"), stdout, stderr, deps)

	if code != 130 {
		t.Fatalf("expected exit code 130, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
}

func TestRunSwitchPathInteractiveSelectionReturnsNonZeroOnEOFWithoutSelection(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
		},
		fzfErr: ui.ErrFzfNotInstalled,
	}

	code := Run(context.Background(), nil, strings.NewReader(""), stdout, stderr, deps)

	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
}

type fakeDeps struct {
	repoKey          string
	repoKeyErr       error
	worktrees        []worktree.Worktree
	err              error
	fzfSelected      worktree.Worktree
	fzfErr           error
	tuiSelected      worktree.Worktree
	tuiErr           error
	createPath       string
	createErr        error
	loadErr          error
	touchErr         error
	state            map[string]map[string]int64
	touched          *touchRecord
	defaultBranch    string
	defaultBranchErr error
	previews         map[string]git.RemovalPreview
	previewErr       error
	removeResult     git.RemoveResult
	removeErr        error
	removed          *removeCall
}

type touchRecord struct {
	repoKey string
	path    string
}

type removeCall struct {
	item worktree.Worktree
	opts git.RemoveOptions
}

func (f fakeDeps) CurrentRepoKey(context.Context) (string, error) {
	if f.repoKeyErr != nil {
		return "", f.repoKeyErr
	}
	return f.repoKey, nil
}

func (f fakeDeps) ListWorktrees(context.Context) (string, []worktree.Worktree, error) {
	if f.err != nil {
		return "", nil, f.err
	}
	return f.repoKey, append([]worktree.Worktree(nil), f.worktrees...), nil
}

func (f fakeDeps) SelectWorktreeWithFzf(context.Context, []worktree.Worktree) (worktree.Worktree, error) {
	if f.fzfErr != nil {
		return worktree.Worktree{}, f.fzfErr
	}
	if f.fzfSelected.Path != "" || f.fzfSelected.Index != 0 {
		return f.fzfSelected, nil
	}
	if len(f.worktrees) > 0 {
		return f.worktrees[0], nil
	}
	return worktree.Worktree{}, nil
}

func (f fakeDeps) SelectWorktreeWithTUI(in io.Reader, out io.Writer, items []worktree.Worktree) (worktree.Worktree, error) {
	if f.tuiErr != nil {
		return worktree.Worktree{}, f.tuiErr
	}
	if f.tuiSelected.Path != "" || f.tuiSelected.Index != 0 {
		return f.tuiSelected, nil
	}
	return ui.SelectWorktreeWithTUI(in, out, items, ui.OSRawMode{})
}

func (f fakeDeps) CreateWorktree(context.Context, string) (string, error) {
	if f.createErr != nil {
		return "", f.createErr
	}
	if f.createPath != "" {
		return f.createPath, nil
	}
	return "", nil
}

func (f fakeDeps) LoadWorktreeState(_ context.Context, repoKey string) (map[string]int64, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	if f.state == nil {
		return map[string]int64{}, nil
	}
	if got, ok := f.state[repoKey]; ok {
		out := make(map[string]int64, len(got))
		for k, v := range got {
			out[k] = v
		}
		return out, nil
	}
	return map[string]int64{}, nil
}

func (f fakeDeps) TouchWorktreeState(_ context.Context, repoKey, path string) error {
	if f.touchErr != nil {
		return f.touchErr
	}
	if f.touched != nil {
		f.touched.repoKey = repoKey
		f.touched.path = path
	}
	return nil
}

func (f fakeDeps) DefaultBranch(context.Context) (string, error) {
	if f.defaultBranchErr != nil {
		return "", f.defaultBranchErr
	}
	return f.defaultBranch, nil
}

func (f fakeDeps) PreviewRemoval(_ context.Context, item worktree.Worktree, _ string) (git.RemovalPreview, error) {
	if f.previewErr != nil {
		return git.RemovalPreview{}, f.previewErr
	}
	if got, ok := f.previews[item.Path]; ok {
		return got, nil
	}
	return git.RemovalPreview{}, nil
}

func (f fakeDeps) RemoveWorktree(_ context.Context, item worktree.Worktree, opts git.RemoveOptions) (git.RemoveResult, error) {
	if f.removed != nil {
		f.removed.item = item
		f.removed.opts = opts
	}
	if f.removeErr != nil {
		return git.RemoveResult{}, f.removeErr
	}
	return f.removeResult, nil
}
