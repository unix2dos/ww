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

	"ww/internal/git"
	"ww/internal/state"
	"ww/internal/tasknote"
	"ww/internal/ui"
	"ww/internal/worktree"
)

type Deps interface {
	CurrentRepoKey(ctx context.Context) (string, error)
	ListWorktrees(ctx context.Context) (string, []worktree.Worktree, error)
	SelectWorktreeWithFzf(ctx context.Context, items []worktree.Worktree) (worktree.Worktree, error)
	SelectWorktreeWithTUI(in io.Reader, out io.Writer, items []worktree.Worktree) (worktree.Worktree, error)
	CreateWorktree(ctx context.Context, name string) (string, error)
	LoadWorktreeState(ctx context.Context, repoKey string) (map[string]int64, error)
	LoadWorktreeMetadata(ctx context.Context, repoKey string) (map[string]state.WorktreeMetadata, error)
	TouchWorktreeState(ctx context.Context, repoKey, path string) error
	RecordWorktreeState(ctx context.Context, repoKey, path string, meta state.WorktreeMetadata) error
	WorktreeGitPath(ctx context.Context, worktreePath string, rel string) (string, error)
	DefaultBranch(ctx context.Context) (string, error)
	AnnotateExtendedStatus(ctx context.Context, items []worktree.Worktree, baseBranch string) error
	PreviewRemoval(ctx context.Context, item worktree.Worktree, baseBranch string) (git.RemovalPreview, error)
	RemoveWorktree(ctx context.Context, item worktree.Worktree, opts git.RemoveOptions) (git.RemoveResult, error)
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

func (d RealDeps) ListWorktrees(ctx context.Context) (string, []worktree.Worktree, error) {
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

func (d RealDeps) AnnotateExtendedStatus(ctx context.Context, items []worktree.Worktree, baseBranch string) error {
	return git.AnnotateExtendedStatus(ctx, git.ExecRunner{}, items, baseBranch)
}

func (d RealDeps) PreviewRemoval(ctx context.Context, item worktree.Worktree, baseBranch string) (git.RemovalPreview, error) {
	return git.PreviewRemoval(ctx, git.ExecRunner{}, item, baseBranch)
}

func (d RealDeps) RemoveWorktree(ctx context.Context, item worktree.Worktree, opts git.RemoveOptions) (git.RemoveResult, error) {
	return git.RemoveWorktree(ctx, git.ExecRunner{}, item, opts)
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
	default:
		return runSwitchPath(ctx, args, in, out, errOut, deps)
	}
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

// annotateExtendedStatusBestEffort calls AnnotateExtendedStatus if DefaultBranch
// can be resolved. Errors are swallowed — if git commands fail, the list shows
// with basic info only.
func annotateExtendedStatusBestEffort(ctx context.Context, deps Deps, items []worktree.Worktree) {
	baseBranch, err := deps.DefaultBranch(ctx)
	if err != nil {
		return
	}
	_ = deps.AnnotateExtendedStatus(ctx, items, baseBranch)
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

		annotateExtendedStatusBestEffort(ctx, deps, items)

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

		annotateExtendedStatusBestEffort(ctx, deps, items)

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
	filters []listFilter
}

type listEntry struct {
	item worktree.Worktree
	meta state.WorktreeMetadata
}

type listFilter struct {
	kind     string
	value    string
	duration state.DurationSpec
}

func runList(ctx context.Context, args []string, out io.Writer, errOut io.Writer, deps Deps) int {
	cfg, err := parseListArgs(args)
	if err != nil {
		return writeCommandError("list", out, errOut, cfg.json, err)
	}

	_, items, metadata, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeCommandError("list", out, errOut, cfg.json, err)
	}
	if !cfg.json {
		warnStateIssue(errOut, warn)
	}

	annotateExtendedStatusBestEffort(ctx, deps, items)

	entries, err := filterListEntries(decorateListEntries(items, metadata), cfg.filters, time.Now())
	if err != nil {
		return writeCommandError("list", out, errOut, cfg.json, err)
	}
	if len(entries) == 0 {
		if cfg.json {
			return writeJSONSuccess(out, "list", []any{})
		}
		return writeCommandError("list", out, errOut, cfg.json, appError{
			Code:     "WORKTREE_NOT_FOUND",
			Message:  "no worktrees available",
			ExitCode: 1,
		})
	}

	if cfg.json {
		payload := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			item := entry.item
			payload = append(payload, map[string]any{
				"path":         item.Path,
				"branch":       item.BranchLabel,
				"dirty":        item.IsDirty,
				"active":       item.IsCurrent,
				"created_at":   item.CreatedAt,
				"last_used_at": entry.meta.LastUsedAt,
				"label":        entry.meta.Label,
				"ttl":          entry.meta.TTL,
				"merged":       item.IsMerged,
				"ahead":        item.Ahead,
				"behind":       item.Behind,
				"staged":       item.Staged,
				"unstaged":     item.Unstaged,
				"untracked":    item.Untracked,
			})
		}
		return writeJSONSuccess(out, "list", payload)
	}

	tableEntries := make([]ui.ListTableEntry, 0, len(entries))
	for _, entry := range entries {
		detail := listVerboseDetail(ctx, deps, entry, cfg.verbose)
		tableEntries = append(tableEntries, ui.ListTableEntry{
			Worktree: entry.item,
			Detail:   detail,
		})
	}
	fmt.Fprintln(out, ui.FormatListTable(tableEntries))
	return 0
}

