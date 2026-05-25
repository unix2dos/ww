package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"ww/internal/config"
	"ww/internal/git"
	"ww/internal/state"
	"ww/internal/syncignored"
	"ww/internal/tasknote"
	"ww/internal/ui"
	"ww/internal/worktree"
)

// syncIgnoredFn is the function used by runNewPath to sync ignored files into
// a freshly created worktree. It is a package-level variable so tests can
// replace it with a noop (see run_sync_test.go).
var syncIgnoredFn = func(ctx context.Context, mainRoot, target string, opts syncignored.Options) (syncignored.Result, error) {
	return syncignored.Sync(ctx, git.ExecRunner{}, mainRoot, target, opts)
}

// loadSyncConfigFn returns the user's config. Package-level var for the same
// test-override reason as syncIgnoredFn.
var loadSyncConfigFn = func() (config.Config, error) {
	return config.LoadDefault()
}

type Deps interface {
	CurrentRepoKey(ctx context.Context) (string, error)
	ListWorktrees(ctx context.Context) (string, []worktree.Worktree, int, error)
	SelectWorktreeWithFzf(ctx context.Context, items []worktree.Worktree) (worktree.Worktree, error)
	SelectWorktreeWithTUI(in io.Reader, out io.Writer, items []worktree.Worktree) (worktree.Worktree, error)
	CreateWorktree(ctx context.Context, name string) (string, error)
	LoadWorktreeState(ctx context.Context, repoKey string) (map[string]int64, error)
	LoadWorktreeMetadata(ctx context.Context, repoKey string) (map[string]state.WorktreeMetadata, error)
	TouchWorktreeState(ctx context.Context, repoKey, path string) error
	RecordWorktreeState(ctx context.Context, repoKey, path string, meta state.WorktreeMetadata) error
	WorktreeGitPath(ctx context.Context, worktreePath string, rel string) (string, error)
	DefaultBranch(ctx context.Context) (string, error)
	DefaultStatusBase(ctx context.Context) (git.StatusBase, error)
	AnnotateExtendedStatus(ctx context.Context, items []worktree.Worktree, baseBranch string) error
	PreviewRemoval(ctx context.Context, item worktree.Worktree, baseBranch string) (git.RemovalPreview, error)
	RemoveWorktree(ctx context.Context, item worktree.Worktree, opts git.RemoveOptions) (git.RemoveResult, error)
	LastCommitSubject(ctx context.Context, worktreePath string) (string, error)
	DetachedUniqueCommits(ctx context.Context, worktreePath, baseBranch string) (int, error)
}

type appError struct {
	Code     string
	Message  string
	ExitCode int
}

func (e appError) Error() string {
	return e.Message
}

type RealDeps struct{}

var executablePath = os.Executable
var evalSymlinks = filepath.EvalSymlinks

var defaultStateStore struct {
	once  sync.Once
	store *state.Store
	err   error
}

func ensureStore() (*state.Store, error) {
	defaultStateStore.once.Do(func() {
		defaultStateStore.store, defaultStateStore.err = state.NewStore()
	})
	return defaultStateStore.store, defaultStateStore.err
}

func (d RealDeps) ListWorktrees(ctx context.Context) (string, []worktree.Worktree, int, error) {
	return git.ListWorktrees(ctx, git.ExecRunner{})
}

func (d RealDeps) CurrentRepoKey(ctx context.Context) (string, error) {
	return git.CurrentRepoKey(ctx, git.ExecRunner{})
}

func (d RealDeps) SelectWorktreeWithFzf(ctx context.Context, items []worktree.Worktree) (worktree.Worktree, error) {
	return ui.SelectWorktreeWithFzf(ctx, items, ui.ExecRunner{})
}

func (d RealDeps) SelectWorktreeWithTUI(in io.Reader, out io.Writer, items []worktree.Worktree) (worktree.Worktree, error) {
	return ui.SelectWorktreeWithTUI(in, out, items, ui.OSRawMode{})
}

func (d RealDeps) CreateWorktree(ctx context.Context, name string) (string, error) {
	return git.CreateWorktree(ctx, git.ExecRunner{}, name)
}

func (d RealDeps) LoadWorktreeState(_ context.Context, repoKey string) (map[string]int64, error) {
	store, err := ensureStore()
	if err != nil {
		return nil, err
	}
	return store.Load(repoKey)
}

func (d RealDeps) LoadWorktreeMetadata(_ context.Context, repoKey string) (map[string]state.WorktreeMetadata, error) {
	store, err := ensureStore()
	if err != nil {
		return nil, err
	}
	return store.LoadMetadata(repoKey)
}

func (d RealDeps) TouchWorktreeState(_ context.Context, repoKey, path string) error {
	store, err := ensureStore()
	if err != nil {
		return err
	}
	return store.Touch(repoKey, path)
}

func (d RealDeps) RecordWorktreeState(_ context.Context, repoKey, path string, meta state.WorktreeMetadata) error {
	store, err := ensureStore()
	if err != nil {
		return err
	}
	return store.RecordWorktree(repoKey, path, meta)
}

func (d RealDeps) WorktreeGitPath(ctx context.Context, worktreePath string, rel string) (string, error) {
	return git.WorktreeGitPath(ctx, git.ExecRunner{}, worktreePath, rel)
}

func (d RealDeps) DefaultBranch(ctx context.Context) (string, error) {
	return git.DefaultBranch(ctx, git.ExecRunner{})
}

func (d RealDeps) DefaultStatusBase(ctx context.Context) (git.StatusBase, error) {
	return git.DefaultStatusBase(ctx, git.ExecRunner{})
}

func (d RealDeps) AnnotateExtendedStatus(ctx context.Context, items []worktree.Worktree, baseBranch string) error {
	return git.AnnotateExtendedStatus(ctx, git.ExecRunner{}, items, baseBranch)
}

func (d RealDeps) PreviewRemoval(ctx context.Context, item worktree.Worktree, baseBranch string) (git.RemovalPreview, error) {
	return git.PreviewRemoval(ctx, git.ExecRunner{}, item, baseBranch)
}

func (d RealDeps) RemoveWorktree(ctx context.Context, item worktree.Worktree, opts git.RemoveOptions) (git.RemoveResult, error) {
	return git.RemoveWorktree(ctx, git.ExecRunner{}, item, opts)
}

func (d RealDeps) LastCommitSubject(ctx context.Context, worktreePath string) (string, error) {
	return git.LastCommitSubject(ctx, git.ExecRunner{}, worktreePath)
}

func (d RealDeps) DetachedUniqueCommits(ctx context.Context, worktreePath, baseBranch string) (int, error) {
	return git.DetachedUniqueCommits(ctx, git.ExecRunner{}, worktreePath, baseBranch)
}

func Run(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
	if deps == nil {
		deps = &RealDeps{}
	}

	if len(args) == 0 {
		return runSwitchPath(ctx, args, in, out, errOut, deps)
	}

	switch args[0] {
	case "--help", "-h", "help":
		printHelperHelp(out)
		return 0
	case "init":
		return runInit(args[1:], out, errOut)
	case "switch-path":
		return runSwitchPath(ctx, args[1:], in, out, errOut, deps)
	case "new-path":
		return runNewPath(ctx, args[1:], out, errOut, deps)
	case "list":
		return runList(ctx, args[1:], out, errOut, deps)
	case "gc":
		return runGC(ctx, args[1:], out, errOut, deps)
	case "rm":
		return runRemove(ctx, args[1:], in, out, errOut, deps)
	case "version":
		return runVersion(args[1:], out, errOut)
	case "mcp":
		return runMCP(ctx, args[1:], errOut, deps)
	default:
		return runSwitchPath(ctx, args, in, out, errOut, deps)
	}
}

