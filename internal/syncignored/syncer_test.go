package syncignored

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// fakeRunner returns a canned ls-files response for any invocation.
type fakeRunner struct {
	stdout string
	stderr string
	err    error
	calls  [][]string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	return []byte(f.stdout), []byte(f.stderr), f.err
}

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data := make([]byte, size)
	for i := range data {
		data[i] = 'x'
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestSyncDisabledIsNoop(t *testing.T) {
	res, err := Sync(context.Background(), &fakeRunner{}, "/main", "/target", Options{Enabled: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Copied) != 0 || len(res.Skipped) != 0 {
		t.Fatalf("expected empty result, got %+v", res)
	}
}

func TestSyncRejectsEmptyPaths(t *testing.T) {
	_, err := Sync(context.Background(), &fakeRunner{}, "", "/target", Options{Enabled: true})
	if err == nil {
		t.Fatal("expected error for empty mainRoot")
	}
}

func TestSyncSameMainAndTargetIsNoop(t *testing.T) {
	res, err := Sync(context.Background(), &fakeRunner{}, "/same", "/same", Options{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Copied) != 0 {
		t.Fatalf("expected no copies, got %v", res.Copied)
	}
}

func TestSyncCopiesIgnoredFiles(t *testing.T) {
	main := t.TempDir()
	target := t.TempDir()

	writeFile(t, filepath.Join(main, ".env"), 50)
	writeFile(t, filepath.Join(main, "config", "local.json"), 100)

	runner := &fakeRunner{stdout: ".env\nconfig/local.json\n"}
	res, err := Sync(context.Background(), runner, main, target, Options{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Strings(res.Copied)
	want := []string{".env", "config/local.json"}
	if strings.Join(res.Copied, ",") != strings.Join(want, ",") {
		t.Fatalf("copied = %v, want %v", res.Copied, want)
	}

	if _, err := os.Stat(filepath.Join(target, ".env")); err != nil {
		t.Fatalf(".env not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "config", "local.json")); err != nil {
		t.Fatalf("config/local.json not copied: %v", err)
	}

	// Verify the git command shape.
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 git call, got %d", len(runner.calls))
	}
	got := runner.calls[0]
	wantCmd := []string{"git", "-C", main, "ls-files", "--others", "--ignored", "--exclude-standard"}
	if strings.Join(got, " ") != strings.Join(wantCmd, " ") {
		t.Fatalf("git call = %v, want %v", got, wantCmd)
	}
}

func TestSyncSkipsBlacklistedSegments(t *testing.T) {
	main := t.TempDir()
	target := t.TempDir()

	writeFile(t, filepath.Join(main, ".env"), 10)
	writeFile(t, filepath.Join(main, "node_modules", "foo", "bar.js"), 10)
	writeFile(t, filepath.Join(main, "src", ".DS_Store"), 10)

	runner := &fakeRunner{stdout: ".env\nnode_modules/foo/bar.js\nsrc/.DS_Store\n"}
	res, err := Sync(context.Background(), runner, main, target, Options{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Copied) != 1 || res.Copied[0] != ".env" {
		t.Fatalf("expected only .env copied, got %v", res.Copied)
	}
	if len(res.Skipped) != 2 {
		t.Fatalf("expected 2 skipped, got %v", res.Skipped)
	}
	for _, s := range res.Skipped {
		if s.Reason != SkipBlacklisted {
			t.Errorf("expected SkipBlacklisted for %s, got %s", s.Path, s.Reason)
		}
	}
}

func TestSyncSkipsLargeFiles(t *testing.T) {
	main := t.TempDir()
	target := t.TempDir()

	writeFile(t, filepath.Join(main, "small.bin"), 100)
	writeFile(t, filepath.Join(main, "big.bin"), 2048)

	runner := &fakeRunner{stdout: "small.bin\nbig.bin\n"}
	res, err := Sync(context.Background(), runner, main, target, Options{
		Enabled:     true,
		MaxFileSize: 1024,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Copied) != 1 || res.Copied[0] != "small.bin" {
		t.Fatalf("expected only small.bin copied, got %v", res.Copied)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].Path != "big.bin" || res.Skipped[0].Reason != SkipTooLarge {
		t.Fatalf("expected big.bin skipped as too_large, got %+v", res.Skipped)
	}
	if res.Skipped[0].Size != 2048 {
		t.Errorf("expected Size=2048, got %d", res.Skipped[0].Size)
	}

	if _, err := os.Stat(filepath.Join(target, "big.bin")); !os.IsNotExist(err) {
		t.Errorf("big.bin should not have been copied; stat err = %v", err)
	}
}

func TestSyncDryRunDoesNotWriteFiles(t *testing.T) {
	main := t.TempDir()
	target := t.TempDir()

	writeFile(t, filepath.Join(main, ".env"), 10)

	runner := &fakeRunner{stdout: ".env\n"}
	res, err := Sync(context.Background(), runner, main, target, Options{
		Enabled: true,
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.DryRun {
		t.Error("expected res.DryRun=true")
	}
	if len(res.Copied) != 1 || res.Copied[0] != ".env" {
		t.Fatalf("expected .env in Copied (planned), got %v", res.Copied)
	}
	if _, err := os.Stat(filepath.Join(target, ".env")); !os.IsNotExist(err) {
		t.Errorf(".env should not have been written in dry-run; stat err = %v", err)
	}
}

func TestSyncPropagatesGitError(t *testing.T) {
	runner := &fakeRunner{
		stderr: "fatal: not a git repository\n",
		err:    &exitErr{msg: "exit 128"},
	}
	_, err := Sync(context.Background(), runner, t.TempDir(), t.TempDir(), Options{Enabled: true})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "git ls-files") {
		t.Errorf("error should mention git ls-files, got %v", err)
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("error should include stderr, got %v", err)
	}
}

func TestSyncCustomBlacklistOverridesDefaults(t *testing.T) {
	main := t.TempDir()
	target := t.TempDir()

	// node_modules is in the *default* blacklist but our override doesn't list
	// it, so it must be copied here.
	writeFile(t, filepath.Join(main, "node_modules", "ok.txt"), 10)
	writeFile(t, filepath.Join(main, "secret", "data"), 10)

	runner := &fakeRunner{stdout: "node_modules/ok.txt\nsecret/data\n"}
	res, err := Sync(context.Background(), runner, main, target, Options{
		Enabled:   true,
		Blacklist: []string{"secret/"}, // trailing slash should be tolerated
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Copied) != 1 || res.Copied[0] != "node_modules/ok.txt" {
		t.Fatalf("expected node_modules/ok.txt copied, got %v", res.Copied)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].Path != "secret/data" {
		t.Fatalf("expected secret/data skipped, got %+v", res.Skipped)
	}
}

func TestSyncPreservesFilePermissions(t *testing.T) {
	main := t.TempDir()
	target := t.TempDir()

	src := filepath.Join(main, "script.sh")
	writeFile(t, src, 10)
	if err := os.Chmod(src, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	runner := &fakeRunner{stdout: "script.sh\n"}
	if _, err := Sync(context.Background(), runner, main, target, Options{Enabled: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(filepath.Join(target, "script.sh"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("expected perm 0755, got %o", info.Mode().Perm())
	}
}

// exitErr is a minimal error type that lets the fake runner mimic a non-zero exit.
type exitErr struct{ msg string }

func (e *exitErr) Error() string { return e.msg }