func listVerboseDetail(ctx context.Context, deps Deps, entry listEntry, verbose bool) string {
	if !verbose {
		return ""
	}

	parts := make([]string, 0, 4)
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
	json    bool
	name    string
	label   string
	ttl     string
	message string
}

func runNewPath(ctx context.Context, args []string, out io.Writer, errOut io.Writer, deps Deps) int {
	cfg, err := parseNewPathArgs(args)
	if err != nil {
		return writeCommandError("new-path", out, errOut, cfg.json, err)
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

	if cfg.json {
		if err := recordWorktreeStateBestEffort(ctx, deps, repoKey, path, meta); err != nil {
			return writeCommandError("new-path", out, errOut, cfg.json, err)
		}
		if err := createTaskNoteIfLabeled(ctx, deps, path, cfg.name, cfg.label, cfg.message, createdAt); err != nil {
			return writeCommandError("new-path", out, errOut, cfg.json, err)
		}
		if err := touchWorktreeStateBestEffort(ctx, deps, repoKey, path); err != nil {
			return writeCommandError("new-path", out, errOut, cfg.json, err)
		}
		return writeJSONSuccess(out, "new-path", map[string]any{
			"worktree_path": path,
			"branch":        cfg.name,
		})
	}

	fmt.Fprintln(out, path)
	warnStateIssue(errOut, recordWorktreeStateBestEffort(ctx, deps, repoKey, path, meta))
	warnStateIssue(errOut, createTaskNoteIfLabeled(ctx, deps, path, cfg.name, cfg.label, cfg.message, createdAt))
	warnStateIssue(errOut, touchWorktreeStateBestEffort(ctx, deps, repoKey, path))
	return 0
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

	_, items, metadata, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeCommandError("gc", out, errOut, cfg.json, err)
	}
	if !cfg.json {
		warnStateIssue(errOut, warn)
	}

	now := time.Now()
	entries := decorateListEntries(items, metadata)
	baseBranch := cfg.base
	candidates := make([]gcCandidate, 0, len(entries))
	for _, entry := range entries {
		candidate := gcCandidate{entry: entry}
		if cfg.ttlExpired && ttlExpired(entry.meta, now) {
			candidate.matchedRules = append(candidate.matchedRules, "ttl_expired")
		}
		if cfg.idleSet && idleExpired(entry.meta, cfg.idle, now) {
			candidate.matchedRules = append(candidate.matchedRules, "idle")
		}
		if cfg.merged && entry.item.BranchRef != "" {
			if baseBranch == "" {
				baseBranch, err = deps.DefaultBranch(ctx)
				if err != nil {
					return writeCommandError("gc", out, errOut, cfg.json, err)
				}
			}
			preview, previewErr := deps.PreviewRemoval(ctx, entry.item, baseBranch)
			if previewErr != nil {
				return writeCommandError("gc", out, errOut, cfg.json, previewErr)
			}
			candidate.preview = preview
			candidate.hasPreview = true
			if preview.BranchMerged {
				candidate.matchedRules = append(candidate.matchedRules, "merged")
			}
		}
		if len(candidate.matchedRules) > 0 {
			candidates = append(candidates, candidate)
		}
	}

	if cfg.dryRun {
		return writeJSONSuccess(out, "gc", gcJSONPayload(candidates, nil))
	}

	results := make([]gcResultItem, 0, len(candidates))
	for _, candidate := range candidates {
		item := candidate.entry.item
		if item.IsCurrent {
			results = append(results, gcResultItem{
				Path:         item.Path,
				Branch:       item.BranchLabel,
				MatchedRules: candidate.matchedRules,
				Action:       "skipped",
				Reason:       "active",
			})
			continue
		}

		preview := candidate.preview
		if !candidate.hasPreview && item.BranchRef != "" {
			if baseBranch == "" {
				baseBranch, err = deps.DefaultBranch(ctx)
				if err != nil {
					return writeCommandError("gc", out, errOut, cfg.json, err)
				}
			}
			preview, err = deps.PreviewRemoval(ctx, item, baseBranch)
			if err != nil {
				return writeCommandError("gc", out, errOut, cfg.json, err)
			}
		}
		if preview.Dirty && !cfg.force {
			results = append(results, gcResultItem{
				Path:         item.Path,
				Branch:       item.BranchLabel,
				MatchedRules: candidate.matchedRules,
				Action:       "skipped",
				Reason:       "dirty",
			})
			continue
		}

		removeResult, removeErr := deps.RemoveWorktree(ctx, item, git.RemoveOptions{
			BaseBranch: baseBranch,
			Force:      cfg.force,
		})
		if removeErr != nil {
			return writeCommandError("gc", out, errOut, cfg.json, removeErr)
		}
		results = append(results, gcResultItem{
			Path:         removeResult.WorktreePath,
			Branch:       removeResult.Branch,
			MatchedRules: candidate.matchedRules,
			Action:       "removed",
		})
	}

	if cfg.json {
		return writeJSONSuccess(out, "gc", gcJSONPayload(candidates, results))
	}
	writeGCHuman(out, results)
	return 0
}