// runMCP dispatches `ww-helper mcp <subcommand>`. Currently the only
// supported subcommand is `serve`. The whole MCP server lives in package
// internal/mcp; this function is the thin CLI entrypoint.
//
// stdout is reserved for the MCP JSON-RPC transport; only errOut receives
// human-readable diagnostics. We never call writeJSONSuccess/writeCommandError
// here because those write to stdout — and even outside an active server
// session, an MCP-aware caller should still be able to invoke
// `ww-helper mcp` defensively without seeing stray JSON envelopes.
func runMCP(ctx context.Context, args []string, errOut io.Writer, deps Deps) int {
	if len(args) != 1 || args[0] != "serve" {
		fmt.Fprintln(errOut, "usage: ww-helper mcp serve")
		return 2
	}
	if MCPServe == nil {
		fmt.Fprintln(errOut, "mcp serve: server is not wired in this binary")
		return 1
	}
	if err := MCPServe(ctx, deps, binaryVersion, errOut); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return 0
}

// MCPServe is the entry point for the MCP server. It is set by
// cmd/ww-helper/main.go at startup to break the import cycle:
// internal/mcp depends on internal/app for ListData/NewPathData/etc.,
// so internal/app cannot import internal/mcp directly.
//
// The variable stays nil in test binaries and tools that don't link the
// MCP server; runMCP guards against that case explicitly.
var MCPServe func(ctx context.Context, deps Deps, binaryVersion string, errOut io.Writer) error

func runVersion(args []string, out io.Writer, errOut io.Writer) int {
	jsonMode := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonMode = true
		default:
			return writeCommandError("version", out, errOut, jsonMode, appError{
				Code:     "input.invalid_argument",
				Message:  fmt.Sprintf("unknown option: %s", arg),
				ExitCode: 2,
			})
		}
	}

	if jsonMode {
		return writeJSONSuccess(out, "version", VersionData())
	}

	fmt.Fprintf(out, "ww-helper %s (protocol %s)\n", humanVersionLabel(), protocolVersion)
	return 0
}

func humanVersionLabel() string {
	label := binaryVersion
	commit, dirty := buildMetadata()
	if label == "dev" && commit != "" {
		label += "+" + commit
	}
	if dirty {
		label += "-dirty"
	}
	return label
}

func runInit(args []string, out io.Writer, errOut io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(errOut, "usage: ww-helper init <zsh|bash>")
		return 2
	}

	switch args[0] {
	case "zsh", "bash":
	default:
		fmt.Fprintf(errOut, "unsupported shell: %q\n", args[0])
		return 2
	}

	helperPath, shellPath, err := resolveInitPaths()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	fmt.Fprintf(out, "WW_HELPER_BIN=%s\n", shellQuote(helperPath))
	fmt.Fprintf(out, "source %s\n", shellQuote(shellPath))
	return 0
}

// annotateExtendedStatusBestEffort calls AnnotateExtendedStatus if DefaultStatusBase
// can be resolved. Errors are swallowed — if git commands fail, the list shows
// with basic info only.
func annotateExtendedStatusBestEffort(ctx context.Context, deps Deps, items []worktree.Worktree) string {
	statusBase, err := deps.DefaultStatusBase(ctx)
	if err != nil || statusBase.Ref == "" {
		return ""
	}
	_ = deps.AnnotateExtendedStatus(ctx, items, statusBase.Ref)
	return statusBase.Ref
}

func decorateDetachedWorktreesForSelection(ctx context.Context, deps Deps, items []worktree.Worktree, baseBranch string) {
	for i := range items {
		if detached, ok := listDetachedPresentation(ctx, deps, items[i], baseBranch); ok {
			items[i] = worktreeWithDetachedPresentation(items[i], detached)
		}
	}
}

func runSwitchPath(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
	if len(args) > 0 && args[0] == "--fzf" {
		if len(args) > 1 {
			fmt.Fprintf(errOut, "unexpected extra arguments: %s\n", strings.Join(args[1:], " "))
			return 2
		}

		repoKey, items, _, warn, err := orderedWorktrees(ctx, deps)
		if err != nil {
			return writeWorktreeError(errOut, err)
		}
		warnStateIssue(errOut, warn)

		baseBranch := annotateExtendedStatusBestEffort(ctx, deps, items)
		decorateDetachedWorktreesForSelection(ctx, deps, items, baseBranch)

		selected, err := deps.SelectWorktreeWithFzf(ctx, items)
		if err != nil {
			switch {
			case errors.Is(err, ui.ErrFzfNotInstalled):
				fmt.Fprintln(errOut, "fzf is not installed")
				return 3
			case errors.Is(err, ui.ErrSelectionCanceled):
				return 130
			default:
				fmt.Fprintln(errOut, err)
				return 1
			}
		}

		fmt.Fprintln(out, selected.Path)
		warnStateIssue(errOut, touchWorktreeStateBestEffort(ctx, deps, repoKey, selected.Path))
		return 0
	}

	if len(args) == 0 {
		repoKey, items, _, warn, err := orderedWorktrees(ctx, deps)
		if err != nil {
			return writeWorktreeError(errOut, err)
		}
		warnStateIssue(errOut, warn)

		baseBranch := annotateExtendedStatusBestEffort(ctx, deps, items)
		decorateDetachedWorktreesForSelection(ctx, deps, items, baseBranch)

		selected, err := selectInteractiveWorktree(ctx, in, errOut, items, deps, false)
		if err != nil {
			return writeSelectionError(errOut, err)
		}
		fmt.Fprintln(out, selected.Path)
		warnStateIssue(errOut, touchWorktreeStateBestEffort(ctx, deps, repoKey, selected.Path))
		return 0
	}

	if len(args) > 1 {
		fmt.Fprintf(errOut, "unexpected extra arguments: %s\n", strings.Join(args[1:], " "))
		return 2
	}

	repoKey, items, _, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeWorktreeError(errOut, err)
	}
	warnStateIssue(errOut, warn)

	index, err := strconv.Atoi(args[0])
	if err == nil {
		if index <= 0 {
			fmt.Fprintf(errOut, "invalid worktree index: %q\n", args[0])
			return 2
		}
		selected, ok := selectByIndex(items, index)
		if !ok {
			fmt.Fprintf(errOut, "worktree index %d out of range\n", index)
			return 2
		}

		fmt.Fprintln(out, selected.Path)
		warnStateIssue(errOut, touchWorktreeStateBestEffort(ctx, deps, repoKey, selected.Path))
		return 0
	}

	selected, err := worktree.Match(items, args[0])
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}

	fmt.Fprintln(out, selected.Path)
	warnStateIssue(errOut, touchWorktreeStateBestEffort(ctx, deps, repoKey, selected.Path))
	return 0
}

type listConfig struct {
	json    bool
	verbose bool
	help    bool
}

type listEntry struct {
	item worktree.Worktree
	meta state.WorktreeMetadata
}

