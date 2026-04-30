package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestCLISelectsOldestIndexPath(t *testing.T) {
	repo := newTestRepo(t)
	second := repo.Root
	repo.AddWorktree(t, "alpha")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "switch-path", "1")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got != second {
		t.Fatalf("expected second worktree path %q, got %q", second, got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestCLISelectsWorktreePathByName(t *testing.T) {
	repo := newTestRepo(t)
	alpha := repo.AddWorktree(t, "alpha")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "switch-path", "alpha")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got != alpha {
		t.Fatalf("expected named worktree path %q, got %q", alpha, got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestCLISelectsWorktreePathByUniquePrefix(t *testing.T) {
	repo := newTestRepo(t)
	alpha := repo.AddWorktree(t, "alpha")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "switch-path", "alp")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got != alpha {
		t.Fatalf("expected unique-prefix worktree path %q, got %q", alpha, got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestCLIAmbiguousPrefixReturnsError(t *testing.T) {
	repo := newTestRepo(t)
	runGit(t, repo.Root, "branch", "alpine")
	repo.AddWorktree(t, "alpha")
	repo.AddWorktree(t, "alpine")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "switch-path", "alp")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected ambiguous prefix to fail")
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "ambiguous worktree match") {
		t.Fatalf("expected ambiguous-match message, got %q", stderr.String())
	}
}

func TestCLIListsWorktrees(t *testing.T) {
	repo := newTestRepo(t)
	repo.AddWorktree(t, "alpha")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "list")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "│ INDEX │ STATUS") || !strings.Contains(stdout.String(), "│ 1     │ [CURRENT]") || !strings.Contains(stdout.String(), "/.worktrees/") || !strings.Contains(stdout.String(), "alpha") {
		t.Fatalf("expected human-readable list output, got %q", stdout.String())
	}
}

func TestCLIListStaysInCreationOrderAfterSwitch(t *testing.T) {
	repo := newTestRepo(t)
	repo.AddWorktree(t, "alpha")
	runGit(t, repo.Root, "branch", "beta")
	repo.AddWorktree(t, "beta")
	bin := buildCLI(t)
	stateHome := t.TempDir()

	switchCmd := exec.CommandContext(context.Background(), bin, "switch-path", "beta")
	switchCmd.Dir = repo.Root
	switchCmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateHome)
	if out, err := switchCmd.CombinedOutput(); err != nil {
		t.Fatalf("switch-path beta failed: %v\n%s", err, out)
	}

	listCmd := exec.CommandContext(context.Background(), bin, "list")
	listCmd.Dir = repo.Root
	listCmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateHome)

	var stdout, stderr bytes.Buffer
	listCmd.Stdout = &stdout
	listCmd.Stderr = &stderr

	if err := listCmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	got := stdout.String()
	if strings.Index(got, "│ 1     │ [CURRENT]") > strings.Index(got, "/.worktrees/alpha") {
		t.Fatalf("expected main before alpha in creation ordering, got %q", got)
	}
	if strings.Index(got, "/.worktrees/alpha") > strings.Index(got, "/.worktrees/beta") {
		t.Fatalf("expected alpha before beta in creation ordering, got %q", got)
	}
}

func TestCLIListShowsDirtyWorktrees(t *testing.T) {
	repo := newTestRepo(t)
	alpha := repo.AddWorktree(t, "alpha")
	if err := os.WriteFile(filepath.Join(alpha, "scratch.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "list")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}
	out := stdout.String()
	stripped := stripAnsi(out)
	// alpha has an untracked file — CHANGES column should show ?1
	if !strings.Contains(stripped, "CHANGES") {
		t.Fatalf("expected CHANGES column header in list output, got %q", stripped)
	}
	if !strings.Contains(stripped, "?1") {
		t.Fatalf("expected ?1 in changes column for dirty alpha, got %q", stripped)
	}
	if !strings.Contains(stripped, "[CURRENT]") {
		t.Fatalf("expected [CURRENT] tag for main, got %q", stripped)
	}
}

func TestCLICreatesNewWorktreePath(t *testing.T) {
	repo := newTestRepo(t)
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "new-path", "beta")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	want := filepath.Join(repo.Root, ".worktrees", "beta")
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Fatalf("expected new worktree path %q, got %q", want, got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected worktree path to exist: %v", err)
	}
}

func TestCLICreatesNewWorktreePathFromLinkedWorktreeAtRepositoryRoot(t *testing.T) {
	repo := newTestRepo(t)
	runGit(t, repo.Root, "branch", "abc")
	linked := repo.AddWorktree(t, "abc")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "new-path", "feature/lw-0320")
	cmd.Dir = linked

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	want := filepath.Join(repo.Root, ".worktrees", "feature", "lw-0320")
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Fatalf("expected new worktree path %q, got %q", want, got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected worktree path to exist: %v", err)
	}
}

