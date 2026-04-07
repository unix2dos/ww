package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupGitignore writes a .gitignore and commits it so git knows which files
// are ignored.
func setupGitignore(t *testing.T, repoRoot string, patterns ...string) {
	t.Helper()
	content := strings.Join(patterns, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(repoRoot, ".gitignore"), []byte(content), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	runGit(t, repoRoot, "add", ".gitignore")
	runGit(t, repoRoot, "commit", "-m", "add .gitignore")
}

func runNewPath(t *testing.T, bin, dir, branch string, extraArgs ...string) (stdout, stderr string) {
	t.Helper()
	args := append([]string{"new-path"}, extraArgs...)
	args = append(args, branch)
	cmd := exec.CommandContext(context.Background(), bin, args...)
	cmd.Dir = dir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("new-path %s failed: %v\nstderr: %s", branch, err, errBuf.String())
	}
	return strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String())
}

func TestSyncCopiesIgnoredFilesIntoNewWorktree(t *testing.T) {
	repo := newTestRepo(t)
	bin := buildCLI(t)

	setupGitignore(t, repo.Root, ".env", "local.cfg")

	// Place ignored files in the main worktree.
	if err := os.WriteFile(filepath.Join(repo.Root, ".env"), []byte("SECRET=abc\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo.Root, "local.cfg"), []byte("key=val\n"), 0o644); err != nil {
		t.Fatalf("write local.cfg: %v", err)
	}

	newPath, stderr := runNewPath(t, bin, repo.Root, "feat-sync")

	// Both files must exist in the new worktree.
	if _, err := os.Stat(filepath.Join(newPath, ".env")); err != nil {
		t.Errorf(".env not copied into new worktree: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newPath, "local.cfg")); err != nil {
		t.Errorf("local.cfg not copied into new worktree: %v", err)
	}

	// Sync summary must appear on stderr.
	if !strings.Contains(stderr, "synced") {
		t.Errorf("expected 'synced' in stderr, got %q", stderr)
	}
}

func TestSyncPreservesSubdirectoryStructure(t *testing.T) {
	repo := newTestRepo(t)
	bin := buildCLI(t)

	setupGitignore(t, repo.Root, "config/")

	if err := os.MkdirAll(filepath.Join(repo.Root, "config"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo.Root, "config", "local.json"), []byte(`{"k":"v"}`), 0o644); err != nil {
		t.Fatalf("write config/local.json: %v", err)
	}

	newPath, _ := runNewPath(t, bin, repo.Root, "feat-subdir")

	if _, err := os.Stat(filepath.Join(newPath, "config", "local.json")); err != nil {
		t.Errorf("config/local.json not copied: %v", err)
	}
}

func TestSyncSkipsBlacklistedDirectories(t *testing.T) {
	repo := newTestRepo(t)
	bin := buildCLI(t)

	setupGitignore(t, repo.Root, "node_modules/", ".env")

	if err := os.MkdirAll(filepath.Join(repo.Root, "node_modules", "lodash"), 0o755); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo.Root, "node_modules", "lodash", "index.js"), []byte("module.exports={}"), 0o644); err != nil {
		t.Fatalf("write node_modules file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo.Root, ".env"), []byte("X=1\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	newPath, _ := runNewPath(t, bin, repo.Root, "feat-blacklist")

	// .env must be copied.
	if _, err := os.Stat(filepath.Join(newPath, ".env")); err != nil {
		t.Errorf(".env should have been copied: %v", err)
	}
	// node_modules must NOT be copied.
	if _, err := os.Stat(filepath.Join(newPath, "node_modules")); err == nil {
		t.Errorf("node_modules should not have been copied")
	}
}

func TestSyncNoSyncFlagDisablesSync(t *testing.T) {
	repo := newTestRepo(t)
	bin := buildCLI(t)

	setupGitignore(t, repo.Root, ".env")

	if err := os.WriteFile(filepath.Join(repo.Root, ".env"), []byte("X=1\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	newPath, stderr := runNewPath(t, bin, repo.Root, "feat-nosync", "--no-sync")

	if _, err := os.Stat(filepath.Join(newPath, ".env")); err == nil {
		t.Errorf(".env should not have been copied with --no-sync")
	}
	if strings.Contains(stderr, "synced") {
		t.Errorf("expected no sync output with --no-sync, got %q", stderr)
	}
}

func TestSyncDryRunDoesNotWriteFiles(t *testing.T) {
	repo := newTestRepo(t)
	bin := buildCLI(t)

	setupGitignore(t, repo.Root, ".env")

	if err := os.WriteFile(filepath.Join(repo.Root, ".env"), []byte("X=1\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	newPath, stderr := runNewPath(t, bin, repo.Root, "feat-dryrun", "--sync-dry-run")

	// File must NOT be written.
	if _, err := os.Stat(filepath.Join(newPath, ".env")); err == nil {
		t.Errorf(".env should not be written in dry-run mode")
	}
	// But dry-run output must appear.
	if !strings.Contains(stderr, "[dry-run]") {
		t.Errorf("expected '[dry-run]' in stderr, got %q", stderr)
	}
}

func TestSyncNoIgnoredFilesIsQuiet(t *testing.T) {
	repo := newTestRepo(t)
	bin := buildCLI(t)

	// No .gitignore, no ignored files.
	_, stderr := runNewPath(t, bin, repo.Root, "feat-quiet")

	if strings.Contains(stderr, "synced") {
		t.Errorf("expected no sync output when nothing to sync, got %q", stderr)
	}
}

func TestSyncConfigDisabledViaEnv(t *testing.T) {
	repo := newTestRepo(t)
	bin := buildCLI(t)
	cfgDir := t.TempDir()

	setupGitignore(t, repo.Root, ".env")
	if err := os.WriteFile(filepath.Join(repo.Root, ".env"), []byte("X=1\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	// Write a config that disables sync.
	wwCfgDir := filepath.Join(cfgDir, "ww")
	if err := os.MkdirAll(wwCfgDir, 0o755); err != nil {
		t.Fatalf("mkdir ww config dir: %v", err)
	}
	cfgJSON := `{"version":1,"sync":{"enabled":false}}`
	if err := os.WriteFile(filepath.Join(wwCfgDir, "config.json"), []byte(cfgJSON), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	args := []string{"new-path", "feat-cfg-disabled"}
	cmd := exec.CommandContext(context.Background(), bin, args...)
	cmd.Dir = repo.Root
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+cfgDir)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("new-path failed: %v\nstderr: %s", err, errBuf.String())
	}

	newPath := strings.TrimSpace(outBuf.String())
	if _, err := os.Stat(filepath.Join(newPath, ".env")); err == nil {
		t.Errorf(".env should not be copied when sync disabled via config")
	}
}
