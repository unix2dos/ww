package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"

	"ww/internal/git"
	"ww/internal/state"
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
	TouchWorktreeState(ctx context.Context, repoKey, path string) error
	DefaultBranch(ctx context.Context) (string, error)
	PreviewRemoval(ctx context.Context, item worktree.Worktree, baseBranch string) (git.RemovalPreview, error)
	RemoveWorktree(ctx context.Context, item worktree.Worktree, opts git.RemoveOptions) (git.RemoveResult, error)
	DiffSummary(ctx context.Context, targetBranch string) (git.DiffReport, error)
	DiffPatch(ctx context.Context, targetBranch string) (string, error)
}

type RealDeps struct{}

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

func (d RealDeps) TouchWorktreeState(_ context.Context, repoKey, path string) error {
	store, err := ensureStore()
	if err != nil {
		return err
	}
	return store.Touch(repoKey, path)
}

func (d RealDeps) DefaultBranch(ctx context.Context) (string, error) {
	return git.DefaultBranch(ctx, git.ExecRunner{})
}

func (d RealDeps) PreviewRemoval(ctx context.Context, item worktree.Worktree, baseBranch string) (git.RemovalPreview, error) {
	return git.PreviewRemoval(ctx, git.ExecRunner{}, item, baseBranch)
}

func (d RealDeps) RemoveWorktree(ctx context.Context, item worktree.Worktree, opts git.RemoveOptions) (git.RemoveResult, error) {
	return git.RemoveWorktree(ctx, git.ExecRunner{}, item, opts)
}

func (d RealDeps) DiffSummary(ctx context.Context, targetBranch string) (git.DiffReport, error) {
	return git.DiffSummary(ctx, git.ExecRunner{}, targetBranch)
}

func (d RealDeps) DiffPatch(ctx context.Context, targetBranch string) (string, error) {
	return git.DiffPatch(ctx, git.ExecRunner{}, targetBranch)
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
	case "switch-path":
		return runSwitchPath(ctx, args[1:], in, out, errOut, deps)
	case "new-path":
		return runNewPath(ctx, args[1:], out, errOut, deps)
	case "list":
		return runList(ctx, out, errOut, deps)
	case "rm":
		return runRemove(ctx, args[1:], in, out, errOut, deps)
	case "diff":
		return runDiff(ctx, args[1:], in, out, errOut, deps)
	default:
		return runSwitchPath(ctx, args, in, out, errOut, deps)
	}
}

func runSwitchPath(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
	if len(args) > 0 && args[0] == "--fzf" {
		if len(args) > 1 {
			fmt.Fprintf(errOut, "unexpected extra arguments: %s\n", strings.Join(args[1:], " "))
			return 2
		}

		repoKey, items, warn, err := orderedWorktrees(ctx, deps)
		if err != nil {
			return writeWorktreeError(errOut, err)
		}
		warnStateIssue(errOut, warn)

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
		repoKey, items, warn, err := orderedWorktrees(ctx, deps)
		if err != nil {
			return writeWorktreeError(errOut, err)
		}
		warnStateIssue(errOut, warn)

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

	repoKey, items, warn, err := orderedWorktrees(ctx, deps)
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

func runList(ctx context.Context, out io.Writer, errOut io.Writer, deps Deps) int {
	_, items, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeWorktreeError(errOut, err)
	}
	warnStateIssue(errOut, warn)
	if len(items) == 0 {
		fmt.Fprintln(errOut, "no worktrees available")
		return 1
	}

	for _, item := range items {
		status := ""
		if item.IsCurrent {
			status = "ACTIVE"
		}
		fmt.Fprintf(out, "[%d] %-6s %s %s\n", item.Index, status, item.BranchLabel, item.Path)
	}
	return 0
}

func runNewPath(ctx context.Context, args []string, out io.Writer, errOut io.Writer, deps Deps) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "missing worktree name")
		return 2
	}
	if len(args) > 1 {
		fmt.Fprintf(errOut, "unexpected extra arguments: %s\n", strings.Join(args[1:], " "))
		return 2
	}

	repoKey, err := deps.CurrentRepoKey(ctx)
	if err != nil {
		return writeWorktreeError(errOut, err)
	}

	path, err := deps.CreateWorktree(ctx, args[0])
	if err != nil {
		return writeWorktreeError(errOut, err)
	}

	fmt.Fprintln(out, path)
	warnStateIssue(errOut, touchWorktreeStateBestEffort(ctx, deps, repoKey, path))
	return 0
}

