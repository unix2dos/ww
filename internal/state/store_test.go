package state

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestStoreTouchAndLoadPersistsPerRepo(t *testing.T) {
	dir := t.TempDir()
	store := &Store{
		path: filepath.Join(dir, "state.json"),
		now: func() time.Time {
			return time.Unix(100, 0)
		},
	}

	if err := store.Touch("/repo-a/.git", "/repo-a/.worktrees/alpha"); err != nil {
		t.Fatalf("touch alpha: %v", err)
	}
	store.now = func() time.Time {
		return time.Unix(200, 0)
	}
	if err := store.Touch("/repo-a/.git", "/repo-a"); err != nil {
		t.Fatalf("touch current: %v", err)
	}
	store.now = func() time.Time {
		return time.Unix(300, 0)
	}
	if err := store.Touch("/repo-b/.git", "/repo-b/.worktrees/beta"); err != nil {
		t.Fatalf("touch beta: %v", err)
	}

	gotA, err := store.Load("/repo-a/.git")
	if err != nil {
		t.Fatalf("load repo a: %v", err)
	}
	wantA := map[string]int64{
		"/repo-a/.worktrees/alpha": time.Unix(100, 0).UnixNano(),
		"/repo-a":                  time.Unix(200, 0).UnixNano(),
	}
	if !reflect.DeepEqual(gotA, wantA) {
		t.Fatalf("repo a state mismatch: got %#v want %#v", gotA, wantA)
	}

	gotB, err := store.Load("/repo-b/.git")
	if err != nil {
		t.Fatalf("load repo b: %v", err)
	}
	wantB := map[string]int64{
		"/repo-b/.worktrees/beta": time.Unix(300, 0).UnixNano(),
	}
	if !reflect.DeepEqual(gotB, wantB) {
		t.Fatalf("repo b state mismatch: got %#v want %#v", gotB, wantB)
	}
}

func TestStoreLoadMissingRepoReturnsEmptyMap(t *testing.T) {
	dir := t.TempDir()
	store := &Store{path: filepath.Join(dir, "state.json")}

	got, err := store.Load("/repo/.git")
	if err != nil {
		t.Fatalf("load missing repo: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %#v", got)
	}
}
