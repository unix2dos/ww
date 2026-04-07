// Package config loads ww's user-facing configuration from
// $XDG_CONFIG_HOME/ww/config.json (default ~/.config/ww/config.json).
//
// The file is OPTIONAL: a missing file is not an error and yields a fully
// populated default Config. The schema is forward-compatible — unknown fields
// are tolerated by the JSON decoder.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Config is the top-level user configuration.
type Config struct {
	Version int        `json:"version"`
	Sync    SyncConfig `json:"sync"`
}

// SyncConfig controls the post-create file-sync feature of `ww new`.
//
// Pointer fields distinguish "absent in JSON" (use built-in default) from
// "explicitly set" (use the user's value, even if it's the zero value).
type SyncConfig struct {
	// Enabled toggles the entire feature. Defaults to true when unset.
	Enabled *bool `json:"enabled,omitempty"`
	// MaxFileSize is the per-file size cap in bytes. Defaults to 1 MiB when
	// unset or non-positive.
	MaxFileSize *int64 `json:"max_file_size,omitempty"`
	// BlacklistExtra is appended to the built-in blacklist.
	BlacklistExtra []string `json:"blacklist_extra,omitempty"`
	// BlacklistOverride, when non-nil, REPLACES the built-in blacklist
	// entirely. Use with care. An empty (but non-nil) slice disables the
	// blacklist completely.
	BlacklistOverride *[]string `json:"blacklist_override,omitempty"`
}

// SyncEnabled returns the effective enabled flag, applying the default.
func (s SyncConfig) SyncEnabled() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

// EffectiveMaxFileSize returns the effective per-file size cap. Callers should
// pass the package default; we return zero to mean "use the consumer's
// default" so this package stays decoupled from internal/syncignored.
func (s SyncConfig) EffectiveMaxFileSize() int64 {
	if s.MaxFileSize == nil || *s.MaxFileSize <= 0 {
		return 0
	}
	return *s.MaxFileSize
}

// EffectiveBlacklist returns the blacklist the syncer should use, given the
// caller-supplied built-in default. Resolution rules:
//
//  1. If BlacklistOverride is non-nil, return it as-is (even if empty).
//  2. Otherwise, return defaults concatenated with BlacklistExtra.
func (s SyncConfig) EffectiveBlacklist(defaults []string) []string {
	if s.BlacklistOverride != nil {
		// Return a copy to prevent callers from mutating user data.
		out := make([]string, len(*s.BlacklistOverride))
		copy(out, *s.BlacklistOverride)
		return out
	}
	if len(s.BlacklistExtra) == 0 {
		return defaults
	}
	out := make([]string, 0, len(defaults)+len(s.BlacklistExtra))
	out = append(out, defaults...)
	out = append(out, s.BlacklistExtra...)
	return out
}

// Default returns a Config populated with all built-in defaults.
func Default() Config {
	return Config{Version: 1}
}

// DefaultPath returns the default config file path, honouring
// $XDG_CONFIG_HOME with a fallback to ~/.config/ww/config.json.
func DefaultPath() (string, error) {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "ww", "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "ww", "config.json"), nil
}

// Load reads the config file at the given path. A missing file is NOT an
// error: the returned Config is the built-in default.
//
// Any other I/O or parse error is returned to the caller, who should treat it
// as a warning and fall back to defaults.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Default(), nil
		}
		return Default(), fmt.Errorf("config: read %s: %w", path, err)
	}

	cfg := Default()
	if len(data) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), fmt.Errorf("config: parse %s: %w", path, err)
	}
	return cfg, nil
}

// LoadDefault is a convenience that resolves DefaultPath and loads it. If the
// path cannot be resolved (e.g. no home directory), it logs nothing and
// returns Default() with the underlying error.
func LoadDefault() (Config, error) {
	path, err := DefaultPath()
	if err != nil {
		return Default(), err
	}
	return Load(path)
}
