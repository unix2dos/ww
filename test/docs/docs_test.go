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
	if !strings.Contains(readme, "One command to switch, create, and clean up worktrees.") {
		t.Fatalf("expected README landing-page value proposition")
	}
	if !strings.Contains(readme, "## Demo") {
		t.Fatalf("expected README demo section")
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
	if !strings.Contains(readme, "## For AI Agents") {
		t.Fatalf("expected README to document the machine-readable entrypoint")
	}
	if !strings.Contains(readme, "ww-helper list --json") {
		t.Fatalf("expected README to show ww-helper json usage")
	}
	if strings.Contains(readme, "### Install From Source") {
		t.Fatalf("expected README to stay in landing-page mode, not inline the full reference")
	}

	reference := mustReadFile(t, filepath.Join(root, "docs", "reference.md"))
	for _, snippet := range []string{
		"# ww Reference",
		"## Install",
		"## Usage",
		"## Release",
		"`ww help` or `ww --help` prints the command summary.",
		"### For AI Agents",
		"ww-helper rm --json --non-interactive feat-a",
		"#### Breaking Change",
	} {
		if !strings.Contains(reference, snippet) {
			t.Fatalf("expected reference doc to contain %q", snippet)
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
	if !strings.Contains(indexHTML, "`switch`, `ww new`, and safe `ww rm`") {
		t.Fatalf("expected Pages demo copy to match the refreshed flow")
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
	if !strings.Contains(expectScript, "send_text \"feat-a\"") {
		t.Fatalf("expected expect demo script to drive the fzf query path")
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

	svg := mustReadFile(t, filepath.Join(root, "docs", "assets", "ww-demo.svg"))
	if !strings.Contains(svg, "<svg") {
		t.Fatalf("expected generated SVG preview asset")
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
