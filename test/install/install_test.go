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
	if !strings.Contains(firstInstall, "fzf when available") {
		t.Fatalf("expected install output to mention auto selector routing, got %q", firstInstall)
	}

	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc file: %v", err)
	}

	if strings.Count(string(data), "ww shell wrapper begin") != 1 {
		t.Fatalf("expected one managed block, got %q", string(data))
	}
	if !strings.Contains(string(data), `source "`) || !strings.Contains(string(data), filepath.Join(home, ".local", "bin", "ww.sh")) {
		t.Fatalf("expected rc to source ww.sh, got %q", string(data))
	}
	if !strings.Contains(string(data), filepath.Join(home, ".local", "bin", "ww-helper")) {
		t.Fatalf("expected rc to point at ww-helper, got %q", string(data))
	}

	binPath := filepath.Join(home, ".local", "bin", "ww-helper")
	if info, err := os.Stat(binPath); err != nil {
		t.Fatalf("expected built binary at %s: %v", binPath, err)
	} else if info.Mode()&0o111 == 0 {
		t.Fatalf("expected built binary to be executable, mode=%v", info.Mode())
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "bin", "ww.sh")); err != nil {
		t.Fatalf("expected ww shell library to exist: %v", err)
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

	binPath := filepath.Join(binDir, "ww-helper")
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("expected custom binary at %s: %v", binPath, err)
	}
	if !strings.Contains(string(data), filepath.Join(binDir, "ww.sh")) {
		t.Fatalf("expected custom rc to source ww.sh, got %q", string(data))
	}
	if !strings.Contains(string(data), filepath.Join(binDir, "ww-helper")) {
		t.Fatalf("expected custom rc to point at ww-helper, got %q", string(data))
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

	if _, err := os.Stat(filepath.Join(home, ".local", "bin", "ww-helper")); !os.IsNotExist(err) {
		t.Fatalf("expected helper binary to be removed, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "bin", "ww.sh")); !os.IsNotExist(err) {
		t.Fatalf("expected ww shell library to be removed, got err=%v", err)
	}
	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc file: %v", err)
	}
	if strings.Contains(string(data), "ww shell wrapper begin") {
		t.Fatalf("expected managed block removed, got %q", string(data))
	}
}

func TestUninstallMigratesOldWtState(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte("# wt shell wrapper begin\nold\n# wt shell wrapper end\n"), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}
	oldBinary := filepath.Join(home, ".local", "bin", "wt")
	if err := os.MkdirAll(filepath.Dir(oldBinary), 0o755); err != nil {
		t.Fatalf("mkdir old binary dir: %v", err)
	}
	if err := os.WriteFile(oldBinary, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write old binary: %v", err)
	}

	runUninstall(t, home)

	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc file: %v", err)
	}
	if strings.Contains(string(data), "wt shell wrapper begin") {
		t.Fatalf("expected old wt managed block removed, got %q", string(data))
	}
	if _, err := os.Stat(oldBinary); !os.IsNotExist(err) {
		t.Fatalf("expected old wt binary removed, got err=%v", err)
	}
}

func TestUninstallMigratesOldWtStateAcrossCommonRcFiles(t *testing.T) {
	home := t.TempDir()
	zshrc := filepath.Join(home, ".zshrc")
	bashrc := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(zshrc, []byte("# wt shell wrapper begin\nold-zsh\n# wt shell wrapper end\n"), 0o644); err != nil {
		t.Fatalf("write zsh rc file: %v", err)
	}
	if err := os.WriteFile(bashrc, []byte("# wt shell wrapper begin\nold-bash\n# wt shell wrapper end\n"), 0o644); err != nil {
		t.Fatalf("write bash rc file: %v", err)
	}

	runUninstall(t, home, "--shell", "zsh")

	for _, rcPath := range []string{zshrc, bashrc} {
		data, err := os.ReadFile(rcPath)
		if err != nil {
			t.Fatalf("read rc file %s: %v", rcPath, err)
		}
		if strings.Contains(string(data), "wt shell wrapper begin") {
			t.Fatalf("expected old wt block removed from %s, got %q", rcPath, string(data))
		}
	}
}