func runList(ctx context.Context, args []string, out io.Writer, errOut io.Writer, deps Deps) int {
	cfg, err := parseListArgs(args)
	if err != nil {
		return writeCommandError("list", out, errOut, cfg.json, err)
	}
	if cfg.help {
		printListHelp(out)
		return 0
	}

	if cfg.json {
		views, _, err := ListData(ctx, deps, ListOptions{})
		if err != nil {
			return writeCommandError("list", out, errOut, cfg.json, err)
		}
		// `views` may be nil when the empty case fires below; `[]` envelope is
		// preserved exactly, including its protocol shape.
		if views == nil {
			views = []WorktreeView{}
		}
		return writeJSONSuccess(out, "list", views)
	}

	// Human path: keep the existing rendering, including verbose detail and
	// state-load warnings.
	repoKey, items, metadata, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeCommandError("list", out, errOut, cfg.json, err)
	}
	warnStateIssue(errOut, warn)

	statusBaseRef := annotateExtendedStatusForList(ctx, deps, items)
	displayRoot := mainWorktreeRootFromRepoKey(repoKey)

	entries := decorateListEntries(items, metadata)
	if len(entries) == 0 {
		return writeCommandError("list", out, errOut, cfg.json, appError{
			Code:     "worktree.not_found",
			Message:  "no worktrees available",
			ExitCode: 1,
		})
	}

	tableEntries := make([]ui.ListTableEntry, 0, len(entries))
	for _, entry := range entries {
		tableEntries = append(tableEntries, listTableEntry(ctx, deps, entry, cfg.verbose, statusBaseRef, displayRoot))
	}
	fmt.Fprintln(out, ui.FormatListTableWithOptions(tableEntries, ui.ListTableOptions{
		ShowEmptyOptionalColumns: cfg.verbose,
	}))

	worktrees := make([]worktree.Worktree, 0, len(entries))
	for _, entry := range entries {
		worktrees = append(worktrees, entry.item)
	}
	fmt.Fprintln(out, ui.FormatSummary(worktrees))
	return 0
}

func annotateExtendedStatusForList(ctx context.Context, deps Deps, items []worktree.Worktree) string {
	statusBase, err := deps.DefaultStatusBase(ctx)
	if err != nil || statusBase.Ref == "" {
		return ""
	}
	_ = deps.AnnotateExtendedStatus(ctx, items, statusBase.Ref)
	return statusBase.Ref
}

func listTableEntry(ctx context.Context, deps Deps, entry listEntry, verbose bool, statusBaseRef string, displayRoot string) ui.ListTableEntry {
	item := entry.item
	if !verbose {
		item.Path = listDisplayPath(item.Path, displayRoot)
	}
	parts := make([]string, 0, 2)
	status := ""
	if detached, ok := listDetachedPresentation(ctx, deps, entry.item, statusBaseRef); ok {
		item = worktreeWithDetachedPresentation(item, detached)
		status = detached.status
		if detached.detail != "" {
			parts = append(parts, detached.detail)
		}
	}
	if verboseDetail := listVerboseDetail(ctx, deps, entry, verbose, statusBaseRef); verboseDetail != "" {
		parts = append(parts, verboseDetail)
	}
	return ui.ListTableEntry{
		Worktree: item,
		Status:   status,
		Detail:   strings.Join(parts, "\n"),
	}
}

func listDisplayPath(path string, displayRoot string) string {
	if path == "" {
		return ""
	}

	cleanPath := filepath.Clean(path)
	if displayRoot != "" {
		if rel, ok := relativePathWithin(filepath.Clean(displayRoot), cleanPath); ok {
			return rel
		}
	}

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		if rel, ok := relativePathWithin(filepath.Clean(home), cleanPath); ok {
			if rel == "." {
				return "~"
			}
			return filepath.Join("~", rel)
		}
	}

	return path
}

func relativePathWithin(root string, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	if rel == "." {
		return ".", true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", false
	}
	return rel, true
}

func worktreeWithDetachedPresentation(item worktree.Worktree, detached detachedListPresentation) worktree.Worktree {
	item.BranchLabel = detached.branch
	item.StatusLabel = detached.status
	return item
}

type detachedListPresentation struct {
	branch string
	status string
	detail string
}

func listDetachedPresentation(ctx context.Context, deps Deps, item worktree.Worktree, baseBranch string) (detachedListPresentation, bool) {
	if !item.IsDetached || baseBranch == "" {
		return detachedListPresentation{}, false
	}

	uniqueCommits, err := deps.DetachedUniqueCommits(ctx, item.Path, baseBranch)
	if err != nil {
		return detachedListPresentation{}, false
	}

	hasLocalChanges := item.IsDirty || item.Staged+item.Unstaged+item.Untracked > 0
	if uniqueCommits == 0 {
		presentation := detachedListPresentation{branch: "temporary", status: "[IDLE]"}
		if hasLocalChanges {
			presentation.status = ""
			presentation.detail = "local changes"
		}
		return presentation, true
	}

	summary := fmt.Sprintf("%d commits", uniqueCommits)
	if uniqueCommits == 1 {
		summary = "1 commit"
	}
	if hasLocalChanges {
		summary += " + local changes"
	}

	subject, err := deps.LastCommitSubject(ctx, item.Path)
	if err != nil || subject == "" {
		return detachedListPresentation{branch: "unbranched", detail: summary}, true
	}
	return detachedListPresentation{branch: "unbranched", detail: summary + "\nlast commit: " + subject}, true
}

func listVerboseDetail(ctx context.Context, deps Deps, entry listEntry, verbose bool, statusBaseRef string) string {
	if !verbose {
		return ""
	}

	parts := make([]string, 0, 5)
	if statusBaseRef != "" {
		parts = append(parts, "status_base_ref="+statusBaseRef)
	}
	if entry.meta.Label != "" {
		parts = append(parts, "label="+entry.meta.Label)
		note, err := readTaskNote(ctx, deps, entry.item.Path, entry.meta.Label)
		if err == nil && note.Intent != "" {
			parts = append(parts, "intent="+note.Intent)
		}
	}
	if entry.meta.TTL != "" {
		parts = append(parts, "ttl="+entry.meta.TTL)
	}
	if entry.meta.LastUsedAt != 0 {
		parts = append(parts, fmt.Sprintf("last_used_at=%d", entry.meta.LastUsedAt))
	}
	return strings.Join(parts, "  ")
}

type newPathConfig struct {
	json       bool
	name       string
	label      string
	ttl        string
	message    string
	noSync     bool
	syncDryRun bool
}

func runNewPath(ctx context.Context, args []string, out io.Writer, errOut io.Writer, deps Deps) int {
	cfg, err := parseNewPathArgs(args)
	if err != nil {
		return writeCommandError("new-path", out, errOut, cfg.json, err)
	}

	if cfg.json {
		result, warnings, err := NewPathData(ctx, deps, NewPathOptions{
			Name:       cfg.name,
			Label:      cfg.label,
			TTL:        cfg.ttl,
			Message:    cfg.message,
			Sync:       !cfg.noSync,
			SyncDryRun: cfg.syncDryRun,
		})
		if err != nil {
			return writeCommandError("new-path", out, errOut, cfg.json, err)
		}
		return writeJSONSuccess(out, "new-path", result, warnings...)
	}

	repoKey, err := deps.CurrentRepoKey(ctx)
	if err != nil {
		return writeCommandError("new-path", out, errOut, cfg.json, err)
	}

	path, err := deps.CreateWorktree(ctx, cfg.name)
	if err != nil {
		return writeCommandError("new-path", out, errOut, cfg.json, err)
	}

	meta := state.WorktreeMetadata{
		CreatedAt: time.Now().UnixNano(),
		Label:     cfg.label,
		TTL:       cfg.ttl,
	}
	createdAt := time.Unix(0, meta.CreatedAt).UTC()

	fmt.Fprintln(out, path)
	warnStateIssue(errOut, recordWorktreeStateBestEffort(ctx, deps, repoKey, path, meta))
	warnStateIssue(errOut, createTaskNoteIfLabeled(ctx, deps, path, cfg.name, cfg.label, cfg.message, createdAt))
	warnStateIssue(errOut, touchWorktreeStateBestEffort(ctx, deps, repoKey, path))
	runSyncIgnored(ctx, errOut, repoKey, path, cfg)
	return 0
}

