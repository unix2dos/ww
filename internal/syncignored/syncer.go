package syncignored

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Runner abstracts command execution so the syncer can be unit tested without
// invoking real git. It mirrors the shape of internal/git.Runner so callers can
// pass the same implementation.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout []byte, stderr []byte, err error)
}

// Options controls a Sync invocation.
type Options struct {
	// Enabled toggles the entire operation. When false, Sync is a no-op and
	// returns an empty Result with no error.
	Enabled bool
	// MaxFileSize is the upper bound (in bytes) for individual files to copy.
	// Files at or above this size are skipped. Zero (or negative) means use
	// DefaultMaxFileSize.
	MaxFileSize int64
	// Blacklist is the set of path segments to skip. Nil means use
	// DefaultBlacklist. Trailing slashes on entries are tolerated and
	// normalised away.
	Blacklist []string
	// DryRun, when true, performs every check but does not write any files.
	// The Result's Copied slice still lists what would have been copied.
	DryRun bool
}

// SkipReason categorises why a file was not copied.
type SkipReason string

const (
	SkipBlacklisted SkipReason = "blacklisted"
	SkipTooLarge    SkipReason = "too_large"
	SkipNotRegular  SkipReason = "not_regular"
	SkipReadError   SkipReason = "read_error"
)

// Skipped describes a single skipped file.
type Skipped struct {
	Path   string
	Reason SkipReason
	Size   int64 // populated for SkipTooLarge and (when known) SkipReadError
}

// Result summarises a Sync run.
type Result struct {
	Copied  []string
	Skipped []Skipped
	DryRun  bool
}

// Sync copies files that are present in mainRoot but ignored by git into
// target. It uses
//
//	git -C mainRoot ls-files --others --ignored --exclude-standard
//
// as the source of truth for "ignored files".
//
// Sync is best-effort: callers should treat any returned error as a warning and
// not fail the surrounding operation. Per-file failures are recorded in
// Result.Skipped instead of being returned as errors.
func Sync(ctx context.Context, runner Runner, mainRoot, target string, opts Options) (Result, error) {
	res := Result{DryRun: opts.DryRun}
	if !opts.Enabled {
		return res, nil
	}
	if runner == nil {
		return res, errors.New("syncignored: runner must not be nil")
	}
	if mainRoot == "" || target == "" {
		return res, errors.New("syncignored: mainRoot and target must be non-empty")
	}
	if mainRoot == target {
		// Nothing to do — caller is asking us to sync into the source.
		return res, nil
	}

	maxSize := opts.MaxFileSize
	if maxSize <= 0 {
		maxSize = DefaultMaxFileSize
	}
	blacklist := normaliseBlacklist(opts.Blacklist)
	blacklistSet := make(map[string]struct{}, len(blacklist))
	for _, b := range blacklist {
		blacklistSet[b] = struct{}{}
	}

	stdout, stderr, err := runner.Run(ctx, "git", "-C", mainRoot,
		"ls-files", "--others", "--ignored", "--exclude-standard")
	if err != nil {
		msg := strings.TrimSpace(string(stderr))
		if msg != "" {
			return res, fmt.Errorf("syncignored: git ls-files: %w: %s", err, msg)
		}
		return res, fmt.Errorf("syncignored: git ls-files: %w", err)
	}

	for _, rel := range splitLines(stdout) {
		if rel == "" {
			continue
		}

		if isBlacklisted(rel, blacklistSet) {
			res.Skipped = append(res.Skipped, Skipped{Path: rel, Reason: SkipBlacklisted})
			continue
		}

		// ls-files always emits forward slashes; translate for the local OS.
		relOS := filepath.FromSlash(rel)
		srcAbs := filepath.Join(mainRoot, relOS)

		info, err := os.Lstat(srcAbs)
		if err != nil {
			res.Skipped = append(res.Skipped, Skipped{Path: rel, Reason: SkipReadError})
			continue
		}
		if !info.Mode().IsRegular() {
			res.Skipped = append(res.Skipped, Skipped{Path: rel, Reason: SkipNotRegular})
			continue
		}
		if info.Size() >= maxSize {
			res.Skipped = append(res.Skipped, Skipped{
				Path: rel, Reason: SkipTooLarge, Size: info.Size(),
			})
			continue
		}

		if opts.DryRun {
			res.Copied = append(res.Copied, rel)
			continue
		}

		dstAbs := filepath.Join(target, relOS)
		if err := copyFile(srcAbs, dstAbs, info.Mode().Perm()); err != nil {
			res.Skipped = append(res.Skipped, Skipped{
				Path: rel, Reason: SkipReadError, Size: info.Size(),
			})
			continue
		}
		res.Copied = append(res.Copied, rel)
	}

	return res, nil
}

// normaliseBlacklist returns a copy of the provided list with trailing slashes
// trimmed and empty entries dropped. nil input yields DefaultBlacklist.
func normaliseBlacklist(in []string) []string {
	if in == nil {
		return DefaultBlacklist
	}
	out := make([]string, 0, len(in))
	for _, e := range in {
		e = strings.TrimSpace(e)
		e = strings.TrimSuffix(e, "/")
		if e == "" {
			continue
		}
		out = append(out, e)
	}
	return out
}

// isBlacklisted returns true if any path segment of rel matches an entry in
// the blacklist set. Matching is exact and case-sensitive.
func isBlacklisted(rel string, set map[string]struct{}) bool {
	for _, part := range strings.Split(rel, "/") {
		if _, ok := set[part]; ok {
			return true
		}
	}
	return false
}

func splitLines(b []byte) []string {
	s := strings.TrimRight(string(b), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func copyFile(src, dst string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
