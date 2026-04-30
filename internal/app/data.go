// Package app exposes data-layer functions that the CLI subcommands call to
// produce their JSON envelopes. These are the same entry points the MCP
// server uses, so JSON output and MCP tool output stay structurally
// identical without duplication.
//
// Each Data function is a pure-data alternative to a `run<Command>`
// function: it takes parsed options and returns typed data, leaving
// argument parsing and output formatting to the caller.
package app

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"ww/internal/git"
	"ww/internal/state"
	"ww/internal/syncignored"
	"ww/internal/worktree"
)

// ErrorCode returns the protocol-level `code` for an error originating in
// this package, or "" if err is not one of the package's typed errors. It
// is the cross-package way to ask "what does the JSON envelope's
// error.code say?" without importing private types.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	classified := classifyError(err)
	return classified.Code
}

// Warning is one entry in an envelope's `warnings` array. Codes follow the
// `domain.subcode` convention (sync.copied, sync.skipped, sync.failed, …)
// and are stable within a v1.x major; messages and context keys may evolve.
type Warning struct {
	Code    string         `json:"code"`
	Message string         `json:"message,omitempty"`
	Context map[string]any `json:"context,omitempty"`
}

// WorktreeView is the protocol-aligned view of one worktree. The field names
// and types match the v1.0 wire contract for `list --json`; in-process
// callers (notably the MCP server) marshal these directly to clients.
type WorktreeView struct {
	Path       string `json:"path"`
	Branch     string `json:"branch"`
	Dirty      bool   `json:"dirty"`
	Active     bool   `json:"active"`
	CreatedAt  int64  `json:"created_at"`   // unix milliseconds; 0 if unknown
	LastUsedAt int64  `json:"last_used_at"` // unix milliseconds; 0 if never
	Label      string `json:"label"`
	TTL        string `json:"ttl"`
	Merged     bool   `json:"merged"`
	Ahead      int    `json:"ahead"`
	Behind     int    `json:"behind"`
	Staged     int    `json:"staged"`
	Unstaged   int    `json:"unstaged"`
	Untracked  int    `json:"untracked"`
}

// ListOptions configures ListData.
type ListOptions struct {
	Filters []listFilter
}

// GCOptions configures GCData. At least one of TTLExpired, IdleSet, or Merged
// must be true; the CLI's argument parser already enforces this with
// `input.missing_selector`, so the data layer assumes the constraint holds.
type GCOptions struct {
	TTLExpired bool
	IdleSet    bool
	Idle       state.DurationSpec
	Merged     bool
	DryRun     bool
	Force      bool
	Base       string // overrides DefaultBranch when non-empty
}

// GCSummary mirrors `gc.summary` in the wire protocol.
type GCSummary struct {
	Matched int `json:"matched"`
	Removed int `json:"removed"`
	Skipped int `json:"skipped"`
}

// GCItem mirrors one element of `gc.items` in the wire protocol.
type GCItem struct {
	Path         string   `json:"path"`
	Branch       string   `json:"branch"`
	MatchedRules []string `json:"matched_rules"`
	Action       string   `json:"action"`           // "removed" | "skipped" | "dry_run"
	Reason       string   `json:"reason,omitempty"` // populated when Action == "skipped"
}

// GCResult is the v1.0 wire shape of `gc --json`.
type GCResult struct {
	Summary GCSummary `json:"summary"`
	Items   []GCItem  `json:"items"`
}

