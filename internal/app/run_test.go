package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ww/internal/git"
	"ww/internal/state"
	"ww/internal/tasknote"
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
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("init")) {
		t.Fatalf("expected help to mention init, got %q", got)
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

func TestRunInitPrintsShellSetupForLibexecLayout(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root := t.TempDir()
	helperPath := filepath.Join(root, "bin", "ww-helper")
	shellPath := filepath.Join(root, "libexec", "ww.sh")
	if err := os.MkdirAll(filepath.Dir(shellPath), 0o755); err != nil {
		t.Fatalf("mkdir libexec: %v", err)
	}
	if err := os.WriteFile(shellPath, []byte("ww() { :; }\n"), 0o644); err != nil {
		t.Fatalf("write ww.sh: %v", err)
	}

	restoreExecutablePath := executablePath
	restoreEvalSymlinks := evalSymlinks
	executablePath = func() (string, error) { return helperPath, nil }
	evalSymlinks = func(path string) (string, error) { return path, nil }
	defer func() {
		executablePath = restoreExecutablePath
		evalSymlinks = restoreEvalSymlinks
	}()

	code := Run(context.Background(), []string{"init", "zsh"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "WW_HELPER_BIN='"+helperPath+"'") {
		t.Fatalf("expected helper path in init output, got %q", got)
	}
	if got := stdout.String(); !strings.Contains(got, "source '"+shellPath+"'") {
		t.Fatalf("expected libexec ww.sh path in init output, got %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunInitPrintsShellSetupForSiblingLayout(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root := t.TempDir()
	helperPath := filepath.Join(root, "bin", "ww-helper")
	shellPath := filepath.Join(root, "bin", "ww.sh")
	if err := os.MkdirAll(filepath.Dir(shellPath), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.WriteFile(shellPath, []byte("ww() { :; }\n"), 0o644); err != nil {
		t.Fatalf("write ww.sh: %v", err)
	}

	restoreExecutablePath := executablePath
	restoreEvalSymlinks := evalSymlinks
	executablePath = func() (string, error) { return helperPath, nil }
	evalSymlinks = func(path string) (string, error) { return path, nil }
	defer func() {
		executablePath = restoreExecutablePath
		evalSymlinks = restoreEvalSymlinks
	}()

	code := Run(context.Background(), []string{"init", "bash"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "source '"+shellPath+"'") {
		t.Fatalf("expected sibling ww.sh path in init output, got %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunInitRejectsUnsupportedShell(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"init", "fish"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), `unsupported shell: "fish"`) {
		t.Fatalf("expected unsupported-shell message, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
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
	if !strings.Contains(stdout.String(), "│ INDEX │ STATUS") || !strings.Contains(stdout.String(), "│ PATH") {
		t.Fatalf("expected list header in output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "┌") || !strings.Contains(stdout.String(), "┼") || !strings.Contains(stdout.String(), "┘") {
		t.Fatalf("expected list divider in output, got %q", stdout.String())
	}
	if strings.Index(stdout.String(), "│ 1     │ [CURRENT]         │ main") > strings.Index(stdout.String(), "/repo/.worktrees/alpha") {
		t.Fatalf("expected main before alpha in creation ordering, got %q", stdout.String())
	}
	if strings.Index(stdout.String(), "/repo/.worktrees/alpha") > strings.Index(stdout.String(), "/repo/.worktrees/beta") {
		t.Fatalf("expected alpha before beta in creation ordering, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "│ 1     │ [CURRENT]         │ main") || !strings.Contains(stdout.String(), "│ /repo") {
		t.Fatalf("expected CURRENT status in list output, got %q", stdout.String())
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
	if !strings.Contains(stdout.String(), "│ INDEX │ STATUS") {
		t.Fatalf("expected list header in output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "│ 1     │ [CURRENT] [DIRTY] │ main") || !strings.Contains(stdout.String(), "│ /repo") {
		t.Fatalf("expected dirty active status in list output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "│ 2     │ [DIRTY]           │ alpha") || !strings.Contains(stdout.String(), "/repo/.worktrees/alpha") {
		t.Fatalf("expected dirty non-current status in list output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunListOutputsJSONWhenRequested(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true, CreatedAt: 10},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", IsDirty: true, CreatedAt: 20},
		},
	}

	code := Run(context.Background(), []string{"list", "--json"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	envelope := decodeEnvelope(t, stdout.String())
	if !envelope.OK {
		t.Fatalf("expected ok envelope, got %#v", envelope)
	}
	if envelope.Command != "list" {
		t.Fatalf("expected command list, got %#v", envelope)
	}
	if envelope.Error != nil {
		t.Fatalf("expected no error, got %#v", envelope.Error)
	}

	var items []struct {
		Path      string `json:"path"`
		Branch    string `json:"branch"`
		Dirty     bool   `json:"dirty"`
		Active    bool   `json:"active"`
		CreatedAt int64  `json:"created_at"`
	}
	decodeEnvelopeData(t, envelope, &items)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %#v", items)
	}
	if items[0].Path != "/repo" || items[0].Branch != "main" || !items[0].Active || items[0].Dirty || items[0].CreatedAt != 10 {
		t.Fatalf("unexpected first item: %#v", items[0])
	}
	if items[1].Path != "/repo/.worktrees/alpha" || items[1].Branch != "alpha" || items[1].Active || !items[1].Dirty || items[1].CreatedAt != 20 {
		t.Fatalf("unexpected second item: %#v", items[1])
	}
}

func TestRunListOutputsJSONErrorWhenRequested(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"list", "--json"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{
		err: git.ErrNotGitRepository,
	})

	if code != 3 {
		t.Fatalf("expected exit code 3, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.OK {
		t.Fatalf("expected error envelope, got %#v", envelope)
	}
	if envelope.Command != "list" {
		t.Fatalf("expected command list, got %#v", envelope)
	}
	if envelope.Error == nil {
		t.Fatalf("expected error details, got %#v", envelope)
	}
	if envelope.Error.Code != "NOT_GIT_REPO" || envelope.Error.ExitCode != 3 {
		t.Fatalf("unexpected error payload: %#v", envelope.Error)
	}
	if !strings.Contains(envelope.Error.Message, "not a git repository") {
		t.Fatalf("expected non-repo message, got %#v", envelope.Error)
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

func TestRunNewPathOutputsJSONWhenRequested(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		createPath: "/repo/.worktrees/alpha",
		repoKey:    "/repo/.git",
		touched:    &touchRecord{},
	}

	code := Run(context.Background(), []string{"new-path", "--json", "alpha"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	envelope := decodeEnvelope(t, stdout.String())
	if !envelope.OK {
		t.Fatalf("expected ok envelope, got %#v", envelope)
	}
	if envelope.Command != "new-path" {
		t.Fatalf("expected command new-path, got %#v", envelope)
	}
	if envelope.Error != nil {
		t.Fatalf("expected no error, got %#v", envelope.Error)
	}

	var data struct {
		WorktreePath string `json:"worktree_path"`
		Branch       string `json:"branch"`
	}
	decodeEnvelopeData(t, envelope, &data)

	if data.WorktreePath != "/repo/.worktrees/alpha" || data.Branch != "alpha" {
		t.Fatalf("unexpected data payload: %#v", data)
	}
	if deps.touched.repoKey != "/repo/.git" || deps.touched.path != "/repo/.worktrees/alpha" {
		t.Fatalf("expected state touch after successful create, got %#v", deps.touched)
	}
}

func TestRunNewPathJSONIncludesMetadataInputs(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	recorded := &recordWorktreeCall{}
	notePath := filepath.Join(t.TempDir(), "git-private", "ww", "task-note.md")
	deps := fakeDeps{
		createPath:      "/repo/.worktrees/alpha",
		repoKey:         "/repo/.git",
		touched:         &touchRecord{},
		recorded:        recorded,
		worktreeGitPath: notePath,
	}

	code := Run(context.Background(), []string{"new-path", "--json", "--label", "agent:claude", "--ttl", "24h", "alpha"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if recorded.repoKey != "/repo/.git" || recorded.path != "/repo/.worktrees/alpha" {
		t.Fatalf("expected metadata record call, got %#v", recorded)
	}
	if recorded.meta.Label != "agent:claude" || recorded.meta.TTL != "24h" || recorded.meta.CreatedAt == 0 {
		t.Fatalf("unexpected recorded metadata: %#v", recorded.meta)
	}
}

func TestRunNewPathLabelCreatesTaskNote(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	recorded := &recordWorktreeCall{}
	gitPath := &gitPathCall{}
	notePath := filepath.Join(t.TempDir(), "git-private", "ww", "task-note.md")
	deps := fakeDeps{
		createPath:          "/repo/.worktrees/alpha",
		repoKey:             "/repo/.git",
		touched:             &touchRecord{},
		recorded:            recorded,
		worktreeGitPath:     notePath,
		worktreeGitPathCall: gitPath,
	}

	code := Run(context.Background(), []string{"new-path", "--label", "task:fix-login", "-m", "Fix the login redirect loop", "alpha"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if gitPath.worktreePath != "/repo/.worktrees/alpha" || gitPath.rel != "ww/task-note.md" {
		t.Fatalf("expected worktree git path lookup, got %#v", gitPath)
	}
	if recorded.meta.Label != "task:fix-login" {
		t.Fatalf("expected recorded label, got %#v", recorded.meta)
	}

	note, err := tasknote.ReadFile(notePath)
	if err != nil {
		t.Fatalf("expected task note to be readable: %v", err)
	}
	if note.TaskLabel != "task:fix-login" {
		t.Fatalf("expected task label %q, got %q", "task:fix-login", note.TaskLabel)
	}
	if note.Branch != "alpha" {
		t.Fatalf("expected branch %q, got %q", "alpha", note.Branch)
	}
	if note.CreatedAt.IsZero() {
		t.Fatalf("expected created_at to be set, got %#v", note)
	}
	if note.Intent != "Fix the login redirect loop" {
		t.Fatalf("expected intent %q, got %q", "Fix the login redirect loop", note.Intent)
	}
	if !strings.Contains(note.Body, "Created by ww.") {
		t.Fatalf("expected scaffold body in note, got %q", note.Body)
	}
}

func TestRunNewPathLabelWithoutMessageHasNoIntent(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	gitPath := &gitPathCall{}
	notePath := filepath.Join(t.TempDir(), "git-private", "ww", "task-note.md")
	deps := fakeDeps{
		createPath:          "/repo/.worktrees/alpha",
		repoKey:             "/repo/.git",
		touched:             &touchRecord{},
		recorded:            &recordWorktreeCall{},
		worktreeGitPath:     notePath,
		worktreeGitPathCall: gitPath,
	}

	code := Run(context.Background(), []string{"new-path", "--label", "task:fix-login", "alpha"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	note, err := tasknote.ReadFile(notePath)
	if err != nil {
		t.Fatalf("expected task note to be readable: %v", err)
	}
	if note.Intent != "" {
		t.Fatalf("expected empty intent without -m flag, got %q", note.Intent)
	}
}

func TestRunNewPathWithoutLabelDoesNotCreateTaskNote(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	gitPath := &gitPathCall{}
	notePath := filepath.Join(t.TempDir(), "git-private", "ww", "task-note.md")
	deps := fakeDeps{
		createPath:          "/repo/.worktrees/alpha",
		repoKey:             "/repo/.git",
		touched:             &touchRecord{},
		worktreeGitPath:     notePath,
		worktreeGitPathCall: gitPath,
	}

	code := Run(context.Background(), []string{"new-path", "alpha"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if gitPath.worktreePath != "" || gitPath.rel != "" {
		t.Fatalf("expected no worktree git path lookup, got %#v", gitPath)
	}
	if _, err := os.Stat(notePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected task note to be absent, got err=%v", err)
	}
}

func TestRunNewPathRejectsInvalidTTL(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"new-path", "--json", "--ttl", "later", "alpha"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.OK {
		t.Fatalf("expected error envelope, got %#v", envelope)
	}
	if envelope.Error == nil || envelope.Error.Code != "INVALID_DURATION" {
		t.Fatalf("unexpected error payload: %#v", envelope.Error)
	}
}

func TestRunNewPathRejectsEmptyLabel(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"new-path", "--json", "--label", "", "alpha"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.OK {
		t.Fatalf("expected error envelope, got %#v", envelope)
	}
	if envelope.Error == nil || envelope.Error.Code != "INVALID_ARGUMENTS" {
		t.Fatalf("unexpected error payload: %#v", envelope.Error)
	}
	if !strings.Contains(envelope.Error.Message, "label") {
		t.Fatalf("expected label validation message, got %#v", envelope.Error)
	}
}

func TestRunNewPathOutputsJSONErrorWhenMissingName(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"new-path", "--json"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.OK {
		t.Fatalf("expected error envelope, got %#v", envelope)
	}
	if envelope.Command != "new-path" {
		t.Fatalf("expected command new-path, got %#v", envelope)
	}
	if envelope.Error == nil || envelope.Error.ExitCode != 2 {
		t.Fatalf("unexpected error payload: %#v", envelope.Error)
	}
	if !strings.Contains(envelope.Error.Message, "missing worktree name") {
		t.Fatalf("expected missing-name message, got %#v", envelope.Error)
	}
}

func TestRunListJSONIncludesMetadataFields(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", CreatedAt: 20},
		},
		metadata: map[string]map[string]state.WorktreeMetadata{
			"/repo/.git": {
				"/repo/.worktrees/alpha": {
					LastUsedAt: 30,
					CreatedAt:  20,
					Label:      "agent:claude",
					TTL:        "24h",
				},
			},
		},
	}

	code := Run(context.Background(), []string{"list", "--json"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	envelope := decodeEnvelope(t, stdout.String())
	var items []struct {
		Path       string `json:"path"`
		LastUsedAt int64  `json:"last_used_at"`
		Label      string `json:"label"`
		TTL        string `json:"ttl"`
	}
	decodeEnvelopeData(t, envelope, &items)

	if len(items) != 1 {
		t.Fatalf("expected one item, got %#v", items)
	}
	if items[0].Path != "/repo/.worktrees/alpha" || items[0].LastUsedAt != 30 || items[0].Label != "agent:claude" || items[0].TTL != "24h" {
		t.Fatalf("unexpected metadata payload: %#v", items[0])
	}
}

func TestRunListVerboseShowsLabelAndTTL(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", CreatedAt: 20},
		},
		metadata: map[string]map[string]state.WorktreeMetadata{
			"/repo/.git": {
				"/repo/.worktrees/alpha": {
					LastUsedAt: time.Unix(200, 0).UnixNano(),
					CreatedAt:  20,
					Label:      "agent:claude",
					TTL:        "24h",
				},
			},
		},
	}

	code := Run(context.Background(), []string{"list", "--verbose"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "agent:claude") || !strings.Contains(stdout.String(), "24h") {
		t.Fatalf("expected verbose output to include metadata, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunListVerboseShowsIntent(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	notePath := filepath.Join(t.TempDir(), "git-private", "ww", "task-note.md")
	if err := tasknote.WriteFile(notePath, tasknote.Note{
		TaskLabel: "task:fix-login",
		Branch:    "alpha",
		CreatedAt: time.Date(2026, 3, 24, 12, 34, 56, 0, time.UTC),
		Intent:    "Fix the login redirect loop",
		Body:      "Created by ww.",
	}); err != nil {
		t.Fatalf("write note: %v", err)
	}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", CreatedAt: 20},
		},
		metadata: map[string]map[string]state.WorktreeMetadata{
			"/repo/.git": {
				"/repo/.worktrees/alpha": {
					Label: "task:fix-login",
				},
			},
		},
		worktreeGitPath: notePath,
	}

	code := Run(context.Background(), []string{"list", "--verbose"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "intent=Fix the login") {
		t.Fatalf("expected intent in verbose output, got %q", out)
	}
}

func TestRunListKeepsDefaultOutputFocusedOnWorktrees(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", CreatedAt: 20},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta", CreatedAt: 30},
		},
		metadata: map[string]map[string]state.WorktreeMetadata{
			"/repo/.git": {
				"/repo/.worktrees/alpha": {
					Label: "task:fix-login",
				},
			},
		},
	}

	code := Run(context.Background(), []string{"list"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "│ INDEX │ STATUS") {
		t.Fatalf("expected list header in output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "│ 1     │                   │ alpha") ||
		!strings.Contains(stdout.String(), "/repo/.worktrees/alpha") ||
		!strings.Contains(stdout.String(), "│ 2     │                   │ beta") ||
		!strings.Contains(stdout.String(), "/repo/.worktrees/beta") {
		t.Fatalf("expected worktree rows in list output, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "task=") || strings.Contains(stdout.String(), "label=") {
		t.Fatalf("expected metadata hidden from default list output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunListFiltersByLabelAndStale(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", CreatedAt: 20},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta", CreatedAt: 30},
		},
		metadata: map[string]map[string]state.WorktreeMetadata{
			"/repo/.git": {
				"/repo/.worktrees/alpha": {
					LastUsedAt: 1,
					CreatedAt:  20,
					Label:      "agent:claude",
					TTL:        "24h",
				},
				"/repo/.worktrees/beta": {
					LastUsedAt: time.Now().UnixNano(),
					CreatedAt:  30,
					Label:      "manual",
				},
			},
		},
	}

	code := Run(context.Background(), []string{"list", "--json", "--filter", "label=agent:claude", "--filter", "stale=7d"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	envelope := decodeEnvelope(t, stdout.String())
	var items []struct {
		Path string `json:"path"`
	}
	decodeEnvelopeData(t, envelope, &items)

	if len(items) != 1 || items[0].Path != "/repo/.worktrees/alpha" {
		t.Fatalf("expected only alpha to match, got %#v", items)
	}
}

func TestRunListRejectsInvalidFilter(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"list", "--json", "--filter", "branch=alpha"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.OK {
		t.Fatalf("expected error envelope, got %#v", envelope)
	}
	if envelope.Error == nil || envelope.Error.Code != "INVALID_FILTER" {
		t.Fatalf("unexpected error payload: %#v", envelope.Error)
	}
}

func TestRunGCRequiresAtLeastOneRule(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"gc", "--json"}, bytes.NewReader(nil), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.OK {
		t.Fatalf("expected error envelope, got %#v", envelope)
	}
	if envelope.Command != "gc" {
		t.Fatalf("expected command gc, got %#v", envelope)
	}
	if envelope.Error == nil || envelope.Error.Code != "GC_RULE_REQUIRED" {
		t.Fatalf("unexpected error payload: %#v", envelope.Error)
	}
}

func TestRunGCDryRunJSONSummarizesMatches(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta", BranchRef: "refs/heads/beta"},
		},
		metadata: map[string]map[string]state.WorktreeMetadata{
			"/repo/.git": {
				"/repo/.worktrees/alpha": {CreatedAt: 1, TTL: "24h"},
				"/repo/.worktrees/beta":  {LastUsedAt: 1},
			},
		},
	}

	code := Run(context.Background(), []string{"gc", "--json", "--dry-run", "--ttl-expired", "--idle", "7d"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	envelope := decodeEnvelope(t, stdout.String())
	var data struct {
		Summary struct {
			Matched int `json:"matched"`
			Removed int `json:"removed"`
			Skipped int `json:"skipped"`
		} `json:"summary"`
		Items []struct {
			Path         string   `json:"path"`
			MatchedRules []string `json:"matched_rules"`
			Action       string   `json:"action"`
		} `json:"items"`
	}
	decodeEnvelopeData(t, envelope, &data)

	if data.Summary.Matched != 2 || data.Summary.Removed != 0 || data.Summary.Skipped != 0 {
		t.Fatalf("unexpected summary: %#v", data.Summary)
	}
	if len(data.Items) != 2 {
		t.Fatalf("expected two dry-run items, got %#v", data.Items)
	}
}

func TestRunGCSkipsDirtyAndActiveWorktrees(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	removed := &removeCall{}
	deps := fakeDeps{
		repoKey:       "/repo/.git",
		defaultBranch: "main",
		removed:       removed,
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/dirty", BranchLabel: "dirty", BranchRef: "refs/heads/dirty", IsDirty: true},
			{Path: "/repo/.worktrees/clean", BranchLabel: "clean", BranchRef: "refs/heads/clean"},
		},
		metadata: map[string]map[string]state.WorktreeMetadata{
			"/repo/.git": {
				"/repo":                  {CreatedAt: 1, TTL: "24h"},
				"/repo/.worktrees/dirty": {CreatedAt: 1, TTL: "24h"},
				"/repo/.worktrees/clean": {CreatedAt: 1, TTL: "24h"},
			},
		},
		previews: map[string]git.RemovalPreview{
			"/repo/.worktrees/dirty": {
				Worktree:   worktree.Worktree{Path: "/repo/.worktrees/dirty", BranchLabel: "dirty", BranchRef: "refs/heads/dirty"},
				BaseBranch: "main",
				Dirty:      true,
			},
			"/repo/.worktrees/clean": {
				Worktree:     worktree.Worktree{Path: "/repo/.worktrees/clean", BranchLabel: "clean", BranchRef: "refs/heads/clean"},
				BaseBranch:   "main",
				BranchMerged: true,
				DeleteBranch: true,
			},
		},
		removeResult: git.RemoveResult{
			WorktreePath:    "/repo/.worktrees/clean",
			Branch:          "clean",
			BaseBranch:      "main",
			RemovedWorktree: true,
			DeletedBranch:   true,
		},
	}

	code := Run(context.Background(), []string{"gc", "--json", "--ttl-expired"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if removed.item.Path != "/repo/.worktrees/clean" {
		t.Fatalf("expected only clean worktree to be removed, got %#v", removed)
	}

	envelope := decodeEnvelope(t, stdout.String())
	var data struct {
		Summary struct {
			Matched int `json:"matched"`
			Removed int `json:"removed"`
			Skipped int `json:"skipped"`
		} `json:"summary"`
		Items []struct {
			Path   string `json:"path"`
			Action string `json:"action"`
			Reason string `json:"reason"`
		} `json:"items"`
	}
	decodeEnvelopeData(t, envelope, &data)

	if data.Summary.Matched != 3 || data.Summary.Removed != 1 || data.Summary.Skipped != 2 {
		t.Fatalf("unexpected summary: %#v", data.Summary)
	}
}

func TestRunGCForceAllowsDirtyRemoval(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	removed := &removeCall{}
	deps := fakeDeps{
		repoKey:       "/repo/.git",
		defaultBranch: "main",
		removed:       removed,
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/dirty", BranchLabel: "dirty", BranchRef: "refs/heads/dirty"},
		},
		metadata: map[string]map[string]state.WorktreeMetadata{
			"/repo/.git": {
				"/repo/.worktrees/dirty": {CreatedAt: 1, TTL: "24h"},
			},
		},
		previews: map[string]git.RemovalPreview{
			"/repo/.worktrees/dirty": {
				Worktree:     worktree.Worktree{Path: "/repo/.worktrees/dirty", BranchLabel: "dirty", BranchRef: "refs/heads/dirty"},
				BaseBranch:   "main",
				Dirty:        true,
				BranchMerged: true,
				DeleteBranch: true,
			},
		},
		removeResult: git.RemoveResult{
			WorktreePath:    "/repo/.worktrees/dirty",
			Branch:          "dirty",
			BaseBranch:      "main",
			RemovedWorktree: true,
			DeletedBranch:   true,
		},
	}

	code := Run(context.Background(), []string{"gc", "--json", "--ttl-expired", "--force"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if removed.item.Path != "/repo/.worktrees/dirty" || !removed.opts.Force {
		t.Fatalf("expected forced removal call, got %#v", removed)
	}
}

func TestRunGCMergedUsesBaseBranchResolution(t *testing.T) {
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
		},
		previews: map[string]git.RemovalPreview{
			"/repo/.worktrees/alpha": {
				Worktree:     worktree.Worktree{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
				BaseBranch:   "main",
				BranchMerged: true,
				DeleteBranch: true,
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

	code := Run(context.Background(), []string{"gc", "--json", "--merged"}, bytes.NewReader(nil), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if removed.opts.BaseBranch != "main" {
		t.Fatalf("expected default base branch to be used, got %#v", removed)
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

func TestRunRemoveSummaryIncludesTaskContext(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	removed := &removeCall{}
	notePath := filepath.Join(t.TempDir(), "git-private", "ww", "task-note.md")
	if err := tasknote.WriteFile(notePath, tasknote.Note{
		TaskLabel: "task:fix-login",
		Branch:    "alpha",
		CreatedAt: time.Date(2026, 3, 24, 12, 34, 56, 0, time.UTC),
		Intent:    "Fix the login redirect loop",
		Body:      "Created by ww.",
	}); err != nil {
		t.Fatalf("write task note: %v", err)
	}

	deps := fakeDeps{
		repoKey:         "/repo/.git",
		defaultBranch:   "main",
		removed:         removed,
		worktreeGitPath: notePath,
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", BranchRef: "refs/heads/main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
		},
		metadata: map[string]map[string]state.WorktreeMetadata{
			"/repo/.git": {
				"/repo/.worktrees/alpha": {
					Label: "task:fix-login",
				},
			},
		},
		previews: map[string]git.RemovalPreview{
			"/repo/.worktrees/alpha": {
				Worktree:     worktree.Worktree{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
				BaseBranch:   "main",
				BranchMerged: true,
				DeleteBranch: true,
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

	code := Run(context.Background(), []string{"rm", "alpha"}, strings.NewReader("y\n"), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if strings.Contains(stderr.String(), "Task context:") {
		t.Fatalf("expected task wording removed from summary, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Workspace context:") ||
		!strings.Contains(stderr.String(), "Fix the login redirect loop") {
		t.Fatalf("expected human workspace context in summary, got %q", stderr.String())
	}
}

func TestRunRemoveSummaryShowsWeakerBoundaryState(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey:       "/repo/.git",
		defaultBranch: "main",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", BranchRef: "refs/heads/main", IsCurrent: true},
			{Path: "/repo/.worktrees/scratch", BranchLabel: "scratch"},
		},
		previews: map[string]git.RemovalPreview{
			"/repo/.worktrees/scratch": {
				Worktree:   worktree.Worktree{Path: "/repo/.worktrees/scratch", BranchLabel: "scratch"},
				BaseBranch: "main",
			},
		},
	}

	code := Run(context.Background(), []string{"rm", "scratch"}, strings.NewReader("n\n"), stdout, stderr, deps)

	if code != 130 {
		t.Fatalf("expected exit code 130, got %d", code)
	}
	if !strings.Contains(stderr.String(), "no saved workspace context") {
		t.Fatalf("expected missing-context boundary warning, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "detached from a branch") {
		t.Fatalf("expected detached boundary warning, got %q", stderr.String())
	}
}

func TestRunRmOutputsJSONErrorForDirtyTargetWhenRequested(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta", BranchRef: "refs/heads/beta"},
		},
		previews: map[string]git.RemovalPreview{
			"/repo/.worktrees/beta": {
				Worktree:   worktree.Worktree{Path: "/repo/.worktrees/beta", BranchLabel: "beta", BranchRef: "refs/heads/beta"},
				BaseBranch: "main",
				Dirty:      true,
			},
		},
		defaultBranch: "main",
	}

	code := Run(context.Background(), []string{"rm", "--json", "--non-interactive", "beta"}, strings.NewReader(""), stdout, stderr, deps)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.OK {
		t.Fatalf("expected error envelope, got %#v", envelope)
	}
	if envelope.Command != "rm" {
		t.Fatalf("expected command rm, got %#v", envelope)
	}
	if envelope.Error == nil {
		t.Fatalf("expected error payload, got %#v", envelope)
	}
	if envelope.Error.Code != "WORKTREE_DIRTY" || envelope.Error.ExitCode != 1 {
		t.Fatalf("unexpected error payload: %#v", envelope.Error)
	}
	if !strings.Contains(envelope.Error.Message, "uncommitted changes") {
		t.Fatalf("expected dirty-worktree message, got %#v", envelope.Error)
	}
}

func TestRunRmNonInteractiveSkipsConfirmation(t *testing.T) {
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
				BranchMerged: true,
				DeleteBranch: true,
			},
		},
		defaultBranch: "main",
		removeResult: git.RemoveResult{
			WorktreePath:    "/repo/.worktrees/alpha",
			Branch:          "alpha",
			BaseBranch:      "main",
			RemovedWorktree: true,
			DeletedBranch:   true,
		},
		removed: removed,
	}

	code := Run(context.Background(), []string{"rm", "--non-interactive", "alpha"}, strings.NewReader(""), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if strings.Contains(stderr.String(), "Delete this worktree? [y/N]:") {
		t.Fatalf("expected confirmation prompt to be skipped, got %q", stderr.String())
	}
	if removed.item.Path != "/repo/.worktrees/alpha" || removed.opts.BaseBranch != "main" || removed.opts.Force {
		t.Fatalf("expected removal call, got %#v", removed)
	}
	if !strings.Contains(stdout.String(), "removed worktree") {
		t.Fatalf("expected human-readable removal output, got %q", stdout.String())
	}
}

func TestRunRmNonInteractiveRejectsMissingTargetWhenMultipleCandidatesExist(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta", BranchRef: "refs/heads/beta"},
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
				BranchMerged: true,
				DeleteBranch: true,
			},
		},
		defaultBranch: "main",
	}

	code := Run(context.Background(), []string{"rm", "--json", "--non-interactive"}, strings.NewReader(""), stdout, stderr, deps)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.OK {
		t.Fatalf("expected error envelope, got %#v", envelope)
	}
	if envelope.Command != "rm" {
		t.Fatalf("expected command rm, got %#v", envelope)
	}
	if envelope.Error == nil {
		t.Fatalf("expected error payload, got %#v", envelope)
	}
	if envelope.Error.Code != "AMBIGUOUS_MATCH" || envelope.Error.ExitCode != 2 {
		t.Fatalf("unexpected error payload: %#v", envelope.Error)
	}
	if !strings.Contains(envelope.Error.Message, "must specify a target") {
		t.Fatalf("expected ambiguous-target message, got %#v", envelope.Error)
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

	envelope := decodeEnvelope(t, stdout.String())
	if !envelope.OK {
		t.Fatalf("expected ok envelope, got %#v", envelope)
	}
	if envelope.Command != "rm" {
		t.Fatalf("expected command rm, got %#v", envelope)
	}

	var data struct {
		WorktreePath    string `json:"worktree_path"`
		BaseBranch      string `json:"base_branch"`
		DeletedBranch   bool   `json:"deleted_branch"`
		RemovedWorktree bool   `json:"removed_worktree"`
	}
	decodeEnvelopeData(t, envelope, &data)

	if data.WorktreePath != "/repo/.worktrees/alpha" || data.BaseBranch != "release/1.0" || !data.DeletedBranch || !data.RemovedWorktree {
		t.Fatalf("unexpected json payload, got %#v", data)
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

func TestRunRmCleanupRejectsTarget(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"rm", "--cleanup", "alpha"}, strings.NewReader(""), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--cleanup") {
		t.Fatalf("expected cleanup validation message, got %q", stderr.String())
	}
}

func TestRunRmCleanupRejectsJSON(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(context.Background(), []string{"rm", "--cleanup", "--json"}, strings.NewReader(""), stdout, stderr, fakeDeps{})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected json error to avoid stderr output, got %q", stderr.String())
	}
	envelope := decodeEnvelope(t, stdout.String())
	if envelope.OK || envelope.Error == nil {
		t.Fatalf("expected json error envelope, got %#v", envelope)
	}
	if !strings.Contains(envelope.Error.Message, "--cleanup") {
		t.Fatalf("expected cleanup validation message, got %#v", envelope.Error)
	}
}

func TestRunRmCleanupRemovesSelectedCandidate(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	removed := &removeCall{}
	deps := fakeDeps{
		repoKey: "/repo/.git",
		worktrees: []worktree.Worktree{
			{Path: "/repo", BranchLabel: "main", IsCurrent: true},
			{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
			{Path: "/repo/.worktrees/beta", BranchLabel: "beta", BranchRef: "refs/heads/beta"},
		},
		previews: map[string]git.RemovalPreview{
			"/repo/.worktrees/alpha": {
				Worktree:     worktree.Worktree{Path: "/repo/.worktrees/alpha", BranchLabel: "alpha", BranchRef: "refs/heads/alpha"},
				BaseBranch:   "main",
				BranchMerged: true,
				DeleteBranch: true,
			},
			"/repo/.worktrees/beta": {
				Worktree:   worktree.Worktree{Path: "/repo/.worktrees/beta", BranchLabel: "beta", BranchRef: "refs/heads/beta"},
				BaseBranch: "main",
			},
		},
		defaultBranch: "main",
		removeResult: git.RemoveResult{
			WorktreePath:    "/repo/.worktrees/alpha",
			Branch:          "alpha",
			BaseBranch:      "main",
			RemovedWorktree: true,
			DeletedBranch:   true,
		},
		removed: removed,
	}

	code := Run(context.Background(), []string{"rm", "--cleanup"}, strings.NewReader("1\ny\n"), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Cleanup mode") {
		t.Fatalf("expected cleanup banner, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "2 workspaces available: 1 safe, 1 review, 0 blocked.") {
		t.Fatalf("expected cleanup availability summary, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Select a workspace to review [number, Enter to finish]:") ||
		!strings.Contains(stderr.String(), "Delete this worktree? [y/N]:") {
		t.Fatalf("expected cleanup confirmation prompt, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Cleanup finished. Removed 1 workspace. 1 workspace still listed.") {
		t.Fatalf("expected cleanup completion summary, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "removed worktree /repo/.worktrees/alpha") {
		t.Fatalf("expected cleanup removal output, got %q", stdout.String())
	}
	if removed.item.Path != "/repo/.worktrees/alpha" {
		t.Fatalf("expected cleanup to remove alpha, got %#v", removed)
	}
}

func TestRunRmCleanupDirtyCandidateStopsBeforeRemoval(t *testing.T) {
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
		defaultBranch: "main",
		removed:       removed,
	}

	code := Run(context.Background(), []string{"rm", "--cleanup"}, strings.NewReader("1\n\n\n"), stdout, stderr, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stderr.String(), "uncommitted changes detected") ||
		!strings.Contains(stderr.String(), "Select a workspace to review [number, Enter to finish]:") {
		t.Fatalf("expected cleanup stop card, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Press Enter to return to cleanup list:") {
		t.Fatalf("expected cleanup continue prompt, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "1 workspace available: 0 safe, 0 review, 1 blocked.") {
		t.Fatalf("expected cleanup availability summary, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Cleanup finished. No workspaces removed. 1 workspace still listed.") {
		t.Fatalf("expected cleanup no-op summary, got %q", stderr.String())
	}
	if strings.Count(stderr.String(), "beta") < 2 {
		t.Fatalf("expected dirty workspace to remain visible in cleanup list, got %q", stderr.String())
	}
	if removed.item.Path != "" {
		t.Fatalf("expected cleanup not to remove dirty candidate, got %#v", removed)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
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
	if !bytes.Contains(stderr.Bytes(), []byte("* [1]                   alpha  /repo/.worktrees/alpha")) {
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

type envelope struct {
	OK      bool            `json:"ok"`
	Command string          `json:"command"`
	Data    json.RawMessage `json:"data"`
	Error   *envelopeError  `json:"error"`
}

type envelopeError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	ExitCode int    `json:"exit_code"`
}

type fakeDeps struct {
	repoKey             string
	repoKeyErr          error
	worktrees           []worktree.Worktree
	err                 error
	fzfSelected         worktree.Worktree
	fzfErr              error
	tuiSelected         worktree.Worktree
	tuiErr              error
	createPath          string
	createErr           error
	loadErr             error
	touchErr            error
	state               map[string]map[string]int64
	metadata            map[string]map[string]state.WorktreeMetadata
	touched             *touchRecord
	recorded            *recordWorktreeCall
	worktreeGitPath     string
	worktreeGitPathErr  error
	worktreeGitPathCall *gitPathCall
	defaultBranch       string
	defaultBranchErr    error
	previews            map[string]git.RemovalPreview
	previewErr          error
	removeResult        git.RemoveResult
	removeErr           error
	removed             *removeCall
}

type touchRecord struct {
	repoKey string
	path    string
}

type recordWorktreeCall struct {
	repoKey string
	path    string
	meta    state.WorktreeMetadata
}

type gitPathCall struct {
	worktreePath string
	rel          string
}

type removeCall struct {
	item worktree.Worktree
	opts git.RemoveOptions
}

func decodeEnvelope(t *testing.T, raw string) envelope {
	t.Helper()

	var got envelope
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("expected valid JSON envelope, got %q: %v", raw, err)
	}
	return got
}

func decodeEnvelopeData(t *testing.T, env envelope, target any) {
	t.Helper()

	if len(env.Data) == 0 {
		t.Fatalf("expected data payload, got %#v", env)
	}
	if err := json.Unmarshal(env.Data, target); err != nil {
		t.Fatalf("expected decodable data payload %q: %v", string(env.Data), err)
	}
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
	if got, ok := f.metadata[repoKey]; ok {
		out := make(map[string]int64, len(got))
		for path, meta := range got {
			out[path] = meta.LastUsedAt
		}
		return out, nil
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

func (f fakeDeps) LoadWorktreeMetadata(_ context.Context, repoKey string) (map[string]state.WorktreeMetadata, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	if got, ok := f.metadata[repoKey]; ok {
		out := make(map[string]state.WorktreeMetadata, len(got))
		for path, meta := range got {
			out[path] = meta
		}
		return out, nil
	}
	if got, ok := f.state[repoKey]; ok {
		out := make(map[string]state.WorktreeMetadata, len(got))
		for path, lastUsedAt := range got {
			out[path] = state.WorktreeMetadata{LastUsedAt: lastUsedAt}
		}
		return out, nil
	}
	return map[string]state.WorktreeMetadata{}, nil
}

func (f fakeDeps) RecordWorktreeState(_ context.Context, repoKey, path string, meta state.WorktreeMetadata) error {
	if f.touchErr != nil {
		return f.touchErr
	}
	if f.recorded != nil {
		f.recorded.repoKey = repoKey
		f.recorded.path = path
		f.recorded.meta = meta
	}
	return nil
}

func (f fakeDeps) WorktreeGitPath(_ context.Context, worktreePath string, rel string) (string, error) {
	if f.worktreeGitPathCall != nil {
		f.worktreeGitPathCall.worktreePath = worktreePath
		f.worktreeGitPathCall.rel = rel
	}
	if f.worktreeGitPathErr != nil {
		return "", f.worktreeGitPathErr
	}
	return f.worktreeGitPath, nil
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