type removeConfig struct {
	force  bool
	json   bool
	target string
}

type removalCandidate struct {
	item    worktree.Worktree
	preview git.RemovalPreview
}

func runRemove(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
	cfg, err := parseRemoveArgs(args)
	if err != nil {
		return writeCommandError("rm", out, errOut, cfg.json, err)
	}

	_, items, _, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeCommandError("rm", out, errOut, cfg.json, err)
	}
	if !cfg.json {
		warnStateIssue(errOut, warn)
	}

	// Check if target is the current worktree before filtering
	if cfg.target != "" {
		if selected, matchErr := worktree.Match(items, cfg.target); matchErr == nil && selected.IsCurrent {
			if cfg.json {
				return writeCommandError("rm", out, errOut, true, appError{
					Code:     "REMOVE_CURRENT",
					Message:  "Cannot remove the current worktree. Switch first: ww go <name>",
					ExitCode: 1,
				})
			}
			fmt.Fprintln(errOut, "Cannot remove the current worktree. Switch first: ww go <name>")
			return 1
		}
	}

	candidates := filterNonCurrent(items)
	if len(candidates) == 0 {
		return writeCommandError("rm", out, errOut, cfg.json, appError{
			Code:     "WORKTREE_NOT_FOUND",
			Message:  "no removable worktrees available",
			ExitCode: 1,
		})
	}

	baseBranch, err := deps.DefaultBranch(ctx)
	if err != nil {
		return writeCommandError("rm", out, errOut, cfg.json, err)
	}

	previewed := make([]removalCandidate, 0, len(candidates))
	for _, item := range candidates {
		preview, err := deps.PreviewRemoval(ctx, item, baseBranch)
		if err != nil {
			return writeCommandError("rm", out, errOut, cfg.json, err)
		}
		previewed = append(previewed, removalCandidate{item: item, preview: preview})
	}

	// Select target
	selected := removalCandidate{}
	if cfg.json {
		selected, err = selectRemovalCandidateNonInteractive(items, previewed, cfg.target)
		if err != nil {
			return writeCommandError("rm", out, errOut, true, err)
		}
		if selected.preview.Dirty && !cfg.force {
			return writeCommandError("rm", out, errOut, true, appError{
				Code:     "WORKTREE_DIRTY",
				Message:  "worktree has uncommitted changes; rerun with --force",
				ExitCode: 1,
			})
		}
	} else {
		reader := bufio.NewReader(in)

		if cfg.target != "" {
			selected, err = matchRemovalCandidate(previewed, cfg.target)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return 2
			}
		} else if len(previewed) == 1 {
			selected = previewed[0]
		} else {
			renderRemovalCandidates(errOut, previewed)
			index, err := readChoice(reader, errOut, "\n> ", len(previewed), 0)
			if err != nil {
				return writeSelectionError(errOut, err)
			}
			selected = previewed[index-1]
		}

		label := removalCandidateLabel(selected.item)
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
	}

	result, err := deps.RemoveWorktree(ctx, selected.item, git.RemoveOptions{
		BaseBranch: baseBranch,
		Force:      cfg.force,
	})
	if err != nil {
		return writeCommandError("rm", out, errOut, cfg.json, err)
	}

	if cfg.json {
		return writeJSONSuccess(out, "rm", removeJSONPayload(result))
	}
	writeRemoveHuman(out, result)
	return 0
}

