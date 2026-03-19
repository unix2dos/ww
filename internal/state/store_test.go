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

func TestStoreTouchSerializesConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	storeA := &Store{
		path: path,
		now: func() time.Time {
			return time.Unix(100, 0)
		},
	}
	storeB := &Store{
		path: path,
		now: func() time.Time {
			return time.Unix(200, 0)
		},
	}

	errCh := make(chan error, 2)
	go func() {
		errCh <- storeA.Touch("/repo/.git", "/repo/.worktrees/alpha")
	}()
	go func() {
		errCh <- storeB.Touch("/repo/.git", "/repo/.worktrees/beta")
	}()

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("touch %d: %v", i, err)
		}
	}

	got, err := storeA.Load("/repo/.git")
	if err != nil {
		t.Fatalf("load after concurrent touch: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected both entries to persist, got %#v", got)
	}
}

func TestStoreLockBlocksSecondInstance(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	storeA := &Store{path: path}
	storeB := &Store{path: path}

	held := make(chan struct{})
	released := make(chan struct{})
	acquired := make(chan struct{})
	errCh := make(chan error, 2)

	go func() {
		errCh <- storeA.withLock(func() error {
			close(held)
			<-released
			return nil
		})
	}()

	<-held
	go func() {
		errCh <- storeB.withLock(func() error {
			close(acquired)
			return nil
		})
	}()

	select {
	case <-acquired:
		t.Fatal("second store acquired lock before first released it")
	case <-time.After(50 * time.Millisecond):
	}

	close(released)

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second store did not acquire lock after release")
	}

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("withLock %d: %v", i, err)
		}
	}
}