// runSyncIgnored copies git-ignored files from the main worktree into the
// freshly created one. It is best-effort: every error is downgraded to a
// warning on stderr so `ww new` always reports success if the worktree itself
// was created.
//
// Called only on the non-JSON success path; JSON callers (machines) don't
// need this implicit behaviour and can manage sync themselves.
func runSyncIgnored(ctx context.Context, errOut io.Writer, repoKey, newPath string, cfg newPathConfig) {
	if cfg.noSync {
		return
	}

	mainRoot := mainWorktreeRootFromRepoKey(repoKey)
	if mainRoot == "" || mainRoot == newPath {
		return
	}

	userCfg, cfgErr := loadSyncConfigFn()
	if cfgErr != nil {
		fmt.Fprintf(errOut, "warning: sync: %v\n", cfgErr)
		// Fall through with defaults.
	}

	if !userCfg.Sync.SyncEnabled() {
		return
	}

	opts := syncignored.Options{
		Enabled:   true,
		Blacklist: userCfg.Sync.EffectiveBlacklist(syncignored.DefaultBlacklist),
		DryRun:    cfg.syncDryRun,
	}
	if v := userCfg.Sync.EffectiveMaxFileSize(); v > 0 {
		opts.MaxFileSize = v
	}

	res, err := syncIgnoredFn(ctx, mainRoot, newPath, opts)
	if err != nil {
		fmt.Fprintf(errOut, "warning: sync: %v\n", err)
		return
	}

	printSyncResult(errOut, res)
}

// mainWorktreeRootFromRepoKey extracts the main worktree's root directory
// from a git "repo key" (which is the absolute path to the .git directory or
// gitfile). Returns "" if the shape is unexpected.
func mainWorktreeRootFromRepoKey(repoKey string) string {
	if repoKey == "" {
		return ""
	}
	if filepath.Base(repoKey) == ".git" {
		return filepath.Dir(repoKey)
	}
	return ""
}

// printSyncResult writes a short human-readable summary of what was (or would
// be) copied. Silent when nothing interesting happened.
func printSyncResult(errOut io.Writer, res syncignored.Result) {
	if len(res.Copied) == 0 && len(res.Skipped) == 0 {
		return
	}
	prefix := "synced"
	if res.DryRun {
		prefix = "[dry-run] would sync"
	}
	if len(res.Copied) > 0 {
		preview := res.Copied
		const maxPreview = 5
		if len(preview) > maxPreview {
			preview = preview[:maxPreview]
			fmt.Fprintf(errOut, "%s %d ignored files (%s, ...)\n",
				prefix, len(res.Copied), strings.Join(preview, ", "))
		} else {
			fmt.Fprintf(errOut, "%s %d ignored files (%s)\n",
				prefix, len(res.Copied), strings.Join(preview, ", "))
		}
	}
	if res.DryRun {
		for _, s := range res.Skipped {
			switch s.Reason {
			case syncignored.SkipTooLarge:
				fmt.Fprintf(errOut, "[dry-run] skip %s (%d bytes, exceeds max_file_size)\n", s.Path, s.Size)
			case syncignored.SkipBlacklisted:
				fmt.Fprintf(errOut, "[dry-run] skip %s (blacklisted)\n", s.Path)
			}
		}
	}
}

func createTaskNoteIfLabeled(ctx context.Context, deps Deps, worktreePath, branch, label, message string, createdAt time.Time) error {
	if label == "" {
		return nil
	}

	notePath, err := deps.WorktreeGitPath(ctx, worktreePath, "ww/task-note.md")
	if err != nil {
		return fmt.Errorf("task note skipped: %w", err)
	}

	note := tasknote.Note{
		TaskLabel: label,
		Branch:    branch,
		CreatedAt: createdAt,
		Intent:    message,
		Body:      "Created by ww.",
	}
	if err := tasknote.WriteFile(notePath, note); err != nil {
		return fmt.Errorf("task note skipped: %w", err)
	}
	return nil
}

func readTaskNote(ctx context.Context, deps Deps, worktreePath, label string) (tasknote.Note, error) {
	if label == "" {
		return tasknote.Note{}, fmt.Errorf("task label is required")
	}
	notePath, err := deps.WorktreeGitPath(ctx, worktreePath, "ww/task-note.md")
	if err != nil {
		return tasknote.Note{}, err
	}
	return tasknote.ReadFile(notePath)
}

type gcConfig struct {
	ttlExpired bool
	idle       state.DurationSpec
	idleSet    bool
	merged     bool
	dryRun     bool
	force      bool
	json       bool
	base       string
}

type gcCandidate struct {
	entry        listEntry
	matchedRules []string
	preview      git.RemovalPreview
	hasPreview   bool
}

func runGC(ctx context.Context, args []string, out io.Writer, errOut io.Writer, deps Deps) int {
	cfg, err := parseGCArgs(args)
	if err != nil {
		return writeCommandError("gc", out, errOut, cfg.json, err)
	}

	if !cfg.json {
		// Surface state-load warnings to the human path; the data layer
		// silently handles them otherwise.
		_, _, _, warn, _ := orderedWorktrees(ctx, deps)
		warnStateIssue(errOut, warn)
	}

	result, err := GCData(ctx, deps, GCOptions{
		TTLExpired: cfg.ttlExpired,
		IdleSet:    cfg.idleSet,
		Idle:       cfg.idle,
		Merged:     cfg.merged,
		DryRun:     cfg.dryRun,
		Force:      cfg.force,
		Base:       cfg.base,
	})
	if err != nil {
		return writeCommandError("gc", out, errOut, cfg.json, err)
	}

	if cfg.json || cfg.dryRun {
		return writeJSONSuccess(out, "gc", result)
	}
	writeGCHuman(out, result.Items)
	return 0
}

type removeConfig struct {
	force   bool
	json    bool
	cleanup bool
	help    bool
	target  string
}

type removalCandidate struct {
	item          worktree.Worktree
	displayItem   worktree.Worktree
	preview       git.RemovalPreview
	idleTemporary bool
}

func newRemovalCandidate(ctx context.Context, deps Deps, item worktree.Worktree, preview git.RemovalPreview, baseBranch string) removalCandidate {
	candidate := removalCandidate{
		item:        item,
		displayItem: item,
		preview:     preview,
	}
	if detached, ok := listDetachedPresentation(ctx, deps, item, baseBranch); ok {
		candidate.displayItem = worktreeWithDetachedPresentation(item, detached)
		candidate.idleTemporary = detached.branch == "temporary" && detached.status == "[IDLE]" && !preview.Dirty
		if preview.Dirty {
			candidate.displayItem.StatusLabel = ""
		}
	}
	return candidate
}