func TestInstallMigratesOldWtStateAcrossCommonRcFiles(t *testing.T) {
	home := t.TempDir()
	zshrc := filepath.Join(home, ".zshrc")
	bashrc := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(zshrc, []byte("# wt shell wrapper begin\nold-zsh\n# wt shell wrapper end\n"), 0o644); err != nil {
		t.Fatalf("write zsh rc file: %v", err)
	}
	if err := os.WriteFile(bashrc, []byte("# wt shell wrapper begin\nold-bash\n# wt shell wrapper end\n"), 0o644); err != nil {
		t.Fatalf("write bash rc file: %v", err)
	}

	runInstall(t, home, "--shell", "zsh")

	zshData, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf("read zsh rc file: %v", err)
	}
	if strings.Contains(string(zshData), "wt shell wrapper begin") {
		t.Fatalf("expected old wt block removed from zsh rc, got %q", string(zshData))
	}
	if !strings.Contains(string(zshData), "ww shell wrapper begin") {
		t.Fatalf("expected ww block in zsh rc, got %q", string(zshData))
	}

	bashData, err := os.ReadFile(bashrc)
	if err != nil {
		t.Fatalf("read bash rc file: %v", err)
	}
	if strings.Contains(string(bashData), "wt shell wrapper begin") {
		t.Fatalf("expected old wt block removed from bash rc, got %q", string(bashData))
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
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), fmt.Sprintf("#!/usr/bin/env bash\n[ \"$1\" = \"switch-path\" ] || exit 9\nprintf '%%s\\n' %q\n", target)); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww >/dev/null
		pwd
	`, origin, rcPath))

	if got := strings.TrimSpace(out); got != target {
		t.Fatalf("expected shell to cd to %q, got %q", target, got)
	}
}

func TestWwHelpDoesNotCd(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), "#!/usr/bin/env bash\nprintf 'helper-called:%s\\n' \"$1\"\nexit 9\n"); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww --help
		printf 'status1:%%d\n' $?
		ww help
		printf 'status2:%%d\n' $?
		pwd
	`, origin, rcPath))

	if !strings.Contains(out, "Usage:\n  ww [switch] [<index>|<name>]") {
		t.Fatalf("expected help output, got %q", out)
	}
	if !strings.Contains(out, "\nCommands:\n") {
		t.Fatalf("expected Commands section in help output, got %q", out)
	}
	if !strings.Contains(out, "\nExamples:\n") {
		t.Fatalf("expected Examples section in help output, got %q", out)
	}
	if strings.Contains(out, "ww-helper") {
		t.Fatalf("expected ww help output to avoid internal helper naming, got %q", out)
	}
	if strings.Contains(out, "helper-called:") {
		t.Fatalf("expected shell help to avoid calling ww-helper --help, got %q", out)
	}
	if !strings.Contains(out, "status1:0") || !strings.Contains(out, "status2:0") {
		t.Fatalf("expected both help paths to succeed, got %q", out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if got := lines[len(lines)-1]; got != origin {
		t.Fatalf("expected shell to stay in %q, got %q", origin, got)
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
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), "#!/usr/bin/env bash\nexit 1\n"); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		if ww >/dev/null 2>&1; then
			echo unexpected-success
			exit 1
		fi
		pwd
	`, origin, rcPath))

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
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), "#!/usr/bin/env bash\n[ \"$1\" = \"switch-path\" ] || exit 9\nexit 0\n"); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww >/dev/null 2>&1
		printf 'status:%%d\n' $?
		pwd
	`, origin, rcPath))

	if !strings.Contains(out, "status:0") {
		t.Fatalf("expected successful no-op status, got %q", out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if got := lines[len(lines)-1]; got != origin {
		t.Fatalf("expected shell to stay in %q, got %q", origin, got)
	}
}

func TestWwSwitchByNameChangesDirectoryOnSuccess(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	target := filepath.Join(t.TempDir(), "alpha")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), fmt.Sprintf("#!/usr/bin/env bash\n[ \"$1\" = \"switch-path\" ] || exit 9\n[ \"$2\" = \"alpha\" ] || exit 8\nprintf '%%s\\n' %q\n", target)); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww switch alpha >/dev/null
		pwd
	`, origin, rcPath))

	if got := strings.TrimSpace(out); got != target {
		t.Fatalf("expected shell to cd to %q, got %q", target, got)
	}
}

func TestWwNewChangesDirectoryOnSuccess(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	target := filepath.Join(t.TempDir(), "feature-x")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), fmt.Sprintf("#!/usr/bin/env bash\n[ \"$1\" = \"new-path\" ] || exit 9\n[ \"$2\" = \"feature-x\" ] || exit 8\nprintf '%%s\\n' %q\n", target)); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww new feature-x >/dev/null
		pwd
	`, origin, rcPath))

	if got := strings.TrimSpace(out); got != target {
		t.Fatalf("expected shell to cd to %q, got %q", target, got)
	}
}

