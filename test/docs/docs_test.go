package docs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPagesDemoContract(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))

	readme := mustReadFile(t, filepath.Join(root, "README.md"))
	if !strings.Contains(readme, "The worktree primitive your AI agents and you share.") {
		t.Fatalf("expected README to lead with the agent-primitive tagline")
	}
	if !strings.Contains(readme, "## Demo") {
		t.Fatalf("expected README demo section")
	}
	if !strings.Contains(readme, "brew tap unix2dos/ww https://github.com/unix2dos/ww") {
		t.Fatalf("expected README to document the Homebrew tap command")
	}
	if !strings.Contains(readme, "brew install ww") {
		t.Fatalf("expected README to document the Homebrew install command")
	}
	if !strings.Contains(readme, `ww-helper init zsh`) {
		t.Fatalf("expected README to document ww-helper init for Homebrew")
	}
	if !strings.Contains(readme, "https://unix2dos.github.io/ww/") {
		t.Fatalf("expected README to link to the GitHub Pages demo")
	}
	if !strings.Contains(readme, "docs/assets/ww-demo.svg") {
		t.Fatalf("expected README to embed the generated SVG demo preview")
	}
	if !strings.Contains(readme, "docs/reference.md") {
		t.Fatalf("expected README to hand off detailed docs to docs/reference.md")
	}
	if !strings.Contains(readme, "## For AI agents and orchestrators") {
		t.Fatalf("expected README to document the machine-readable entrypoint")
	}
	if !strings.Contains(readme, "docs/protocol.md") {
		t.Fatalf("expected README to link to the wire protocol spec")
	}
	if !strings.Contains(readme, "ww-helper list --json") {
		t.Fatalf("expected README to show ww-helper json usage")
	}
	if !strings.Contains(readme, "ww new feat-demo") {
		t.Fatalf("expected README to document simple worktree creation")
	}
	if strings.Contains(readme, "--cleanup") {
		t.Fatalf("expected README to stop referencing the removed --cleanup flag")
	}
	if strings.Contains(readme, "--non-interactive") {
		t.Fatalf("expected README to stop referencing the removed --non-interactive flag")
	}
	if strings.Contains(readme, "ww new feat-a --label agent:claude-code --ttl 24h") {
		t.Fatalf("expected README to keep metadata flags out of the human path")
	}
	if strings.Contains(readme, "ww gc --ttl-expired --dry-run") {
		t.Fatalf("expected README to stop documenting ww gc for humans")
	}
	if strings.Contains(readme, "### Install From Source") {
		t.Fatalf("expected README to stay in landing-page mode, not inline the full reference")
	}

	reference := mustReadFile(t, filepath.Join(root, "docs", "reference.md"))
	for _, snippet := range []string{
		"# ww Reference",
		"## Install",
		"### Homebrew Tap",
		`ww-helper init zsh`,
		`ww-helper init bash`,
		`eval "$("`,
		"Homebrew installs the helper and shell library, but leaves shell activation to you.",
		"## Usage",
		"## Release",
		"`ww help` or `ww --help` prints the command summary.",
		"### For AI Agents",
		"ww rm --cleanup",
		"ww-helper rm --json --non-interactive feat-a",
		"ww-helper new-path --json --label agent:claude-code --ttl 24h -m",
		"ww-helper gc --ttl-expired --idle 7d --dry-run --json",
		"#### Breaking Change",
		"[CURRENT]",
		"[DIRTY]",
		"┌───────┬",
		"Long `PATH` values are wrapped inside the `PATH` cell",
	} {
		if !strings.Contains(reference, snippet) {
			t.Fatalf("expected reference doc to contain %q", snippet)
		}
	}
	for _, forbidden := range []string{
		"ww new feat-a --label agent:claude-code --ttl 24h",
		"ww gc --ttl-expired --dry-run",
		"ww gc --idle 7d",
		"ww gc --merged",
		"ACTIVE*",
	} {
		if strings.Contains(reference, forbidden) {
			t.Fatalf("expected reference doc to stop teaching human-facing %q", forbidden)
		}
	}

	formula := mustReadFile(t, filepath.Join(root, "Formula", "ww.rb"))
	for _, snippet := range []string{
		"class Ww < Formula",
		`bin.install "bin/ww-helper"`,
		`libexec.install "shell/ww.sh"`,
		"def caveats",
		`eval "$("#{opt_bin}/ww-helper" init zsh)"`,
		`assert_match "Usage: ww-helper"`,
	} {
		if !strings.Contains(formula, snippet) {
			t.Fatalf("expected committed formula to contain %q", snippet)
		}
	}

	indexHTML := mustReadFile(t, filepath.Join(root, "docs", "index.html"))
	if !strings.Contains(indexHTML, "asciinema-player@3.15.1") {
		t.Fatalf("expected Pages demo to pin asciinema-player 3.15.1")
	}
	if !strings.Contains(indexHTML, "assets/ww-demo.cast") {
		t.Fatalf("expected Pages demo to load the local cast asset")
	}
	if !strings.Contains(indexHTML, "ww Demo") {
		t.Fatalf("expected Pages demo to show a visible title")
	}
	if !strings.Contains(indexHTML, "`fzf` switch, `ww list`, `ww new`, safe removal, cleanup review") {
		t.Fatalf("expected Pages demo copy to describe the workflow overview")
	}
	if !strings.Contains(indexHTML, "`ww-helper --json` tail for automation") {
		t.Fatalf("expected Pages demo copy to mention the automation tail")
	}
	if !strings.Contains(indexHTML, "speed: 0.6") {
		t.Fatalf("expected Pages demo to default to slower playback")
	}

	pagesWorkflow := mustReadFile(t, filepath.Join(root, ".github", "workflows", "pages.yml"))
	for _, snippet := range []string{
		"actions/configure-pages@v5",
		"actions/upload-pages-artifact@v4",
		"actions/deploy-pages@v4",
		"path: docs",
	} {
		if !strings.Contains(pagesWorkflow, snippet) {
			t.Fatalf("expected pages workflow to contain %q", snippet)
		}
	}

	generateScript := mustReadFile(t, filepath.Join(root, "scripts", "generate-demo.sh"))
	if !strings.Contains(generateScript, "svg-term-cli@2.1.1") {
		t.Fatalf("expected generator script to pin svg-term-cli 2.1.1")
	}
	if !strings.Contains(generateScript, "asciinema") {
		t.Fatalf("expected generator script to use asciinema")
	}
	if !strings.Contains(generateScript, "scripts/demo-fzf.sh") {
		t.Fatalf("expected generator script to install the deterministic demo fzf shim")
	}
	if !strings.Contains(generateScript, "WW_DEMO_KEYSTROKE_DELAY_MS") {
		t.Fatalf("expected generator script to expose demo pacing knobs")
	}

	expectScript := mustReadFile(t, filepath.Join(root, "scripts", "demo-record.exp"))
	if !strings.Contains(expectScript, "ww new") {
		t.Fatalf("expected expect demo script to exercise ww new")
	}
	if !strings.Contains(expectScript, "ww rm feat-demo") {
		t.Fatalf("expected expect demo script to exercise ww rm")
	}
	if !strings.Contains(expectScript, "send_nav_up") || !strings.Contains(expectScript, "send_nav_down") {
		t.Fatalf("expected expect demo script to drive visible picker navigation")
	}
	if strings.Contains(expectScript, "Use Up/Down") {
		t.Fatalf("expected expect demo script to stop driving the built-in selector")
	}

	cast := mustReadFile(t, filepath.Join(root, "docs", "assets", "ww-demo.cast"))
	if !strings.Contains(cast, "\"version\":2") {
		t.Fatalf("expected local asciinema cast asset")
	}
	if !strings.Contains(cast, "Select a worktree>") {
		t.Fatalf("expected demo cast to show the fzf prompt")
	}
	if !strings.Contains(cast, "ww rm feat-demo") {
		t.Fatalf("expected demo cast to cover the removal flow")
	}
	if !strings.Contains(cast, "[CURRENT]") {
		t.Fatalf("expected demo cast to show CURRENT status tags")
	}
	if strings.Contains(cast, "ACTIVE") {
		t.Fatalf("expected demo cast to stop showing ACTIVE statuses")
	}

	svg := mustReadFile(t, filepath.Join(root, "docs", "assets", "ww-demo.svg"))
	if !strings.Contains(svg, "<svg") {
		t.Fatalf("expected generated SVG preview asset")
	}
	if strings.Contains(svg, "ACTIVE") {
		t.Fatalf("expected generated SVG preview to stop showing ACTIVE statuses")
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
