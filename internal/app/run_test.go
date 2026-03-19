package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"wt/internal/git"
	"wt/internal/ui"
	"wt/internal/worktree"
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

func TestRunSwitchPathPrintsSelectedPath(t *testing.T) {
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
	if stdout.String() != "/repo/.worktrees/beta\n" {
		t.Fatalf("expected selected path on stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if deps.touched.repoKey != "/repo/.git" || deps.touched.path != "/repo/.worktrees/beta" {
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
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta"},
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
	if strings.Index(stdout.String(), "/repo/.worktrees/beta") > strings.Index(stdout.String(), "/repo/.worktrees/alpha") {
		t.Fatalf("expected MRU ordering in list output, got %q", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte("Select a worktree")) {
		t.Fatalf("expected no prompt in list output, got %q", stdout.String())
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

func TestRunSwitchPathContinuesWhenStateLoadFails(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta"},
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
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha"},
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
	if !bytes.Contains(stderr.Bytes(), []byte("> [2]   alpha /repo/.worktrees/alpha")) {
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
	repoKey     string
	repoKeyErr  error
	worktrees   []worktree.Worktree
	err         error
	fzfSelected worktree.Worktree
	fzfErr      error
	tuiSelected worktree.Worktree
	tuiErr      error
	createPath  string
	createErr   error
	loadErr     error
	touchErr    error
	state       map[string]map[string]int64
	touched     *touchRecord
}

type touchRecord struct {
	repoKey string
	path    string
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