func runRemove(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
	cfg, err := parseRemoveArgs(args)
	if err != nil {
		return writeCommandError("rm", out, errOut, cfg.json, err)
	}
	if cfg.help {
		printRemoveHelp(out)
		return 0
	}

	if cfg.json {
		result, err := RemoveData(ctx, deps, RemoveOptions{Target: cfg.target, Force: cfg.force})
		if err != nil {
			return writeCommandError("rm", out, errOut, true, err)
		}
		return writeJSONSuccess(out, "rm", result)
	}

	if cfg.cleanup {
		return runRemoveCleanup(ctx, in, out, errOut, deps)
	}

	// Human path keeps the interactive prompt flow.
	repoKey, items, _, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeCommandError("rm", out, errOut, cfg.json, err)
	}
	warnStateIssue(errOut, warn)

	if cfg.target != "" {
		if selected, matchErr := worktree.Match(items, cfg.target); matchErr == nil && selected.IsCurrent {
			fmt.Fprintln(errOut, "Cannot remove the current worktree. Switch first: ww go <name>")
			return 1
		}
	}

	candidates := filterNonCurrent(items)
	if len(candidates) == 0 {
		return writeCommandError("rm", out, errOut, cfg.json, appError{
			Code:     "worktree.not_found",
			Message:  "no removable worktrees available",
			ExitCode: 1,
		})
	}

	baseBranch, err := deps.DefaultBranch(ctx)
	if err != nil {
		return writeCommandError("rm", out, errOut, cfg.json, err)
	}
	displayRoot := mainWorktreeRootFromRepoKey(repoKey)

	previewed := make([]removalCandidate, 0, len(candidates))
	for _, item := range candidates {
		preview, err := deps.PreviewRemoval(ctx, item, baseBranch)
		if err != nil {
			return writeCommandError("rm", out, errOut, cfg.json, err)
		}
		previewed = append(previewed, newRemovalCandidate(ctx, deps, item, preview, baseBranch))
	}

	reader := bufio.NewReader(in)
	var selected removalCandidate

	if cfg.target != "" {
		selected, err = matchRemovalCandidate(previewed, cfg.target)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
	} else if len(previewed) == 1 {
		selected = previewed[0]
	} else {
		renderRemovalCandidates(errOut, previewed, displayRoot)
		index, err := readChoice(reader, errOut, "\n> ", len(previewed), 0)
		if err != nil {
			return writeSelectionError(errOut, err)
		}
		selected = previewed[index-1]
	}

	label := removalCandidateLabel(selected.displayItem)
	if selected.preview.Dirty && !cfg.force {
		fmt.Fprintf(errOut, "%s has uncommitted changes. Use --force to remove.\n", label)
		return 1
	}

	var prompt string
	if selected.preview.Dirty && cfg.force {
		prompt = fmt.Sprintf("Remove %s? Uncommitted changes will be lost. [y/N] ", label)
	} else {
		prompt = fmt.Sprintf("Remove %s? [y/N] ", label)
	}
	confirmed, err := confirmPrompt(reader, errOut, prompt)
	if err != nil {
		return writeSelectionError(errOut, err)
	}
	if !confirmed {
		return 130
	}

	result, err := deps.RemoveWorktree(ctx, selected.displayItem, git.RemoveOptions{
		BaseBranch: baseBranch,
		Force:      cfg.force,
	})
	if err != nil {
		return writeCommandError("rm", out, errOut, cfg.json, err)
	}

	writeRemoveHuman(out, result)
	return 0
}

func orderedWorktrees(ctx context.Context, deps Deps) (string, []worktree.Worktree, map[string]state.WorktreeMetadata, error, error) {
	repoKey, items, prunableCount, err := deps.ListWorktrees(ctx)
	if err != nil {
		return "", nil, nil, nil, err
	}
	var prunableWarn error
	if prunableCount > 0 {
		prunableWarn = fmt.Errorf("warning: skipped %d prunable worktree(s) with missing directories, run 'git worktree prune' to clean up", prunableCount)
	}
	metadata, err := deps.LoadWorktreeMetadata(ctx, repoKey)
	if err != nil {
		normalized := worktree.Normalize(items)
		return repoKey, normalized, map[string]state.WorktreeMetadata{}, errors.Join(prunableWarn, fmt.Errorf("state load unavailable: %w", err)), nil
	}
	for i := range items {
		meta := metadata[items[i].Path]
		items[i].LastUsedAt = meta.LastUsedAt
		if meta.CreatedAt != 0 {
			items[i].CreatedAt = meta.CreatedAt
		}
	}
	return repoKey, worktree.Normalize(items), metadata, prunableWarn, nil
}

func parseRemoveArgs(args []string) (removeConfig, error) {
	var cfg removeConfig
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--force":
			cfg.force = true
		case arg == "--json":
			cfg.json = true
		case arg == "--cleanup":
			cfg.cleanup = true
		case arg == "--help" || arg == "-h" || arg == "help":
			cfg.help = true
		case strings.HasPrefix(arg, "-"):
			return cfg, appError{Code: "input.invalid_argument", Message: fmt.Sprintf("unknown option: %s", arg), ExitCode: 2}
		default:
			if cfg.target != "" {
				return cfg, appError{Code: "input.invalid_argument", Message: fmt.Sprintf("unexpected extra arguments: %s", strings.Join(args[i:], " ")), ExitCode: 2}
			}
			cfg.target = arg
		}
	}
	if cfg.cleanup {
		switch {
		case cfg.json:
			return cfg, appError{Code: "input.invalid_argument", Message: "--cleanup is only available for human-readable rm", ExitCode: 2}
		case cfg.force:
			return cfg, appError{Code: "input.invalid_argument", Message: "--cleanup cannot be combined with --force", ExitCode: 2}
		case cfg.target != "":
			return cfg, appError{Code: "input.invalid_argument", Message: "--cleanup cannot be combined with a target", ExitCode: 2}
		}
	}
	if cfg.help && (cfg.json || cfg.force || cfg.cleanup || cfg.target != "") {
		return cfg, appError{Code: "input.invalid_argument", Message: "rm help cannot be combined with other arguments", ExitCode: 2}
	}
	return cfg, nil
}

func filterNonCurrent(items []worktree.Worktree) []worktree.Worktree {
	out := make([]worktree.Worktree, 0, len(items))
	for _, item := range items {
		if item.IsCurrent {
			continue
		}
		out = append(out, item)
	}
	return out
}

func parseListArgs(args []string) (listConfig, error) {
	var cfg listConfig
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--json":
			cfg.json = true
		case arg == "--verbose":
			cfg.verbose = true
		case arg == "--help" || arg == "-h" || arg == "help":
			cfg.help = true
		case strings.HasPrefix(arg, "-"):
			return cfg, appError{Code: "input.invalid_argument", Message: fmt.Sprintf("unknown option: %s", arg), ExitCode: 2}
		default:
			return cfg, appError{Code: "input.invalid_argument", Message: fmt.Sprintf("unexpected extra arguments: %s", strings.Join(args[i:], " ")), ExitCode: 2}
		}
	}
	if cfg.help && (cfg.json || cfg.verbose) {
		return cfg, appError{Code: "input.invalid_argument", Message: "list help cannot be combined with other arguments", ExitCode: 2}
	}
	return cfg, nil
}

