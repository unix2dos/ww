package install_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallIsIdempotentAndBuildsBinary(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	firstInstall := runInstall(t, home)
	runInstall(t, home)

	if !strings.Contains(firstInstall, "Use `ww` to switch") {
		t.Fatalf("expected install output to mention ww, got %q", firstInstall)
	}
	if !strings.Contains(firstInstall, "Use `ww --fzf`") {
		t.Fatalf("expected install output to mention ww --fzf, got %q", firstInstall)
	}

	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc file: %v", err)
	}

	if strings.Count(string(data), "ww shell wrapper begin") != 1 {
		t.Fatalf("expected one managed block, got %q", string(data))
	}
	if !strings.Contains(string(data), "ww()") {
		t.Fatalf("expected ww shell function, got %q", string(data))
	}
	if !strings.Contains(string(data), "command ww") {
		t.Fatalf("expected ww shell function to call command ww, got %q", string(data))
	}

	binPath := filepath.Join(home, ".local", "bin", "ww")
	if info, err := os.Stat(binPath); err != nil {
		t.Fatalf("expected built binary at %s: %v", binPath, err)
	} else if info.Mode()&0o111 == 0 {
		t.Fatalf("expected built binary to be executable, mode=%v", info.Mode())
	}

}

func TestInstallSupportsCustomRcFileAndBinDir(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".config", "wt-test.rc")
	binDir := filepath.Join(home, ".bin")

	runInstall(t, home, "--shell", "bash", "--rc-file", rcPath, "--bin-dir", binDir)

	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc file: %v", err)
	}

	binPath := filepath.Join(binDir, "ww")
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("expected custom binary at %s: %v", binPath, err)
	}
	if !strings.Contains(string(data), "ww()") {
		t.Fatalf("expected ww shell function in custom rc file, got %q", string(data))
	}
	if !strings.Contains(string(data), "command ww") {
		t.Fatalf("expected ww shell function to call command ww, got %q", string(data))
	}
}

func TestUninstallRemovesManagedBlockAndBinary(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)
	runUninstall(t, home)

	if _, err := os.Stat(filepath.Join(home, ".local", "bin", "ww")); !os.IsNotExist(err) {
		t.Fatalf("expected binary to be removed, got err=%v", err)
	}
	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc file: %v", err)
	}
	if strings.Contains(string(data), "ww shell wrapper begin") {
		t.Fatalf("expected managed block removed, got %q", string(data))
	}
}

func TestWwChangesDirectoryOnSuccess(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	target := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	fakeBinDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(fakeBinDir, 0o755); err != nil {
		t.Fatalf("mkdir fake bin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fakeBinDir, "ww"), []byte(fmt.Sprintf("#!/usr/bin/env bash\nprintf '%%s\\n' %q\n", target)), 0o755); err != nil {
		t.Fatalf("write fake ww: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		export PATH=%q:%q:$PATH
		ww >/dev/null
		pwd
	`, origin, rcPath, fakeBinDir, filepath.Join(home, ".local", "bin")))

	if got := strings.TrimSpace(out); got != target {
		t.Fatalf("expected shell to cd to %q, got %q", target, got)
	}
}

func TestWwLeavesDirectoryOnFailure(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	fakeBinDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(fakeBinDir, 0o755); err != nil {
		t.Fatalf("mkdir fake bin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fakeBinDir, "ww"), []byte("#!/usr/bin/env bash\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write fake ww: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		export PATH=%q:%q:$PATH
		if ww >/dev/null 2>&1; then
			echo unexpected-success
			exit 1
		fi
		pwd
	`, origin, rcPath, fakeBinDir, filepath.Join(home, ".local", "bin")))

	if got := strings.TrimSpace(out); got != origin {
		t.Fatalf("expected shell to stay in %q, got %q", origin, got)
	}
}

func TestWwLeavesDirectoryOnEmptyOutput(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	fakeBinDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(fakeBinDir, 0o755); err != nil {
		t.Fatalf("mkdir fake bin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fakeBinDir, "ww"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake ww: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		export PATH=%q:%q:$PATH
		if ww >/dev/null 2>&1; then
			echo unexpected-success
			exit 1
		fi
		pwd
	`, origin, rcPath, fakeBinDir, filepath.Join(home, ".local", "bin")))

	if got := strings.TrimSpace(out); got != origin {
		t.Fatalf("expected shell to stay in %q, got %q", origin, got)
	}
}

func runInstall(t *testing.T, home string, args ...string) string {
	t.Helper()

	cmdArgs := append([]string{"install.sh"}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = projectRoot(t)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"SHELL=/bin/zsh",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out)
	}
	t.Fatalf("install failed: %v\n%s", err, out)
	return ""
}

func runUninstall(t *testing.T, home string, args ...string) {
	t.Helper()

	cmdArgs := append([]string{"uninstall.sh"}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = projectRoot(t)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"SHELL=/bin/zsh",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return
	}
	t.Fatalf("uninstall failed: %v\n%s", err, out)
}

func runShell(t *testing.T, workdir, script string) string {
	t.Helper()

	cmd := exec.Command("bash", "-lc", script)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shell script failed: %v\n%s", err, out)
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
