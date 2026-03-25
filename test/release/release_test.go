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
		filepath.Join(outDir, "ww.rb"),
	}
	for _, archive := range expected {
		if _, err := os.Stat(archive); err != nil {
			t.Fatalf("expected archive %s: %v", archive, err)
		}
		if strings.HasSuffix(archive, ".sh") || strings.HasSuffix(archive, ".rb") {
			continue
		}
		contents := listTar(t, archive)
		for _, required := range []string{
			"README.md",
			"install.sh",
			"uninstall.sh",
			"shell/ww.sh",
			"bin/ww-helper",
		} {
			if !strings.Contains(contents, required) {
				t.Fatalf("expected %s in %s, got:\n%s", required, archive, contents)
			}
		}
	}

	checksums := mustReadFile(t, filepath.Join(outDir, "checksums.txt"))
	formula := mustReadFile(t, filepath.Join(outDir, "ww.rb"))
	for _, snippet := range []string{
		"class Ww < Formula",
		`bin.install "bin/ww-helper"`,
		`libexec.install "shell/ww.sh"`,
		`source "#{opt_libexec}/ww.sh"`,
		`export WW_HELPER_BIN="#{opt_bin}/ww-helper"`,
		`assert_match "Usage: ww-helper"`,
		"https://github.com/unix2dos/ww/releases/download/v0.1.0/ww-v0.1.0-darwin-arm64.tar.gz",
		"https://github.com/unix2dos/ww/releases/download/v0.1.0/ww-v0.1.0-linux-amd64.tar.gz",
	} {
		if !strings.Contains(formula, snippet) {
			t.Fatalf("expected generated formula to contain %q, got:\n%s", snippet, formula)
		}
	}

	for _, checksum := range []string{
		checksumForArchive(t, checksums, "ww-v0.1.0-darwin-arm64.tar.gz"),
		checksumForArchive(t, checksums, "ww-v0.1.0-linux-amd64.tar.gz"),
	} {
		if !strings.Contains(formula, checksum) {
			t.Fatalf("expected generated formula to contain checksum %q, got:\n%s", checksum, formula)
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

func mustReadFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func checksumForArchive(t *testing.T, checksums string, archive string) string {
	t.Helper()

	for _, line := range strings.Split(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if fields[1] == archive {
			return fields[0]
		}
	}

	t.Fatalf("checksum for %s not found in:\n%s", archive, checksums)
	return ""
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