func parseNewPathArgs(args []string) (newPathConfig, error) {
	var cfg newPathConfig
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--json":
			cfg.json = true
		case arg == "--label":
			if i+1 >= len(args) {
				return cfg, appError{Code: "input.invalid_argument", Message: "missing value for --label", ExitCode: 2}
			}
			i++
			cfg.label = strings.TrimSpace(args[i])
			if cfg.label == "" {
				return cfg, appError{Code: "input.invalid_argument", Message: "label cannot be empty", ExitCode: 2}
			}
		case strings.HasPrefix(arg, "--label="):
			cfg.label = strings.TrimSpace(strings.TrimPrefix(arg, "--label="))
			if cfg.label == "" {
				return cfg, appError{Code: "input.invalid_argument", Message: "label cannot be empty", ExitCode: 2}
			}
		case arg == "--message" || arg == "-m":
			if i+1 >= len(args) {
				return cfg, appError{Code: "input.invalid_argument", Message: "missing value for --message", ExitCode: 2}
			}
			i++
			cfg.message = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--message="):
			cfg.message = strings.TrimSpace(strings.TrimPrefix(arg, "--message="))
		case arg == "--ttl":
			if i+1 >= len(args) {
				return cfg, appError{Code: "input.invalid_argument", Message: "missing value for --ttl", ExitCode: 2}
			}
			i++
			spec, err := state.ParseHumanDuration(args[i])
			if err != nil {
				return cfg, appError{Code: "input.invalid_duration", Message: err.Error(), ExitCode: 2}
			}
			cfg.ttl = spec.String()
		case strings.HasPrefix(arg, "--ttl="):
			spec, err := state.ParseHumanDuration(strings.TrimPrefix(arg, "--ttl="))
			if err != nil {
				return cfg, appError{Code: "input.invalid_duration", Message: err.Error(), ExitCode: 2}
			}
			cfg.ttl = spec.String()
		case arg == "--no-sync":
			cfg.noSync = true
		case arg == "--sync-dry-run":
			cfg.syncDryRun = true
		case strings.HasPrefix(arg, "-"):
			return cfg, appError{Code: "input.invalid_argument", Message: fmt.Sprintf("unknown option: %s", arg), ExitCode: 2}
		default:
			if cfg.name != "" {
				return cfg, appError{Code: "input.invalid_argument", Message: fmt.Sprintf("unexpected extra arguments: %s", strings.Join(args[i:], " ")), ExitCode: 2}
			}
			cfg.name = arg
		}
	}
	if cfg.name == "" {
		return cfg, appError{Code: "input.invalid_argument", Message: "missing worktree name", ExitCode: 2}
	}
	return cfg, nil
}

func parseGCArgs(args []string) (gcConfig, error) {
	var cfg gcConfig
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--ttl-expired":
			cfg.ttlExpired = true
		case arg == "--merged":
			cfg.merged = true
		case arg == "--dry-run":
			cfg.dryRun = true
		case arg == "--force":
			cfg.force = true
		case arg == "--json":
			cfg.json = true
		case arg == "--idle":
			if i+1 >= len(args) {
				return cfg, appError{Code: "input.invalid_argument", Message: "missing value for --idle", ExitCode: 2}
			}
			i++
			spec, err := state.ParseHumanDuration(args[i])
			if err != nil {
				return cfg, appError{Code: "input.invalid_duration", Message: err.Error(), ExitCode: 2}
			}
			cfg.idle = spec
			cfg.idleSet = true
		case strings.HasPrefix(arg, "--idle="):
			spec, err := state.ParseHumanDuration(strings.TrimPrefix(arg, "--idle="))
			if err != nil {
				return cfg, appError{Code: "input.invalid_duration", Message: err.Error(), ExitCode: 2}
			}
			cfg.idle = spec
			cfg.idleSet = true
		case arg == "--base":
			if i+1 >= len(args) {
				return cfg, appError{Code: "input.invalid_argument", Message: "missing value for --base", ExitCode: 2}
			}
			i++
			cfg.base = args[i]
		case strings.HasPrefix(arg, "--base="):
			cfg.base = strings.TrimPrefix(arg, "--base=")
		case strings.HasPrefix(arg, "-"):
			return cfg, appError{Code: "input.invalid_argument", Message: fmt.Sprintf("unknown option: %s", arg), ExitCode: 2}
		default:
			return cfg, appError{Code: "input.invalid_argument", Message: fmt.Sprintf("unexpected extra arguments: %s", strings.Join(args[i:], " ")), ExitCode: 2}
		}
	}

	if !cfg.ttlExpired && !cfg.idleSet && !cfg.merged {
		return cfg, appError{Code: "input.missing_selector", Message: "at least one gc rule is required", ExitCode: 2}
	}
	return cfg, nil
}

func ttlExpired(meta state.WorktreeMetadata, now time.Time) bool {
	if meta.CreatedAt == 0 || meta.TTL == "" {
		return false
	}
	spec, err := state.ParseHumanDuration(meta.TTL)
	if err != nil {
		return false
	}
	return !time.Unix(0, meta.CreatedAt).Add(spec.Value).After(now)
}

func idleExpired(meta state.WorktreeMetadata, spec state.DurationSpec, now time.Time) bool {
	if meta.LastUsedAt == 0 {
		return false
	}
	return now.Sub(time.Unix(0, meta.LastUsedAt)) >= spec.Value
}

func decorateListEntries(items []worktree.Worktree, metadata map[string]state.WorktreeMetadata) []listEntry {
	entries := make([]listEntry, 0, len(items))
	for _, item := range items {
		meta := metadata[item.Path]
		if meta.CreatedAt == 0 {
			meta.CreatedAt = item.CreatedAt
		}
		if meta.LastUsedAt == 0 {
			meta.LastUsedAt = item.LastUsedAt
		}
		entries = append(entries, listEntry{item: item, meta: meta})
	}
	return entries
}

func writeGCHuman(out io.Writer, results []GCItem) {
	for _, item := range results {
		switch item.Action {
		case "removed":
			fmt.Fprintf(out, "removed %s\n", item.Path)
		case "skipped":
			fmt.Fprintf(out, "skipped %s (%s)\n", item.Path, item.Reason)
		}
	}
}

func selectRemovalCandidateNonInteractive(allItems []worktree.Worktree, candidates []removalCandidate, target string) (removalCandidate, error) {
	if target == "" {
		if len(candidates) == 1 {
			return candidates[0], nil
		}
		return removalCandidate{}, appError{
			Code:     "worktree.ambiguous_match",
			Message:  "must specify a target when multiple removable worktrees exist",
			ExitCode: 2,
		}
	}

	if selected, err := worktree.Match(allItems, target); err == nil && selected.IsCurrent {
		return removalCandidate{}, appError{
			Code:     "worktree.remove_current",
			Message:  "cannot remove the active worktree",
			ExitCode: 1,
		}
	}

	return matchRemovalCandidate(candidates, target)
}

func matchRemovalCandidate(candidates []removalCandidate, target string) (removalCandidate, error) {
	items := make([]worktree.Worktree, 0, len(candidates))
	byPath := make(map[string]removalCandidate, len(candidates))
	for _, candidate := range candidates {
		items = append(items, candidate.item)
		byPath[candidate.item.Path] = candidate
	}
	selected, err := worktree.Match(items, target)
	if err != nil {
		return removalCandidate{}, err
	}
	return byPath[selected.Path], nil
}

