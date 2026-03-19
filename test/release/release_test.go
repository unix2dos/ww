package release_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleaseScriptBuildsExpectedArchives(t *testing.T) {
	outDir := t.TempDir()
	repoRoot := projectRoot(t)

	cmd := exec.Command("bash", "scripts/release.sh", "v0.1.0")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"WT_RELEASE_OUT_DIR="+outDir,
		"WT_RELEASE_TARGETS=darwin/arm64 linux/amd64",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("release script failed: %v\n%s", err, output)
	}

	expected := []string{
		filepath.Join(outDir, "ww-v0.1.0-darwin-arm64.tar.gz"),
		filepath.Join(outDir, "ww-v0.1.0-linux-amd64.tar.gz"),
		filepath.Join(outDir, "install-release.sh"),
	}
	for _, archive := range expected {
		if _, err := os.Stat(archive); err != nil {
			t.Fatalf("expected archive %s: %v", archive, err)
		}
		if strings.HasSuffix(archive, ".sh") {
			continue
		}
		contents := listTar(t, archive)
		for _, required := range []string{
			"README.md",
			"install.sh",
			"uninstall.sh",
			"shell/cwt.sh",
			"bin/ww",
		} {
			if !strings.Contains(contents, required) {
				t.Fatalf("expected %s in %s, got:\n%s", required, archive, contents)
			}
		}
	}
}

func listTar(t *testing.T, archive string) string {
	t.Helper()

	cmd := exec.Command("tar", "-tzf", archive)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tar list failed: %v\n%s", err, out)
	}
	return string(out)
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
