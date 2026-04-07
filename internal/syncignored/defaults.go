// Package syncignored copies git-ignored files (typically local config such as
// .env files) from a project's main worktree into a freshly created linked
// worktree. It is intentionally best-effort: failures must never block the
// surrounding `ww new` operation.
package syncignored

// DefaultBlacklist is the built-in set of path segments that are skipped when
// syncing ignored files into a new worktree. Each entry is matched against any
// path segment of an ignored file's path (relative to the main worktree). The
// list covers common large dependency / build directories across popular
// ecosystems, plus a few well-known nuisance files.
//
// Users can extend this via config (`sync.blacklist_extra`) or replace it
// entirely (`sync.blacklist_override`).
var DefaultBlacklist = []string{
	// JS/TS
	"node_modules", ".next", ".nuxt", "dist", "build",
	".vite", ".turbo", ".parcel-cache", "coverage",
	// Python
	"__pycache__", ".venv", "venv", "env",
	".pytest_cache", ".mypy_cache",
	// Go / Rust / Java
	"vendor", "target", ".gradle",
	// General
	"tmp", "temp", "logs", ".cache",
	// Nuisance files
	".DS_Store",
}

// DefaultMaxFileSize is the default upper bound (in bytes) for individual
// ignored files to be copied. Files at or above this size are skipped to
// guarantee `ww new` stays fast even if the blacklist misses something.
const DefaultMaxFileSize int64 = 1024 * 1024 // 1 MiB