func runRemoveCleanup(ctx context.Context, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
	repoKey, items, _, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeCommandError("rm", out, errOut, false, err)
	}
	warnStateIssue(errOut, warn)

	baseBranch, err := deps.DefaultBranch(ctx)
	if err != nil {
		return writeCommandError("rm", out, errOut, false, err)
	}
	displayRoot := mainWorktreeRootFromRepoKey(repoKey)

	type cleanupCandidate struct {
		removalCandidate
		commitSubject string
	}

	var safe []cleanupCandidate
	reviewCount := 0
	blockedCount := 0
	for _, item := range filterNonCurrent(items) {
		preview, err := deps.PreviewRemoval(ctx, item, baseBranch)
		if err != nil {
			return writeCommandError("rm", out, errOut, false, err)
		}
		candidate := newRemovalCandidate(ctx, deps, item, preview, baseBranch)
		switch {
		case preview.Dirty:
			blockedCount++
		case candidate.idleTemporary:
			safe = append(safe, cleanupCandidate{removalCandidate: candidate})
		case item.BranchRef != "" && item.BranchLabel != baseBranch && preview.BranchMerged:
			subject, _ := deps.LastCommitSubject(ctx, item.Path)
			safe = append(safe, cleanupCandidate{removalCandidate: candidate, commitSubject: subject})
		default:
			reviewCount++
		}
	}

	if len(safe) == 0 {
		fmt.Fprintln(errOut, ui.Yellow("No worktrees are clearly safe to delete."))
		if reviewCount > 0 {
			fmt.Fprintln(errOut, ui.Yellow(cleanupNeedsReview(reviewCount)+"."))
		}
		if blockedCount > 0 {
			fmt.Fprintln(errOut, ui.Red(cleanupBlocked(blockedCount)+"."))
		}
		return 0
	}

	fmt.Fprintln(errOut, ui.Green(cleanupSafeToDelete(len(safe))+"."))
	fmt.Fprintln(errOut, ui.Dim("Safe because: clean files and no work that needs preserving."))
	fmt.Fprintln(errOut)
	for i, candidate := range safe {
		fmt.Fprintf(errOut, "%d. %s", i+1, ui.Bold(removalCandidateLabel(candidate.displayItem)))
		if candidate.idleTemporary {
			fmt.Fprintf(errOut, "  %s", ui.Dim(listDisplayPath(candidate.displayItem.Path, displayRoot)))
		}
		fmt.Fprintln(errOut)
		if candidate.commitSubject != "" {
			fmt.Fprintln(errOut, ui.Dim(fmt.Sprintf("   last commit: %s", candidate.commitSubject)))
		}
		if i != len(safe)-1 {
			fmt.Fprintln(errOut)
		}
	}
	if reviewCount > 0 || blockedCount > 0 {
		fmt.Fprintln(errOut)
	}
	if reviewCount > 0 {
		fmt.Fprintln(errOut, ui.Yellow(cleanupNeedsReview(reviewCount)+"."))
		fmt.Fprintln(errOut, ui.Dim("Run ww rm to remove one manually."))
	}
	if blockedCount > 0 {
		fmt.Fprintln(errOut, ui.Red(cleanupBlocked(blockedCount)+"."))
	}
	fmt.Fprintln(errOut)

	reader := bufio.NewReader(in)
	confirmed, err := confirmPrompt(reader, errOut, ui.Bold(cleanupDeletePrompt(len(safe))))
	if err != nil {
		return writeSelectionError(errOut, err)
	}
	if !confirmed {
		return 130
	}

	for _, candidate := range safe {
		result, err := deps.RemoveWorktree(ctx, candidate.displayItem, git.RemoveOptions{BaseBranch: baseBranch})
		if err != nil {
			return writeCommandError("rm", out, errOut, false, err)
		}
		writeRemoveHuman(out, result)
	}
	return 0
}

func cleanupSafeToDelete(n int) string {
	if n == 1 {
		return "1 worktree is safe to delete"
	}
	return fmt.Sprintf("%d worktrees are safe to delete", n)
}

func cleanupNeedsReview(n int) string {
	if n == 1 {
		return "1 worktree needs review"
	}
	return fmt.Sprintf("%d worktrees need review", n)
}

func cleanupBlocked(n int) string {
	if n == 1 {
		return "1 worktree is blocked"
	}
	return fmt.Sprintf("%d worktrees are blocked", n)
}

func cleanupDeletePrompt(n int) string {
	if n == 1 {
		return "Delete this worktree? [y/N] "
	}
	return fmt.Sprintf("Delete these %d? [y/N] ", n)
}

func renderRemovalCandidates(w io.Writer, candidates []removalCandidate, displayRoot string) {
	fmt.Fprintln(w, "Remove which worktree?")
	fmt.Fprintln(w)
	for i, c := range candidates {
		label := removalCandidateLabel(c.displayItem)
		status, reason := removalCandidateJudgement(c)
		if c.idleTemporary {
			label += "  " + ui.Dim(listDisplayPath(c.displayItem.Path, displayRoot))
		}
		fmt.Fprintf(w, "  %d  %s  %s  %s\n", i+1, status, label, reason)
	}
}

func removalCandidateJudgement(candidate removalCandidate) (string, string) {
	preview := candidate.preview
	switch {
	case preview.Dirty:
		return ui.Yellow("! review"), ui.Dim("local changes")
	case candidate.idleTemporary:
		return ui.Green("✓ safe  "), ui.Dim("clean + idle")
	case candidate.item.BranchRef == "":
		return ui.Yellow("! review"), ui.Dim("no branch")
	case preview.DeleteBranch:
		return ui.Green("✓ safe  "), ui.Dim("clean + merged")
	case !preview.BranchMerged:
		return ui.Yellow("! review"), ui.Dim("not merged")
	default:
		return ui.Yellow("! review"), ui.Dim("branch kept")
	}
}

func removalCandidateLabel(item worktree.Worktree) string {
	label := strings.TrimSpace(item.BranchLabel)
	if label != "" {
		if item.StatusLabel != "" {
			return item.StatusLabel + " " + label
		}
		return label
	}
	return filepath.Base(item.Path)
}

func readChoice(reader *bufio.Reader, errOut io.Writer, prompt string, max int, defaultIndex int) (int, error) {
	for {
		fmt.Fprint(errOut, prompt)
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if defaultIndex > 0 {
				return defaultIndex, nil
			}
			if errors.Is(err, io.EOF) {
				return 0, io.EOF
			}
			fmt.Fprintln(errOut, "empty selection")
			continue
		}

		index, convErr := strconv.Atoi(trimmed)
		if convErr != nil || index <= 0 || index > max {
			fmt.Fprintf(errOut, "invalid selection: %q\n", trimmed)
			if errors.Is(err, io.EOF) {
				return 0, io.EOF
			}
			continue
		}
		return index, nil
	}
}

func confirmPrompt(reader *bufio.Reader, errOut io.Writer, prompt string) (bool, error) {
	fmt.Fprint(errOut, prompt)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func writeRemoveHuman(out io.Writer, result git.RemoveResult) {
	label := result.Branch
	if label == "" {
		label = filepath.Base(result.WorktreePath)
	}
	switch {
	case result.DeletedBranch:
		fmt.Fprintf(out, "Removed %s (branch deleted)\n", label)
	case result.Branch != "" && result.KeptBranchReason != "":
		fmt.Fprintf(out, "Removed %s (branch kept, %s)\n", label, result.KeptBranchReason)
	default:
		fmt.Fprintf(out, "Removed %s\n", label)
	}
}

// protocolVersion is the wire-format version reported in every JSON envelope.
// See docs/protocol.md for the contract.
const protocolVersion = "1.1"

// binaryVersion is the build version of the ww-helper binary. Defaults to
// "dev"; release builds inject the tag via:
//
//	go build -ldflags "-X 'ww/internal/app.binaryVersion=v0.4.0'" ./cmd/ww-helper
var binaryVersion = "dev"

// buildCommit and buildDirty are injected by local/release builds when Git
// metadata is available. They are intentionally separate from binaryVersion
// so unreleased local builds can still say "dev+<commit>".
var buildCommit = ""
var buildDirty = "false"

// nanosToMillis converts an internal unix-nanosecond timestamp to the
// unix-millisecond form documented in protocol §4.1. A zero input (no value)
// is preserved as zero.
func nanosToMillis(nanos int64) int64 {
	if nanos == 0 {
		return 0
	}
	return nanos / 1_000_000
}

func writeJSONSuccess(out io.Writer, command string, data any, warnings ...Warning) int {
	ws := warnings
	if ws == nil {
		ws = []Warning{}
	}
	payload := map[string]any{
		"protocol": protocolVersion,
		"ok":       true,
		"command":  command,
		"data":     data,
		"warnings": ws,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return 1
	}
	fmt.Fprintln(out, string(encoded))
	return 0
}

func writeJSONError(out io.Writer, command string, err appError) int {
	payload := map[string]any{
		"protocol": protocolVersion,
		"ok":       false,
		"command":  command,
		"error": map[string]any{
			"code":    err.Code,
			"message": err.Message,
			"context": map[string]any{},
		},
	}
	encoded, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return 1
	}
	fmt.Fprintln(out, string(encoded))
	return err.ExitCode
}