type removeConfig struct {
	force  bool
	json   bool
	base   string
	target string
}

type diffConfig struct {
	patch  bool
	target string
}

type removalCandidate struct {
	item    worktree.Worktree
	preview git.RemovalPreview
}

type diffTarget struct {
	branch    string
	path      string
	isDefault bool
	virtual   bool
}

func runRemove(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
	cfg, err := parseRemoveArgs(args)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}

	_, items, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeWorktreeError(errOut, err)
	}
	warnStateIssue(errOut, warn)

	candidates := filterNonCurrent(items)
	if len(candidates) == 0 {
		fmt.Fprintln(errOut, "no removable worktrees available")
		return 1
	}

	baseBranch := cfg.base
	if baseBranch == "" {
		baseBranch, err = deps.DefaultBranch(ctx)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	}

	previewed := make([]removalCandidate, 0, len(candidates))
	for _, item := range candidates {
		preview, err := deps.PreviewRemoval(ctx, item, baseBranch)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		previewed = append(previewed, removalCandidate{item: item, preview: preview})
	}

	reader := bufio.NewReader(in)
	selected, ok, exitCode := selectRemovalCandidate(reader, errOut, previewed, cfg.target)
	if !ok {
		return exitCode
	}

	fmt.Fprintf(errOut, "Selected: %s %s [%s]\n", selected.item.BranchLabel, selected.item.Path, removalAction(selected.preview))
	confirmed, err := confirmPrompt(reader, errOut, fmt.Sprintf("Remove worktree %s? [y/N]: ", selected.item.BranchLabel))
	if err != nil {
		return writeSelectionError(errOut, err)
	}
	if !confirmed {
		return 130
	}

	result, err := deps.RemoveWorktree(ctx, selected.item, git.RemoveOptions{
		BaseBranch: baseBranch,
		Force:      cfg.force,
	})
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	if cfg.json {
		return writeRemoveJSON(out, result)
	}
	writeRemoveHuman(out, result)
	return 0
}

func runDiff(ctx context.Context, args []string, in io.Reader, out io.Writer, errOut io.Writer, deps Deps) int {
	cfg, err := parseDiffArgs(args)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}

	_, items, warn, err := orderedWorktrees(ctx, deps)
	if err != nil {
		return writeWorktreeError(errOut, err)
	}
	warnStateIssue(errOut, warn)

	targetBranch, err := resolveDiffTarget(ctx, cfg, in, errOut, items, deps)
	if err != nil {
		if errors.Is(err, ui.ErrSelectionCanceled) {
			return 130
		}
		if errors.Is(err, io.EOF) {
			return 1
		}
		fmt.Fprintln(errOut, err)
		return 1
	}

	report, err := deps.DiffSummary(ctx, targetBranch)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	writeDiffSummary(out, report)

	if cfg.patch {
		patch, err := deps.DiffPatch(ctx, targetBranch)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		if patch != "" {
			fmt.Fprintln(out)
			fmt.Fprint(out, patch)
			if !strings.HasSuffix(patch, "\n") {
				fmt.Fprintln(out)
			}
		}
	}
	return 0
}

func orderedWorktrees(ctx context.Context, deps Deps) (string, []worktree.Worktree, error, error) {
	repoKey, items, err := deps.ListWorktrees(ctx)
	if err != nil {
		return "", nil, nil, err
	}
	mru, err := deps.LoadWorktreeState(ctx, repoKey)
	if err != nil {
		normalized := worktree.Normalize(items)
		return repoKey, normalized, fmt.Errorf("state load unavailable: %w", err), nil
	}
	for i := range items {
		items[i].LastUsedAt = mru[items[i].Path]
	}
	return repoKey, worktree.Normalize(items), nil, nil
}