func TestWwNewRejectsHelperOnlyLabelFlag(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	marker := filepath.Join(home, "helper-invoked")
	script := fmt.Sprintf("#!/usr/bin/env bash\ntouch %q\nexit 99\n", marker)
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), script); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out, err := runShellResult(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww new --label task:demo feature-x
	`, origin, rcPath))

	if err == nil {
		t.Fatalf("expected ww new --label to fail")
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("expected shell guard to stop before ww-helper, marker err=%v", statErr)
	}
	if !strings.Contains(out, "ww-helper new-path") {
		t.Fatalf("expected migration guidance, got %q", out)
	}
}

func TestWwNewRejectsHelperOnlyTTLFlag(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	marker := filepath.Join(home, "helper-invoked")
	script := fmt.Sprintf("#!/usr/bin/env bash\ntouch %q\nexit 99\n", marker)
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), script); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out, err := runShellResult(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww new --ttl 24h feature-x
	`, origin, rcPath))

	if err == nil {
		t.Fatalf("expected ww new --ttl to fail")
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("expected shell guard to stop before ww-helper, marker err=%v", statErr)
	}
	if !strings.Contains(out, "ww-helper new-path") {
		t.Fatalf("expected migration guidance, got %q", out)
	}
}

func TestWwGCPrintsCleanupGuidance(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	marker := filepath.Join(home, "helper-invoked")
	script := fmt.Sprintf("#!/usr/bin/env bash\ntouch %q\nexit 99\n", marker)
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), script); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out, err := runShellResult(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww gc
	`, origin, rcPath))

	if err == nil {
		t.Fatalf("expected ww gc to fail with guidance")
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("expected shell guidance before ww-helper, marker err=%v", statErr)
	}
	if !strings.Contains(out, "ww rm --cleanup") {
		t.Fatalf("expected cleanup guidance, got %q", out)
	}
}

func TestWwListPrintsOutputWithoutChangingDirectory(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	listOutput := "[1] * main /repo"
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), fmt.Sprintf("#!/usr/bin/env bash\n[ \"$1\" = \"list\" ] || exit 9\nprintf '%%s\\n' %q\n", listOutput)); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww list
		pwd
	`, origin, rcPath))

	if !strings.Contains(out, listOutput) {
		t.Fatalf("expected list output, got %q", out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if got := lines[len(lines)-1]; got != origin {
		t.Fatalf("expected shell to stay in %q, got %q", origin, got)
	}
}

func TestWwListJSONPrintsOutputWithoutChangingDirectory(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	listOutput := `{"ok":true,"command":"list","data":[{"path":"/repo","branch":"main"}]}`
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), fmt.Sprintf("#!/usr/bin/env bash\n[ \"$1\" = \"list\" ] || exit 9\n[ \"$2\" = \"--json\" ] || exit 8\nprintf '%%s\\n' %q\n", listOutput)); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww list --json
		pwd
	`, origin, rcPath))

	if !strings.Contains(out, listOutput) {
		t.Fatalf("expected list json output, got %q", out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if got := lines[len(lines)-1]; got != origin {
		t.Fatalf("expected shell to stay in %q, got %q", origin, got)
	}
}

func TestWwNewJSONPassesThroughWithoutChangingDirectory(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	jsonOutput := `{"ok":true,"command":"new-path","data":{"worktree_path":"/repo/.worktrees/feature-x","branch":"feature-x"}}`
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), fmt.Sprintf("#!/usr/bin/env bash\n[ \"$1\" = \"new-path\" ] || exit 9\n[ \"$2\" = \"--json\" ] || exit 8\n[ \"$3\" = \"feature-x\" ] || exit 7\nprintf '%%s\\n' %q\n", jsonOutput)); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww new --json feature-x
		pwd
	`, origin, rcPath))

	if !strings.Contains(out, jsonOutput) {
		t.Fatalf("expected new json output, got %q", out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if got := lines[len(lines)-1]; got != origin {
		t.Fatalf("expected shell to stay in %q, got %q", origin, got)
	}
}

func TestWwSwitchJSONPassesThroughWithoutChangingDirectory(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	jsonOutput := `{"ok":true,"command":"switch-path","data":{"worktree_path":"/repo/.worktrees/alpha","branch":"alpha"}}`
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), fmt.Sprintf("#!/usr/bin/env bash\n[ \"$1\" = \"switch-path\" ] || exit 9\n[ \"$2\" = \"--json\" ] || exit 8\n[ \"$3\" = \"alpha\" ] || exit 7\nprintf '%%s\\n' %q\n", jsonOutput)); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww switch --json alpha
		pwd
	`, origin, rcPath))

	if !strings.Contains(out, jsonOutput) {
		t.Fatalf("expected switch json output, got %q", out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if got := lines[len(lines)-1]; got != origin {
		t.Fatalf("expected shell to stay in %q, got %q", origin, got)
	}
}

func TestWwRmPassThroughWithoutChangingDirectory(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write rc file: %v", err)
	}

	runInstall(t, home)

	origin := t.TempDir()
	script := "#!/usr/bin/env bash\ncase \"$1\" in\n  rm) printf 'removed worktree /repo/.worktrees/alpha\\n' ;;\n  *) exit 9 ;;\nesac\n"
	if err := writeExecutableScript(filepath.Join(home, ".local", "bin", "ww-helper"), script); err != nil {
		t.Fatalf("write fake ww-helper: %v", err)
	}

	out := runShell(t, home, fmt.Sprintf(`
		cd %q
		source %q
		ww rm alpha
		pwd
	`, origin, rcPath))

	if !strings.Contains(out, "removed worktree /repo/.worktrees/alpha") {
		t.Fatalf("expected rm output, got %q", out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if got := lines[len(lines)-1]; got != origin {
		t.Fatalf("expected shell to stay in %q, got %q", origin, got)
	}
}

func runInstall(t *testing.T, home string, args ...string) string {
	t.Helper()

	cmdArgs := append([]string{"install.sh"}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = projectRoot(t)
	goPath := filepath.Join(home, "go")
	goCache := filepath.Join(home, ".cache", "go-build")
	goModCache := filepath.Join(goPath, "pkg", "mod")
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"SHELL=/bin/zsh",
		"GOPATH="+goPath,
		"GOCACHE="+goCache,
		"GOMODCACHE="+goModCache,
		"GOFLAGS=-modcacherw",
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
	goPath := filepath.Join(home, "go")
	goCache := filepath.Join(home, ".cache", "go-build")
	goModCache := filepath.Join(goPath, "pkg", "mod")
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"SHELL=/bin/zsh",
		"GOPATH="+goPath,
		"GOCACHE="+goCache,
		"GOMODCACHE="+goModCache,
		"GOFLAGS=-modcacherw",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return
	}
	t.Fatalf("uninstall failed: %v\n%s", err, out)
}

func runShell(t *testing.T, workdir, script string) string {
	t.Helper()

	out, err := runShellResult(t, workdir, script)
	if err != nil {
		t.Fatalf("shell script failed: %v\n%s", err, out)
	}
	return out
}

func runShellResult(t *testing.T, workdir, script string) (string, error) {
	t.Helper()

	cmd := exec.Command("bash", "-lc", script)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeExecutableScript(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o755)
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
