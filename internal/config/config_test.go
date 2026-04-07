package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFileReturnsDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Sync.SyncEnabled() {
		t.Error("default sync should be enabled")
	}
	if cfg.Sync.EffectiveMaxFileSize() != 0 {
		t.Error("default max_file_size should signal 'use caller default' (0)")
	}
}

func TestLoadEmptyFileReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Sync.SyncEnabled() {
		t.Error("expected default enabled=true")
	}
}

func TestLoadParsesUserValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	body := `{
		"version": 1,
		"sync": {
			"enabled": false,
			"max_file_size": 2048,
			"blacklist_extra": ["my-stuff/"]
		}
	}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Sync.SyncEnabled() {
		t.Error("expected enabled=false")
	}
	if cfg.Sync.EffectiveMaxFileSize() != 2048 {
		t.Errorf("expected MaxFileSize=2048, got %d", cfg.Sync.EffectiveMaxFileSize())
	}
	bl := cfg.Sync.EffectiveBlacklist([]string{"node_modules", "dist"})
	want := []string{"node_modules", "dist", "my-stuff/"}
	if strings.Join(bl, ",") != strings.Join(want, ",") {
		t.Errorf("blacklist = %v, want %v", bl, want)
	}
}

func TestBlacklistOverrideReplacesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	body := `{"sync": {"blacklist_override": ["only-this/"]}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bl := cfg.Sync.EffectiveBlacklist([]string{"node_modules", "dist"})
	if len(bl) != 1 || bl[0] != "only-this/" {
		t.Errorf("override should replace defaults, got %v", bl)
	}
}

func TestBlacklistOverrideEmptyDisablesAll(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"sync": {"blacklist_override": []}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bl := cfg.Sync.EffectiveBlacklist([]string{"node_modules"})
	if len(bl) != 0 {
		t.Errorf("expected empty blacklist, got %v", bl)
	}
}

func TestLoadInvalidJSONReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
	// Caller should still get a usable default Config back.
	if !cfg.Sync.SyncEnabled() {
		t.Error("expected fallback default to be enabled")
	}
}

func TestDefaultPathHonoursXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/custom-xdg")
	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/tmp/custom-xdg/ww/config.json"
	if got != want {
		t.Errorf("DefaultPath = %q, want %q", got, want)
	}
}

func TestDefaultPathFallbackToHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(got, filepath.Join(".config", "ww", "config.json")) {
		t.Errorf("expected ~/.config/ww/config.json suffix, got %q", got)
	}
}

func TestBlacklistOverrideReturnsCopy(t *testing.T) {
	override := []string{"a", "b"}
	cfg := Config{Sync: SyncConfig{BlacklistOverride: &override}}
	got := cfg.Sync.EffectiveBlacklist(nil)
	got[0] = "MUTATED"
	if override[0] != "a" {
		t.Errorf("EffectiveBlacklist should return a copy, but original was mutated: %v", override)
	}
}