func parseRemoveArgs(args []string) (removeConfig, error) {
	var cfg removeConfig
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--force":
			cfg.force = true
		case arg == "--json":
			cfg.json = true
		case arg == "--base":
			if i+1 >= len(args) {
				return removeConfig{}, fmt.Errorf("missing value for --base")
			}
			i++
			cfg.base = args[i]
		case strings.HasPrefix(arg, "--base="):
			cfg.base = strings.TrimPrefix(arg, "--base=")
		case strings.HasPrefix(arg, "-"):
			return removeConfig{}, fmt.Errorf("unknown option: %s", arg)
		default:
			if cfg.target != "" {
				return removeConfig{}, fmt.Errorf("unexpected extra arguments: %s", strings.Join(args[i:], " "))
			}
			cfg.target = arg
		}
	}
	return cfg, nil
}

func parseDiffArgs(args []string) (diffConfig, error) {
	var cfg diffConfig
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--patch":
			cfg.patch = true
		case strings.HasPrefix(arg, "-"):
			return diffConfig{}, fmt.Errorf("unknown option: %s", arg)
		default:
			if cfg.target != "" {
				return diffConfig{}, fmt.Errorf("unexpected extra arguments: %s", strings.Join(args[i:], " "))
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

func selectRemovalCandidate(reader *bufio.Reader, errOut io.Writer, candidates []removalCandidate, target string) (removalCandidate, bool, int) {
	if target != "" {
		items := make([]worktree.Worktree, 0, len(candidates))
		byPath := make(map[string]removalCandidate, len(candidates))
		for _, candidate := range candidates {
			items = append(items, candidate.item)
			byPath[candidate.item.Path] = candidate
		}
		selected, err := worktree.Match(items, target)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return removalCandidate{}, false, 2
		}
		return byPath[selected.Path], true, 0
	}

	renderRemovalCandidates(errOut, candidates)
	index, err := readChoice(reader, errOut, "Select worktree to remove [number]: ", len(candidates), 0)
	if err != nil {
		return removalCandidate{}, false, writeSelectionError(errOut, err)
	}
	return candidates[index-1], true, 0
}

func renderRemovalCandidates(w io.Writer, candidates []removalCandidate) {
	for i, candidate := range candidates {
		fmt.Fprintf(w, "[%d] %s %s %s [%s; %s]\n",
			i+1,
			removalAction(candidate.preview),
			candidate.item.BranchLabel,
			candidate.item.Path,
			dirtyLabel(candidate.preview.Dirty),
			mergeLabel(candidate.preview),
		)
	}
}

func removalAction(preview git.RemovalPreview) string {
	if preview.DeleteBranch {
		return "DELETE_BRANCH"
	}
	return "KEEP_BRANCH"
}

func dirtyLabel(dirty bool) string {
	if dirty {
		return "DIRTY"
	}
	return "CLEAN"
}

func mergeLabel(preview git.RemovalPreview) string {
	if preview.Worktree.BranchRef == "" {
		return "DETACHED"
	}
	if preview.BranchMerged {
		return "MERGED"
	}
	return "NOT_MERGED"
}

func resolveDiffTarget(ctx context.Context, cfg diffConfig, in io.Reader, errOut io.Writer, items []worktree.Worktree, deps Deps) (string, error) {
	if cfg.target != "" {
		selected, err := worktree.Match(items, cfg.target)
		if err == nil {
			return selected.BranchLabel, nil
		}
		if !strings.Contains(err.Error(), "no worktree matches") {
			return "", err
		}
		return cfg.target, nil
	}

	defaultBranch, err := deps.DefaultBranch(ctx)
	if err != nil {
		return "", err
	}

	targets, defaultIndex, err := buildDiffTargets(items, defaultBranch)
	if err != nil {
		return "", err
	}
	renderDiffTargets(errOut, targets, defaultIndex)

	reader := bufio.NewReader(in)
	index, err := readChoice(reader, errOut, fmt.Sprintf("Select diff target [number, default %d]: ", defaultIndex), len(targets), defaultIndex)
	if err != nil {
		return "", err
	}
	return targets[index-1].branch, nil
}

func buildDiffTargets(items []worktree.Worktree, defaultBranch string) ([]diffTarget, int, error) {
	current := currentWorktree(items)
	targets := make([]diffTarget, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		if item.IsCurrent {
			continue
		}
		targets = append(targets, diffTarget{branch: item.BranchLabel, path: item.Path})
		seen[item.BranchLabel] = struct{}{}
	}

	if current.BranchLabel != defaultBranch {
		if _, ok := seen[defaultBranch]; !ok {
			targets = append(targets, diffTarget{branch: defaultBranch, virtual: true})
		}
	}
	if len(targets) == 0 {
		return nil, 0, fmt.Errorf("no diff targets available")
	}

	sort.SliceStable(targets, func(i, j int) bool {
		if targets[i].branch != targets[j].branch {
			return targets[i].branch < targets[j].branch
		}
		return targets[i].path < targets[j].path
	})

	defaultIndex := 1
	if current.BranchLabel != defaultBranch {
		for i := range targets {
			if targets[i].branch == defaultBranch {
				targets[i].isDefault = true
				defaultIndex = i + 1
				break
			}
		}
	} else {
		targets[0].isDefault = true
	}

	return targets, defaultIndex, nil
}

func currentWorktree(items []worktree.Worktree) worktree.Worktree {
	for _, item := range items {
		if item.IsCurrent {
			return item
		}
	}
	return worktree.Worktree{}
}

func renderDiffTargets(w io.Writer, targets []diffTarget, defaultIndex int) {
	for i, target := range targets {
		suffix := ""
		if i+1 == defaultIndex {
			suffix = " [default]"
		}
		if target.path != "" {
			fmt.Fprintf(w, "[%d] %s %s%s\n", i+1, target.branch, target.path, suffix)
			continue
		}
		fmt.Fprintf(w, "[%d] %s <branch>%s\n", i+1, target.branch, suffix)
	}
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
	fmt.Fprintf(out, "removed worktree %s\n", result.WorktreePath)
	if result.DeletedBranch {
		fmt.Fprintf(out, "deleted branch %s\n", result.Branch)
		return
	}
	if result.Branch != "" && result.KeptBranchReason != "" {
		fmt.Fprintf(out, "kept branch %s (%s)\n", result.Branch, result.KeptBranchReason)
	}
}

func writeRemoveJSON(out io.Writer, result git.RemoveResult) int {
	payload := map[string]any{
		"worktree_path":      result.WorktreePath,
		"branch":             result.Branch,
		"base_branch":        result.BaseBranch,
		"removed_worktree":   result.RemovedWorktree,
		"deleted_branch":     result.DeletedBranch,
		"kept_branch_reason": result.KeptBranchReason,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return 1
	}
	fmt.Fprintln(out, string(encoded))
	return 0
}

func writeDiffSummary(out io.Writer, report git.DiffReport) {
	fmt.Fprintf(out, "Current: %s\n", report.CurrentBranch)
	fmt.Fprintf(out, "Target: %s\n", report.TargetBranch)
	fmt.Fprintf(out, "Ahead: %d\n", report.Ahead)
	fmt.Fprintf(out, "Behind: %d\n", report.Behind)
	fmt.Fprintf(out, "Files: %d changed, %d insertions(+), %d deletions(-)\n", report.ChangedFiles, report.Insertions, report.Deletions)
	if len(report.Commits) > 0 {
		fmt.Fprintln(out, "Commits:")
		for _, commit := range report.Commits {
			fmt.Fprintf(out, "- %s %s\n", shortHash(commit.Hash), commit.Subject)
		}
	}
	if len(report.Files) > 0 {
		fmt.Fprintln(out, "Files:")
		for _, file := range report.Files {
			fmt.Fprintf(out, "- %s (+%d -%d)\n", file.Path, file.Insertions, file.Deletions)
		}
	}
}

func shortHash(hash string) string {
	if len(hash) <= 7 {
		return hash
	}
	return hash[:7]
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

func warnStateIssue(errOut io.Writer, err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(errOut, err)
}

func printHelperHelp(out io.Writer) {
	fmt.Fprintln(out, "Usage: ww-helper [switch-path|list|new-path|rm|diff|--help]")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "switch-path prints the selected git worktree path.")
	fmt.Fprintln(out, "Interactive switch uses fzf when available, otherwise the built-in selector.")
	fmt.Fprintln(out, "list prints the current worktree table.")
	fmt.Fprintln(out, "new-path creates a worktree and prints its path.")
	fmt.Fprintln(out, "rm removes a worktree and optionally deletes its merged branch.")
	fmt.Fprintln(out, "diff prints a PR-style diff summary for the current branch.")
}