func TestCLIRemovesMergedWorktreeAndBranch(t *testing.T) {
	repo := newTestRepo(t)
	alpha := repo.AddWorktree(t, "alpha")
	runGit(t, repo.Root, "merge", "--no-ff", "alpha", "-m", "merge alpha")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "rm", "alpha")
	cmd.Dir = repo.Root
	cmd.Stdin = strings.NewReader("y\n")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Remove alpha?") {
		t.Fatalf("expected confirmation prompt on stderr, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Removed alpha (branch deleted)") {
		t.Fatalf("expected branch-deleted summary on stdout, got %q", stdout.String())
	}
	if _, err := os.Stat(alpha); !os.IsNotExist(err) {
		t.Fatalf("expected removed worktree path to disappear, got err=%v", err)
	}
	out := runGitOutput(t, repo.Root, "branch", "--list", "alpha")
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected alpha branch to be deleted, got %q", out)
	}
}

func TestCLIRemovesWorktreeButKeepsUnmergedBranch(t *testing.T) {
	repo := newTestRepo(t)
	alpha := repo.AddWorktree(t, "alpha")
	if err := os.WriteFile(filepath.Join(alpha, "README.md"), []byte("alpha-only\n"), 0o644); err != nil {
		t.Fatalf("write feature file: %v", err)
	}
	runGit(t, alpha, "add", "README.md")
	runGit(t, alpha, "commit", "-m", "alpha only change")
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "rm", "alpha")
	cmd.Dir = repo.Root
	cmd.Stdin = strings.NewReader("y\n")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Removed alpha (branch kept, not merged)") {
		t.Fatalf("expected keep-branch output, got %q", stdout.String())
	}
	if _, err := os.Stat(alpha); !os.IsNotExist(err) {
		t.Fatalf("expected removed worktree path to disappear, got err=%v", err)
	}
	out := runGitOutput(t, repo.Root, "branch", "--list", "alpha")
	if !strings.Contains(out, "alpha") {
		t.Fatalf("expected alpha branch to remain, got %q", out)
	}
}

