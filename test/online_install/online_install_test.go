package onlineinstall_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallReleaseScriptInstallsFromTarballURL(t *testing.T) {
	repoRoot := projectRoot(t)
	outDir := t.TempDir()
	home := t.TempDir()
	rcPath := filepath.Join(home, ".bashrc")
	binDir := filepath.Join(home, ".bin")

	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	run(t, repoRoot, append(os.Environ(),
		"WT_RELEASE_OUT_DIR="+outDir,
		"WT_RELEASE_TARGETS="+goos+"/"+goarch,
	), "bash", "scripts/release.sh", "v9.9.9-test")

	tarball := filepath.Join(outDir, "ww-v9.9.9-test-"+goos+"-"+goarch+".tar.gz")
	if _, err := os.Stat(tarball); err != nil {
		t.Fatalf("expected tarball %s: %v", tarball, err)
	}

	run(t, repoRoot, append(os.Environ(),
		"HOME="+home,
		"WT_TARBALL_URL=file://"+tarball,
	), "bash", "scripts/install-release.sh", "--shell", "bash", "--rc-file", rcPath, "--bin-dir", binDir)

	if _, err := os.Stat(filepath.Join(binDir, "ww")); err != nil {
		t.Fatalf("expected installed ww binary: %v", err)
	}

	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc file: %v", err)
	}
	if !strings.Contains(string(data), "ww()") {
		t.Fatalf("expected ww shell function, got %q", string(data))
	}
	if !strings.Contains(string(data), filepath.Join(binDir, "ww")) {
		t.Fatalf("expected ww shell function to call installed binary, got %q", string(data))
	}
	if !strings.Contains(string(data), "ww shell wrapper begin") {
		t.Fatalf("expected managed block marker, got %q", string(data))
	}
}

func run(t *testing.T, dir string, env []string, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