func writeCommandError(command string, out io.Writer, errOut io.Writer, jsonMode bool, err error) int {
	appErr := classifyError(err)
	if jsonMode {
		return writeJSONError(out, command, appErr)
	}
	if appErr.Message != "" {
		fmt.Fprintln(errOut, appErr.Message)
	}
	return appErr.ExitCode
}

func classifyError(err error) appError {
	var appErr appError
	if errors.As(err, &appErr) {
		return appErr
	}

	switch {
	case errors.Is(err, git.ErrNotGitRepository):
		return appError{Code: "git.repo_missing", Message: "not a git repository", ExitCode: 3}
	case errors.Is(err, ui.ErrSelectionCanceled):
		return appError{Code: "selector.cancelled", Message: "selection canceled", ExitCode: 130}
	case errors.Is(err, ui.ErrFzfNotInstalled):
		return appError{Code: "selector.fzf_not_installed", Message: "fzf is not installed", ExitCode: 3}
	}

	message := err.Error()
	switch {
	case strings.HasPrefix(message, "ambiguous worktree match"):
		return appError{Code: "worktree.ambiguous_match", Message: message, ExitCode: 2}
	case strings.HasPrefix(message, "no worktree matches"):
		return appError{Code: "worktree.not_found", Message: message, ExitCode: 2}
	case strings.Contains(message, "uncommitted changes"):
		return appError{Code: "worktree.dirty", Message: message, ExitCode: 1}
	default:
		return appError{Code: "git.command_failed", Message: message, ExitCode: 1}
	}
}

func selectInteractiveWorktree(ctx context.Context, in io.Reader, errOut io.Writer, items []worktree.Worktree, deps Deps, forceFzf bool) (worktree.Worktree, error) {
	if forceFzf {
		return deps.SelectWorktreeWithFzf(ctx, items)
	}

	selected, err := deps.SelectWorktreeWithFzf(ctx, items)
	switch {
	case err == nil:
		return selected, nil
	case errors.Is(err, ui.ErrFzfNotInstalled):
		return deps.SelectWorktreeWithTUI(in, errOut, items)
	default:
		return worktree.Worktree{}, err
	}
}

func selectByIndex(items []worktree.Worktree, index int) (worktree.Worktree, bool) {
	for i := range items {
		if items[i].Index == index {
			return items[i], true
		}
	}
	return worktree.Worktree{}, false
}

func writeWorktreeError(errOut io.Writer, err error) int {
	if errors.Is(err, git.ErrNotGitRepository) {
		fmt.Fprintln(errOut, "not a git repository")
		return 3
	}
	fmt.Fprintln(errOut, err)
	return 1
}

func writeSelectionError(errOut io.Writer, err error) int {
	switch {
	case errors.Is(err, ui.ErrFzfNotInstalled):
		fmt.Fprintln(errOut, "fzf is not installed")
		return 3
	case errors.Is(err, ui.ErrSelectionCanceled):
		return 130
	case errors.Is(err, io.EOF):
		return 1
	default:
		fmt.Fprintln(errOut, err)
		return 1
	}
}

func touchWorktreeStateBestEffort(ctx context.Context, deps Deps, repoKey, path string) error {
	if err := deps.TouchWorktreeState(ctx, repoKey, path); err != nil {
		return fmt.Errorf("state update skipped: %w", err)
	}
	return nil
}

func recordWorktreeStateBestEffort(ctx context.Context, deps Deps, repoKey, path string, meta state.WorktreeMetadata) error {
	if err := deps.RecordWorktreeState(ctx, repoKey, path, meta); err != nil {
		return fmt.Errorf("state update skipped: %w", err)
	}
	return nil
}

func warnStateIssue(errOut io.Writer, err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(errOut, err)
}

func resolveInitPaths() (string, string, error) {
	execPath, err := executablePath()
	if err != nil {
		return "", "", fmt.Errorf("resolve ww-helper path: %w", err)
	}

	resolvedExecPath, err := evalSymlinks(execPath)
	if err != nil {
		resolvedExecPath = execPath
	}
	resolvedExecPath = filepath.Clean(resolvedExecPath)

	candidates := []string{
		filepath.Join(filepath.Dir(resolvedExecPath), "ww.sh"),
		filepath.Join(filepath.Dir(resolvedExecPath), "..", "libexec", "ww.sh"),
	}
	for _, candidate := range candidates {
		info, statErr := os.Stat(candidate)
		if statErr == nil && !info.IsDir() {
			return resolvedExecPath, filepath.Clean(candidate), nil
		}
	}

	return "", "", fmt.Errorf("resolve ww.sh path relative to %s: file not found", resolvedExecPath)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func printHelperHelp(out io.Writer) {
	fmt.Fprintln(out, "Usage: ww-helper [switch-path|list|new-path|init|gc|rm|version|help|--help]")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "switch-path prints the selected git worktree path.")
	fmt.Fprintln(out, "Interactive switch uses fzf when available, otherwise the built-in selector.")
	fmt.Fprintln(out, "list prints the current worktree table.")
	fmt.Fprintln(out, "new-path creates a worktree and prints its path.")
	fmt.Fprintln(out, "init prints shell code that activates ww for zsh or bash.")
	fmt.Fprintln(out, "gc evaluates explicit cleanup rules and prints matched worktrees.")
	fmt.Fprintln(out, "rm removes one worktree.")
	fmt.Fprintln(out, "rm --cleanup removes clearly safe worktrees.")
	fmt.Fprintln(out, "[IDLE] temporary = clean detached worktree with no commits beyond the status base.")
	fmt.Fprintln(out, "version prints the binary and protocol version. Pass --json for the envelope form.")
	fmt.Fprintln(out, "help prints this command summary.")
}

func printListHelp(out io.Writer) {
	fmt.Fprintln(out, "Usage: ww list [--verbose] [--json]")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Shows worktrees without switching.")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Status:")
	fmt.Fprintln(out, "  [CURRENT]  current shell worktree")
	fmt.Fprintln(out, "  [MERGED]   branch already merged into the status base")
	fmt.Fprintln(out, "  [IDLE]     temporary detached worktree with no local work")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Branch:")
	fmt.Fprintln(out, "  temporary  detached worktree with clean files and no commits beyond status base")
	fmt.Fprintln(out, "  unbranched detached worktree with commits; review before removing")
}

func printRemoveHelp(out io.Writer) {
	fmt.Fprintln(out, "Usage: ww rm [--force] [target]")
	fmt.Fprintln(out, "       ww rm --cleanup")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Without a target, opens a selector.")
	fmt.Fprintln(out, "--cleanup removes only clearly safe worktrees.")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Safe:")
	fmt.Fprintln(out, "  - clean merged branch worktrees")
	fmt.Fprintln(out, "  - [IDLE] temporary worktrees")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Review:")
	fmt.Fprintln(out, "  - dirty worktrees")
	fmt.Fprintln(out, "  - unbranched detached worktrees with commits")
}