func TestCLIRmDirtyWorktreeStopsBeforeConfirmationWithoutForce(t *testing.T) {
	repo := newTestRepo(t)
	alpha := repo.AddWorktree(t, "alpha")
	if err := os.WriteFile(filepath.Join(alpha, "README.md"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}
	bin := buildCLI(t)

	cmd := exec.CommandContext(context.Background(), bin, "rm", "alpha")
	cmd.Dir = repo.Root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected dirty removal to fail without --force")
	}
	if !strings.Contains(stderr.String(), "uncommitted changes") || !strings.Contains(stderr.String(), "--force") {
		t.Fatalf("expected dirty-stop message mentioning uncommitted changes and --force, got %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "Remove alpha?") {
		t.Fatalf("expected no confirmation prompt for dirty worktree, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
	if _, err := os.Stat(alpha); err != nil {
		t.Fatalf("expected dirty worktree to remain, got err=%v", err)
	}
}

func TestCLIListMigratesV1StateToV2(t *testing.T) {
	repo := newTestRepo(t)
	alpha := repo.AddWorktree(t, "alpha")
	bin := buildCLI(t)
	stateHome := t.TempDir()
	repoKey := filepath.Join(repo.Root, ".git")

	stateDir := filepath.Join(stateHome, "ww")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	original := []byte("{\n  \"repos\": {\n    \"" + repoKey + "\": {\n      \"" + alpha + "\": 123000000\n    }\n  }\n}\n")
	v1Path := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(v1Path, original, 0o644); err != nil {
		t.Fatalf("write v1 state: %v", err)
	}

	cmd := exec.CommandContext(context.Background(), bin, "list", "--json")
	cmd.Dir = repo.Root
	cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateHome)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	var envelope struct {
		OK      bool            `json:"ok"`
		Command string          `json:"command"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode list envelope: %v\nraw=%s", err, stdout.String())
	}
	var items []struct {
		Path       string `json:"path"`
		LastUsedAt int64  `json:"last_used_at"`
	}
	if err := json.Unmarshal(envelope.Data, &items); err != nil {
		t.Fatalf("decode list items: %v", err)
	}
	found := false
	for _, item := range items {
		if item.Path == alpha {
			found = true
			if item.LastUsedAt != 123 {
				t.Fatalf("expected migrated last_used_at 123, got %#v", item)
			}
		}
	}
	if !found {
		t.Fatalf("expected migrated alpha entry in %#v", items)
	}

	if _, err := os.Stat(filepath.Join(stateDir, "state-v2.json")); err != nil {
		t.Fatalf("expected state-v2.json after migration: %v", err)
	}
	after, err := os.ReadFile(v1Path)
	if err != nil {
		t.Fatalf("read v1 state: %v", err)
	}
	if string(after) != string(original) {
		t.Fatalf("expected v1 state to remain unchanged, got %s", string(after))
	}
}

func TestCLINewPathMetadataAndGCDryRunJSON(t *testing.T) {
	repo := newTestRepo(t)
	bin := buildCLI(t)
	stateHome := t.TempDir()

	newCmd := exec.CommandContext(context.Background(), bin, "new-path", "--json", "--label", "agent:claude-code", "--ttl", "24h", "beta")
	newCmd.Dir = repo.Root
	newCmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateHome)

	var newStdout, newStderr bytes.Buffer
	newCmd.Stdout = &newStdout
	newCmd.Stderr = &newStderr

	if err := newCmd.Run(); err != nil {
		t.Fatalf("new-path failed: %v\nstderr: %s", err, newStderr.String())
	}

	listCmd := exec.CommandContext(context.Background(), bin, "list", "--json")
	listCmd.Dir = repo.Root
	listCmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateHome)

	var listStdout, listStderr bytes.Buffer
	listCmd.Stdout = &listStdout
	listCmd.Stderr = &listStderr

	if err := listCmd.Run(); err != nil {
		t.Fatalf("list failed: %v\nstderr: %s", err, listStderr.String())
	}

	var listEnvelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(listStdout.Bytes(), &listEnvelope); err != nil {
		t.Fatalf("decode list envelope: %v\nraw=%s", err, listStdout.String())
	}
	var items []struct {
		Path  string `json:"path"`
		Label string `json:"label"`
		TTL   string `json:"ttl"`
	}
	if err := json.Unmarshal(listEnvelope.Data, &items); err != nil {
		t.Fatalf("decode list items: %v", err)
	}

	var betaPath string
	found := false
	for _, item := range items {
		if strings.HasSuffix(item.Path, "/.worktrees/beta") {
			found = true
			betaPath = item.Path
			if item.Label != "agent:claude-code" || item.TTL != "24h" {
				t.Fatalf("expected metadata for beta, got %#v", item)
			}
		}
	}
	if !found {
		t.Fatalf("expected beta in list output, got %#v", items)
	}

	stateV2Path := filepath.Join(stateHome, "ww", "state-v2.json")
	raw, err := os.ReadFile(stateV2Path)
	if err != nil {
		t.Fatalf("read state-v2: %v", err)
	}

	var disk struct {
		Repos map[string]struct {
			Worktrees map[string]map[string]any `json:"worktrees"`
		} `json:"repos"`
	}
	if err := json.Unmarshal(raw, &disk); err != nil {
		t.Fatalf("decode state-v2: %v", err)
	}
	repoKey := filepath.Join(repo.Root, ".git")
	record := disk.Repos[repoKey].Worktrees[betaPath]
	record["created_at"] = float64(1)
	disk.Repos[repoKey].Worktrees[betaPath] = record

	updated, err := json.MarshalIndent(disk, "", "  ")
	if err != nil {
		t.Fatalf("re-encode state-v2: %v", err)
	}
	updated = append(updated, '\n')
	if err := os.WriteFile(stateV2Path, updated, 0o644); err != nil {
		t.Fatalf("write updated state-v2: %v", err)
	}

	gcCmd := exec.CommandContext(context.Background(), bin, "gc", "--json", "--dry-run", "--ttl-expired")
	gcCmd.Dir = repo.Root
	gcCmd.Env = append(os.Environ(), "XDG_STATE_HOME="+stateHome)

	var gcStdout, gcStderr bytes.Buffer
	gcCmd.Stdout = &gcStdout
	gcCmd.Stderr = &gcStderr

	if err := gcCmd.Run(); err != nil {
		t.Fatalf("gc failed: %v\nstderr: %s", err, gcStderr.String())
	}

	var gcEnvelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(gcStdout.Bytes(), &gcEnvelope); err != nil {
		t.Fatalf("decode gc envelope: %v\nraw=%s", err, gcStdout.String())
	}
	var gcData struct {
		Summary struct {
			Matched int `json:"matched"`
			Removed int `json:"removed"`
		} `json:"summary"`
		Items []struct {
			Path   string `json:"path"`
			Action string `json:"action"`
		} `json:"items"`
	}
	if err := json.Unmarshal(gcEnvelope.Data, &gcData); err != nil {
		t.Fatalf("decode gc data: %v", err)
	}
	if gcData.Summary.Matched != 1 || gcData.Summary.Removed != 0 {
		t.Fatalf("unexpected gc summary: %#v", gcData.Summary)
	}
	if len(gcData.Items) != 1 || gcData.Items[0].Path != betaPath || gcData.Items[0].Action != "dry_run" {
		t.Fatalf("unexpected gc items: %#v", gcData.Items)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return root
}
