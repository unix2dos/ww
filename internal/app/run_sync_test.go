package app

import (
	"context"

	"ww/internal/config"
	"ww/internal/syncignored"
)

// init neutralises the ignored-file sync in all unit tests of this package.
// End-to-end coverage of the sync behaviour lives in test/e2e and in
// internal/syncignored's own unit tests; package-level unit tests here should
// not touch the real filesystem or shell out to git from fake repo paths.
func init() {
	syncIgnoredFn = func(context.Context, string, string, syncignored.Options) (syncignored.Result, error) {
		return syncignored.Result{}, nil
	}
	loadSyncConfigFn = func() (config.Config, error) {
		return config.Default(), nil
	}
}