func orderedWorktrees(ctx context.Context, deps Deps) (string, []worktree.Worktree, map[string]state.WorktreeMetadata, error, error) {
	repoKey, items, err := deps.ListWorktrees(ctx)
	if err != nil {
		return "", nil, nil, nil, err
	}
	metadata, err := deps.LoadWorktreeMetadata(ctx, repoKey)
	if err != nil {
		normalized := worktree.Normalize(items)
		return repoKey, normalized, map[string]state.WorktreeMetadata{}, fmt.Errorf("state load unavailable: %w", err), nil
	}
	for i := range items {
		meta := metadata[items[i].Path]
		items[i].LastUsedAt = meta.LastUsedAt
		if meta.CreatedAt != 0 {
			items[i].CreatedAt = meta.CreatedAt
		}
	}
	return repoKey, worktree.Normalize(items), metadata, nil, nil
}

func parseRemoveArgs(args []string) (removeConfig, error) {
	var cfg removeConfig
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--force":
			cfg.force = true
		case arg == "--json":
			cfg.json = true
		case strings.HasPrefix(arg, "-"):
			return cfg, appError{Code: "INVALID_ARGUMENTS", Message: fmt.Sprintf("unknown option: %s", arg), ExitCode: 2}
		default:
			if cfg.target != "" {
				return cfg, appError{Code: "INVALID_ARGUMENTS", Message: fmt.Sprintf("unexpected extra arguments: %s", strings.Join(args[i:], " ")), ExitCode: 2}
			}
			cfg.target = arg
		}
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
		case arg == "--filter":
			if i+1 >= len(args) {
				return cfg, appError{Code: "INVALID_ARGUMENTS", Message: "missing value for --filter", ExitCode: 2}
			}
			i++
			filter, err := parseListFilter(args[i])
			if err != nil {
				return cfg, err
			}
			cfg.filters = append(cfg.filters, filter)
		case strings.HasPrefix(arg, "--filter="):
			filter, err := parseListFilter(strings.TrimPrefix(arg, "--filter="))
			if err != nil {
				return cfg, err
			}
			cfg.filters = append(cfg.filters, filter)
		case strings.HasPrefix(arg, "-"):
			return cfg, appError{Code: "INVALID_ARGUMENTS", Message: fmt.Sprintf("unknown option: %s", arg), ExitCode: 2}
		default:
			return cfg, appError{Code: "INVALID_ARGUMENTS", Message: fmt.Sprintf("unexpected extra arguments: %s", strings.Join(args[i:], " ")), ExitCode: 2}
		}
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
				return cfg, appError{Code: "INVALID_ARGUMENTS", Message: "missing value for --label", ExitCode: 2}
			}
			i++
			cfg.label = strings.TrimSpace(args[i])
			if cfg.label == "" {
				return cfg, appError{Code: "INVALID_ARGUMENTS", Message: "label cannot be empty", ExitCode: 2}
			}
		case strings.HasPrefix(arg, "--label="):
			cfg.label = strings.TrimSpace(strings.TrimPrefix(arg, "--label="))
			if cfg.label == "" {
				return cfg, appError{Code: "INVALID_ARGUMENTS", Message: "label cannot be empty", ExitCode: 2}
			}
		case arg == "--message" || arg == "-m":
			if i+1 >= len(args) {
				return cfg, appError{Code: "INVALID_ARGUMENTS", Message: "missing value for --message", ExitCode: 2}
			}
			i++
			cfg.message = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--message="):
			cfg.message = strings.TrimSpace(strings.TrimPrefix(arg, "--message="))
		case arg == "--ttl":
			if i+1 >= len(args) {
				return cfg, appError{Code: "INVALID_ARGUMENTS", Message: "missing value for --ttl", ExitCode: 2}
			}
			i++
			spec, err := state.ParseHumanDuration(args[i])
			if err != nil {
				return cfg, appError{Code: "INVALID_DURATION", Message: err.Error(), ExitCode: 2}
			}
			cfg.ttl = spec.String()
		case strings.HasPrefix(arg, "--ttl="):
			spec, err := state.ParseHumanDuration(strings.TrimPrefix(arg, "--ttl="))
			if err != nil {
				return cfg, appError{Code: "INVALID_DURATION", Message: err.Error(), ExitCode: 2}
			}
			cfg.ttl = spec.String()
		case strings.HasPrefix(arg, "-"):
			return cfg, appError{Code: "INVALID_ARGUMENTS", Message: fmt.Sprintf("unknown option: %s", arg), ExitCode: 2}
		default:
			if cfg.name != "" {
				return cfg, appError{Code: "INVALID_ARGUMENTS", Message: fmt.Sprintf("unexpected extra arguments: %s", strings.Join(args[i:], " ")), ExitCode: 2}
			}
			cfg.name = arg
		}
	}
	if cfg.name == "" {
		return cfg, appError{Code: "INVALID_ARGUMENTS", Message: "missing worktree name", ExitCode: 2}
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
				return cfg, appError{Code: "INVALID_ARGUMENTS", Message: "missing value for --idle", ExitCode: 2}
			}
			i++
			spec, err := state.ParseHumanDuration(args[i])
			if err != nil {
				return cfg, appError{Code: "INVALID_DURATION", Message: err.Error(), ExitCode: 2}
			}
			cfg.idle = spec
			cfg.idleSet = true
		case strings.HasPrefix(arg, "--idle="):
			spec, err := state.ParseHumanDuration(strings.TrimPrefix(arg, "--idle="))
			if err != nil {
				return cfg, appError{Code: "INVALID_DURATION", Message: err.Error(), ExitCode: 2}
			}
			cfg.idle = spec
			cfg.idleSet = true
		case arg == "--base":
			if i+1 >= len(args) {
				return cfg, appError{Code: "INVALID_ARGUMENTS", Message: "missing value for --base", ExitCode: 2}
			}
			i++
			cfg.base = args[i]
		case strings.HasPrefix(arg, "--base="):
			cfg.base = strings.TrimPrefix(arg, "--base=")
		case strings.HasPrefix(arg, "-"):
			return cfg, appError{Code: "INVALID_ARGUMENTS", Message: fmt.Sprintf("unknown option: %s", arg), ExitCode: 2}
		default:
			return cfg, appError{Code: "INVALID_ARGUMENTS", Message: fmt.Sprintf("unexpected extra arguments: %s", strings.Join(args[i:], " ")), ExitCode: 2}
		}
	}

	if !cfg.ttlExpired && !cfg.idleSet && !cfg.merged {
		return cfg, appError{Code: "GC_RULE_REQUIRED", Message: "at least one gc rule is required", ExitCode: 2}
	}
	return cfg, nil
}

