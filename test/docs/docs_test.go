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
	if !strings.Contains(readme, "## Demo") {
		t.Fatalf("expected README demo section")
	}
	if !strings.Contains(readme, "https://unix2dos.github.io/ww/") {
		t.Fatalf("expected README to link to the GitHub Pages demo")
	}
	if !strings.Contains(readme, "docs/assets/ww-demo.svg") {
		t.Fatalf("expected README to embed the generated SVG demo preview")
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
	if !strings.Contains(indexHTML, "speed: 0.5") {
		t.Fatalf("expected Pages demo to default to 0.5x playback")
	}

	pagesWorkflow := mustReadFile(t, filepath.Join(root, ".github", "workflows", "pages.yml"))
	for _, snippet := range []string{
		"actions/configure-pages@v5",
		"actions/upload-pages-artifact@v3",
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

	expectScript := mustReadFile(t, filepath.Join(root, "scripts", "demo-record.exp"))
	if !strings.Contains(expectScript, "ww new") {
		t.Fatalf("expected expect demo script to exercise ww new")
	}
	if !strings.Contains(expectScript, "Use Up/Down") {
		t.Fatalf("expected expect demo script to drive the interactive selector")
	}

	cast := mustReadFile(t, filepath.Join(root, "docs", "assets", "ww-demo.cast"))
	if !strings.Contains(cast, "\"version\":2") {
		t.Fatalf("expected local asciinema cast asset")
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