// GCData evaluates the configured selectors against existing worktrees and,
// unless DryRun is set, removes the matches. Dirty worktrees are skipped
// unless Force is set; the caller's current worktree is always skipped.
func GCData(ctx context.Context, deps Deps, opts GCOptions) (GCResult, error) {
	_, items, metadata, _, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return GCResult{}, err
	}

	now := time.Now()
	entries := decorateListEntries(items, metadata)
	baseBranch := opts.Base
	candidates := make([]gcCandidate, 0, len(entries))
	for _, entry := range entries {
		candidate := gcCandidate{entry: entry}
		if opts.TTLExpired && ttlExpired(entry.meta, now) {
			candidate.matchedRules = append(candidate.matchedRules, "ttl_expired")
		}
		if opts.IdleSet && idleExpired(entry.meta, opts.Idle, now) {
			candidate.matchedRules = append(candidate.matchedRules, "idle")
		}
		if opts.Merged && entry.item.BranchRef != "" {
			if baseBranch == "" {
				baseBranch, err = deps.DefaultBranch(ctx)
				if err != nil {
					return GCResult{}, err
				}
			}
			preview, previewErr := deps.PreviewRemoval(ctx, entry.item, baseBranch)
			if previewErr != nil {
				return GCResult{}, previewErr
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

	if opts.DryRun {
		items := make([]GCItem, 0, len(candidates))
		for _, candidate := range candidates {
			items = append(items, GCItem{
				Path:         candidate.entry.item.Path,
				Branch:       candidate.entry.item.BranchLabel,
				MatchedRules: candidate.matchedRules,
				Action:       "dry_run",
			})
		}
		return GCResult{
			Summary: GCSummary{Matched: len(candidates)},
			Items:   items,
		}, nil
	}

	results := make([]GCItem, 0, len(candidates))
	for _, candidate := range candidates {
		item := candidate.entry.item
		if item.IsCurrent {
			results = append(results, GCItem{
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
					return GCResult{}, err
				}
			}
			preview, err = deps.PreviewRemoval(ctx, item, baseBranch)
			if err != nil {
				return GCResult{}, err
			}
		}
		if preview.Dirty && !opts.Force {
			results = append(results, GCItem{
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
			Force:      opts.Force,
		})
		if removeErr != nil {
			return GCResult{}, removeErr
		}
		results = append(results, GCItem{
			Path:         removeResult.WorktreePath,
			Branch:       removeResult.Branch,
			MatchedRules: candidate.matchedRules,
			Action:       "removed",
		})
	}

	removed, skipped := 0, 0
	for _, item := range results {
		switch item.Action {
		case "removed":
			removed++
		case "skipped":
			skipped++
		}
	}
	return GCResult{
		Summary: GCSummary{Matched: len(candidates), Removed: removed, Skipped: skipped},
		Items:   results,
	}, nil
}

// RemoveOptions configures RemoveData. Target is required; an empty Target
// is rejected. The data layer never prompts; the CLI's interactive prompt
// flow stays in `runRemove`.
type RemoveOptions struct {
	Target string
	Force  bool
}

// RemoveResult is the v1.0 wire shape of `rm --json`.
type RemoveResult struct {
	WorktreePath     string `json:"worktree_path"`
	Branch           string `json:"branch"`
	BaseBranch       string `json:"base_branch"`
	RemovedWorktree  bool   `json:"removed_worktree"`
	DeletedBranch    bool   `json:"deleted_branch"`
	KeptBranchReason string `json:"kept_branch_reason"`
}

// RemoveData removes a worktree by name/path/index. It performs the same
// dirty-check as the CLI's JSON path and refuses to remove the current
// worktree.
func RemoveData(ctx context.Context, deps Deps, opts RemoveOptions) (RemoveResult, error) {
	if opts.Target == "" {
		return RemoveResult{}, appError{
			Code:     "worktree.not_found",
			Message:  "no worktree target specified",
			ExitCode: 2,
		}
	}

	_, items, _, _, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return RemoveResult{}, err
	}

	// Refuse to remove the current worktree.
	if selected, matchErr := worktree.Match(items, opts.Target); matchErr == nil && selected.IsCurrent {
		return RemoveResult{}, appError{
			Code:     "worktree.remove_current",
			Message:  "Cannot remove the current worktree. Switch first: ww go <name>",
			ExitCode: 1,
		}
	}

	candidates := filterNonCurrent(items)
	if len(candidates) == 0 {
		return RemoveResult{}, appError{
			Code:     "worktree.not_found",
			Message:  "no removable worktrees available",
			ExitCode: 1,
		}
	}

	baseBranch, err := deps.DefaultBranch(ctx)
	if err != nil {
		return RemoveResult{}, err
	}

	previewed := make([]removalCandidate, 0, len(candidates))
	for _, item := range candidates {
		preview, err := deps.PreviewRemoval(ctx, item, baseBranch)
		if err != nil {
			return RemoveResult{}, err
		}
		previewed = append(previewed, removalCandidate{item: item, preview: preview})
	}

	selected, err := selectRemovalCandidateNonInteractive(items, previewed, opts.Target)
	if err != nil {
		return RemoveResult{}, err
	}

	if selected.preview.Dirty && !opts.Force {
		return RemoveResult{}, appError{
			Code:     "worktree.dirty",
			Message:  "worktree has uncommitted changes; rerun with --force",
			ExitCode: 1,
		}
	}

	gitResult, err := deps.RemoveWorktree(ctx, selected.item, git.RemoveOptions{
		BaseBranch: baseBranch,
		Force:      opts.Force,
	})
	if err != nil {
		return RemoveResult{}, err
	}

	return RemoveResult{
		WorktreePath:     gitResult.WorktreePath,
		Branch:           gitResult.Branch,
		BaseBranch:       gitResult.BaseBranch,
		RemovedWorktree:  gitResult.RemovedWorktree,
		DeletedBranch:    gitResult.DeletedBranch,
		KeptBranchReason: gitResult.KeptBranchReason,
	}, nil
}

// SwitchPathResult is the wire shape of a non-interactive path resolution.
// The CLI's `switch-path` command does not use this — it emits the path as
// raw stdout (out-of-contract per protocol §4.3) so `cd "$(ww-helper switch-path X)"`
// works. MCP wraps the same resolution in this envelope-friendly type.
type SwitchPathResult struct {
	Path string `json:"path"`
}

// SwitchPathData resolves a worktree target to its absolute path and touches
// its last-used metadata. Target must be a name (substring match) or a 1-based
// index as a numeric string. Empty target is rejected; interactive selection
// is the CLI's responsibility, not the data layer's.
func SwitchPathData(ctx context.Context, deps Deps, target string) (SwitchPathResult, error) {
	if target == "" {
		return SwitchPathResult{}, appError{
			Code:     "input.invalid_argument",
			Message:  "switch-path requires a target",
			ExitCode: 2,
		}
	}

	repoKey, items, _, _, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return SwitchPathResult{}, err
	}

	if idx, parseErr := strconv.Atoi(target); parseErr == nil {
		if idx <= 0 {
			return SwitchPathResult{}, appError{
				Code:     "input.invalid_argument",
				Message:  fmt.Sprintf("invalid worktree index: %q", target),
				ExitCode: 2,
			}
		}
		selected, ok := selectByIndex(items, idx)
		if !ok {
			return SwitchPathResult{}, appError{
				Code:     "worktree.not_found",
				Message:  fmt.Sprintf("worktree index %d out of range", idx),
				ExitCode: 2,
			}
		}
		_ = touchWorktreeStateBestEffort(ctx, deps, repoKey, selected.Path)
		return SwitchPathResult{Path: selected.Path}, nil
	}

	selected, err := worktree.Match(items, target)
	if err != nil {
		// worktree.Match's error messages already start with the protocol's
		// expected prefixes ("ambiguous worktree match", "no worktree
		// matches"); classifyError will translate them to the right codes.
		return SwitchPathResult{}, err
	}
	_ = touchWorktreeStateBestEffort(ctx, deps, repoKey, selected.Path)
	return SwitchPathResult{Path: selected.Path}, nil
}

// VersionResult is the v1.0 wire shape of `version --json`. The protocol
// version is reported via the envelope's top-level `protocol` field, not
// inside `data`, to avoid skew between the two.
type VersionResult struct {
	Binary string `json:"binary"`
}

// VersionData returns the binary's build version. The CLI subcommand wraps
// this; MCP exposes it as the `ww_version` tool.
func VersionData() VersionResult {
	return VersionResult{Binary: binaryVersion}
}

// NewPathOptions configures NewPathData.
type NewPathOptions struct {
	Name    string
	Label   string
	TTL     string
	Message string

	// Sync controls whether git-ignored files (e.g. .env, local config) are
	// copied from the main worktree into the freshly created one. The CLI
	// JSON path and the MCP server set this to true by default; pass false
	// for opt-out (`--no-sync`). Sync is best-effort and failures are
	// surfaced as warnings rather than errors.
	Sync       bool
	SyncDryRun bool
}

// NewPathResult is the v1.0 wire shape of `new-path --json`.
type NewPathResult struct {
	WorktreePath string `json:"worktree_path"`
	Branch       string `json:"branch"`
}

// NewPathData creates a worktree, records metadata, writes a task note if
// labeled, and (if opts.Sync is true) copies ignored files from the main
// worktree. Sync results are returned as warnings; sync errors do not fail
// the operation.
//
// Name validation (non-empty) is the caller's responsibility — the CLI's
// argument parser handles it before reaching this function.
func NewPathData(ctx context.Context, deps Deps, opts NewPathOptions) (NewPathResult, []Warning, error) {
	repoKey, err := deps.CurrentRepoKey(ctx)
	if err != nil {
		return NewPathResult{}, nil, err
	}

	path, err := deps.CreateWorktree(ctx, opts.Name)
	if err != nil {
		return NewPathResult{}, nil, err
	}

	meta := state.WorktreeMetadata{
		CreatedAt: time.Now().UnixNano(),
		Label:     opts.Label,
		TTL:       opts.TTL,
	}
	createdAt := time.Unix(0, meta.CreatedAt).UTC()

	if err := recordWorktreeStateBestEffort(ctx, deps, repoKey, path, meta); err != nil {
		return NewPathResult{}, nil, err
	}
	if err := createTaskNoteIfLabeled(ctx, deps, path, opts.Name, opts.Label, opts.Message, createdAt); err != nil {
		return NewPathResult{}, nil, err
	}
	if err := touchWorktreeStateBestEffort(ctx, deps, repoKey, path); err != nil {
		return NewPathResult{}, nil, err
	}

	var warnings []Warning
	if opts.Sync {
		warnings = syncIgnoredAsWarnings(ctx, repoKey, path, opts.SyncDryRun)
	}

	return NewPathResult{WorktreePath: path, Branch: opts.Name}, warnings, nil
}

// syncIgnoredAsWarnings runs the ignored-file sync and translates its result
// into protocol-shaped Warning entries. It mirrors runSyncIgnored's behavior
// but routes output through warnings instead of stderr text.
func syncIgnoredAsWarnings(ctx context.Context, repoKey, newPath string, dryRun bool) []Warning {
	mainRoot := mainWorktreeRootFromRepoKey(repoKey)
	if mainRoot == "" || mainRoot == newPath {
		return nil
	}

	var warnings []Warning
	userCfg, cfgErr := loadSyncConfigFn()
	if cfgErr != nil {
		warnings = append(warnings, Warning{
			Code:    "sync.config_error",
			Message: cfgErr.Error(),
		})
		// Fall through with defaults, matching runSyncIgnored.
	}

	if !userCfg.Sync.SyncEnabled() {
		return warnings
	}

	syncOpts := syncignored.Options{
		Enabled:   true,
		Blacklist: userCfg.Sync.EffectiveBlacklist(syncignored.DefaultBlacklist),
		DryRun:    dryRun,
	}
	if v := userCfg.Sync.EffectiveMaxFileSize(); v > 0 {
		syncOpts.MaxFileSize = v
	}

	res, err := syncIgnoredFn(ctx, mainRoot, newPath, syncOpts)
	if err != nil {
		warnings = append(warnings, Warning{
			Code:    "sync.failed",
			Message: err.Error(),
		})
		return warnings
	}

	for _, copied := range res.Copied {
		ctxMap := map[string]any{"file": copied}
		if res.DryRun {
			ctxMap["dry_run"] = true
		}
		warnings = append(warnings, Warning{Code: "sync.copied", Context: ctxMap})
	}
	for _, sk := range res.Skipped {
		ctxMap := map[string]any{
			"file":   sk.Path,
			"reason": string(sk.Reason),
		}
		if sk.Size > 0 {
			ctxMap["size"] = sk.Size
		}
		warnings = append(warnings, Warning{Code: "sync.skipped", Context: ctxMap})
	}
	return warnings
}

// ListData returns worktrees in the current repository, optionally filtered.
// It does not write to any io.Writer; the CLI subcommand (`runList`) wraps
// this with output rendering.
//
// The second return value is a non-fatal state-load warning — when it is
// non-nil the data is still valid but some metadata may be missing. CLI
// callers print it to stderr in human mode and suppress it in JSON mode;
// MCP callers can surface it via the envelope's `warnings` array.
func ListData(ctx context.Context, deps Deps, opts ListOptions) ([]WorktreeView, error, error) {
	_, items, metadata, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return nil, nil, err
	}

	annotateExtendedStatusBestEffort(ctx, deps, items)

	entries, err := filterListEntries(decorateListEntries(items, metadata), opts.Filters, time.Now())
	if err != nil {
		return nil, warn, err
	}

	views := make([]WorktreeView, 0, len(entries))
	for _, entry := range entries {
		item := entry.item
		views = append(views, WorktreeView{
			Path:       item.Path,
			Branch:     item.BranchLabel,
			Dirty:      item.IsDirty,
			Active:     item.IsCurrent,
			CreatedAt:  nanosToMillis(item.CreatedAt),
			LastUsedAt: nanosToMillis(entry.meta.LastUsedAt),
			Label:      entry.meta.Label,
			TTL:        entry.meta.TTL,
			Merged:     item.IsMerged,
			Ahead:      item.Ahead,
			Behind:     item.Behind,
			Staged:     item.Staged,
			Unstaged:   item.Unstaged,
			Untracked:  item.Untracked,
		})
	}

	return views, warn, nil
}