func parseListFilter(expr string) (listFilter, error) {
	switch {
	case expr == "dirty":
		return listFilter{kind: "dirty"}, nil
	case strings.HasPrefix(expr, "label="):
		value := strings.TrimPrefix(expr, "label=")
		if strings.TrimSpace(value) == "" {
			return listFilter{}, appError{Code: "INVALID_FILTER", Message: fmt.Sprintf("invalid filter: %s", expr), ExitCode: 2}
		}
		return listFilter{kind: "label_eq", value: value}, nil
	case strings.HasPrefix(expr, "label~"):
		value := strings.TrimPrefix(expr, "label~")
		if strings.TrimSpace(value) == "" {
			return listFilter{}, appError{Code: "INVALID_FILTER", Message: fmt.Sprintf("invalid filter: %s", expr), ExitCode: 2}
		}
		return listFilter{kind: "label_contains", value: value}, nil
	case strings.HasPrefix(expr, "stale="):
		spec, err := state.ParseHumanDuration(strings.TrimPrefix(expr, "stale="))
		if err != nil {
			return listFilter{}, appError{Code: "INVALID_FILTER", Message: fmt.Sprintf("invalid filter: %s", expr), ExitCode: 2}
		}
		return listFilter{kind: "stale", duration: spec}, nil
	default:
		return listFilter{}, appError{Code: "INVALID_FILTER", Message: fmt.Sprintf("invalid filter: %s", expr), ExitCode: 2}
	}
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

func filterListEntries(entries []listEntry, filters []listFilter, now time.Time) ([]listEntry, error) {
	if len(filters) == 0 {
		return entries, nil
	}

	filtered := make([]listEntry, 0, len(entries))
	for _, entry := range entries {
		if matchesAllListFilters(entry, filters, now) {
			filtered = append(filtered, entry)
		}
	}
	return filtered, nil
}

func matchesAllListFilters(entry listEntry, filters []listFilter, now time.Time) bool {
	for _, filter := range filters {
		switch filter.kind {
		case "dirty":
			if !entry.item.IsDirty {
				return false
			}
		case "label_eq":
			if entry.meta.Label != filter.value {
				return false
			}
		case "label_contains":
			if !strings.Contains(entry.meta.Label, filter.value) {
				return false
			}
		case "stale":
			if entry.meta.LastUsedAt == 0 {
				return false
			}
			lastUsedAt := time.Unix(0, entry.meta.LastUsedAt)
			if now.Sub(lastUsedAt) < filter.duration.Value {
				return false
			}
		default:
			return false
		}
	}
	return true
}

type gcResultItem struct {
	Path         string   `json:"path"`
	Branch       string   `json:"branch"`
	MatchedRules []string `json:"matched_rules"`
	Action       string   `json:"action"`
	Reason       string   `json:"reason,omitempty"`
}

func gcJSONPayload(candidates []gcCandidate, results []gcResultItem) map[string]any {
	if results == nil {
		items := make([]gcResultItem, 0, len(candidates))
		for _, candidate := range candidates {
			items = append(items, gcResultItem{
				Path:         candidate.entry.item.Path,
				Branch:       candidate.entry.item.BranchLabel,
				MatchedRules: candidate.matchedRules,
				Action:       "dry_run",
			})
		}
		return map[string]any{
			"summary": map[string]any{
				"matched": len(candidates),
				"removed": 0,
				"skipped": 0,
			},
			"items": items,
		}
	}

	removed := 0
	skipped := 0
	for _, item := range results {
		switch item.Action {
		case "removed":
			removed++
		case "skipped":
			skipped++
		}
	}
	return map[string]any{
		"summary": map[string]any{
			"matched": len(candidates),
			"removed": removed,
			"skipped": skipped,
		},
		"items": results,
	}
}

func writeGCHuman(out io.Writer, results []gcResultItem) {
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
			Code:     "AMBIGUOUS_MATCH",
			Message:  "must specify a target when multiple removable worktrees exist",
			ExitCode: 2,
		}
	}

	if selected, err := worktree.Match(allItems, target); err == nil && selected.IsCurrent {
		return removalCandidate{}, appError{
			Code:     "REMOVE_CURRENT",
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

func renderRemovalCandidates(w io.Writer, candidates []removalCandidate) {
	fmt.Fprintln(w, "Remove which worktree?")
	fmt.Fprintln(w)
	hasDirty := false
	for i, c := range candidates {
		label := removalCandidateLabel(c.item)
		if c.preview.Dirty {
			fmt.Fprintf(w, "  %d  %s  ●\n", i+1, label)
			hasDirty = true
		} else {
			fmt.Fprintf(w, "  %d  %s\n", i+1, label)
		}
	}
	if hasDirty {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "● uncommitted changes")
	}
}

func removalCandidateLabel(item worktree.Worktree) string {
	if strings.TrimSpace(item.BranchLabel) != "" {
		return item.BranchLabel
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

func removeJSONPayload(result git.RemoveResult) map[string]any {
	return map[string]any{
		"worktree_path":      result.WorktreePath,
		"branch":             result.Branch,
		"base_branch":        result.BaseBranch,
		"removed_worktree":   result.RemovedWorktree,
		"deleted_branch":     result.DeletedBranch,
		"kept_branch_reason": result.KeptBranchReason,
	}
}

func writeJSONSuccess(out io.Writer, command string, data any) int {
	payload := map[string]any{
		"ok":      true,
		"command": command,
		"data":    data,
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
		"ok":      false,
		"command": command,
		"error": map[string]any{
			"code":      err.Code,
			"message":   err.Message,
			"exit_code": err.ExitCode,
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
		return appError{Code: "NOT_GIT_REPO", Message: "not a git repository", ExitCode: 3}
	case errors.Is(err, ui.ErrSelectionCanceled):
		return appError{Code: "CANCELLED", Message: "selection canceled", ExitCode: 130}
	case errors.Is(err, ui.ErrFzfNotInstalled):
		return appError{Code: "GIT_ERROR", Message: "fzf is not installed", ExitCode: 3}
	}

	message := err.Error()
	switch {
	case strings.HasPrefix(message, "ambiguous worktree match"):
		return appError{Code: "AMBIGUOUS_MATCH", Message: message, ExitCode: 2}
	case strings.HasPrefix(message, "no worktree matches"):
		return appError{Code: "WORKTREE_NOT_FOUND", Message: message, ExitCode: 2}
	case strings.Contains(message, "uncommitted changes"):
		return appError{Code: "WORKTREE_DIRTY", Message: message, ExitCode: 1}
	default:
		return appError{Code: "GIT_ERROR", Message: message, ExitCode: 1}
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
	fmt.Fprintln(out, "Usage: ww-helper [switch-path|list|new-path|init|gc|rm|help|--help]")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "switch-path prints the selected git worktree path.")
	fmt.Fprintln(out, "Interactive switch uses fzf when available, otherwise the built-in selector.")
	fmt.Fprintln(out, "list prints the current worktree table.")
	fmt.Fprintln(out, "new-path creates a worktree and prints its path.")
	fmt.Fprintln(out, "init prints shell code that activates ww for zsh or bash.")
	fmt.Fprintln(out, "gc evaluates explicit cleanup rules and prints matched worktrees.")
	fmt.Fprintln(out, "rm removes a worktree and optionally deletes its merged branch.")
	fmt.Fprintln(out, "help prints this command summary.")
}
