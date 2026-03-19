package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Store struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
}

type diskState struct {
	Repos map[string]map[string]int64 `json:"repos"`
}

func NewStore() (*Store, error) {
	path, err := defaultPath()
	if err != nil {
		return nil, err
	}
	return &Store{path: path, now: time.Now}, nil
}

func NewStoreAt(path string) *Store {
	return &Store{path: path, now: time.Now}
}

func (s *Store) Load(repoKey string) (map[string]int64, error) {
	var out map[string]int64
	err := s.withLock(func() error {
		state, err := s.readLocked()
		if err != nil {
			return err
		}
		out = cloneRepoState(state.Repos[repoKey])
		return nil
	})
	return out, err
}

func (s *Store) Touch(repoKey, path string) error {
	return s.withLock(func() error {
		state, err := s.readLocked()
		if err != nil {
			return err
		}
		if state.Repos == nil {
			state.Repos = make(map[string]map[string]int64)
		}
		repo := state.Repos[repoKey]
		if repo == nil {
			repo = make(map[string]int64)
		}
		repo[path] = s.nowFunc().UnixNano()
		state.Repos[repoKey] = repo
		return s.writeLocked(state)
	})
}

func (s *Store) withLock(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.path == "" {
		return errors.New("state path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	lockPath := s.lockPath()
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	return fn()
}

func (s *Store) readLocked() (diskState, error) {
	if s.path == "" {
		return diskState{}, errors.New("state path is empty")
	}

	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return diskState{}, nil
		}
		return diskState{}, err
	}

	if len(raw) == 0 {
		return diskState{}, nil
	}

	var state diskState
	if err := json.Unmarshal(raw, &state); err != nil {
		return diskState{}, err
	}
	if state.Repos == nil {
		state.Repos = make(map[string]map[string]int64)
	}
	return state, nil
}

func (s *Store) writeLocked(state diskState) error {
	if s.path == "" {
		return errors.New("state path is empty")
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')

	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, filepath.Base(s.path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(encoded); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.path)
}

func (s *Store) lockPath() string {
	return s.path + ".lock"
}

func (s *Store) nowFunc() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func cloneRepoState(src map[string]int64) map[string]int64 {
	if len(src) == 0 {
		return map[string]int64{}
	}
	out := make(map[string]int64, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func defaultPath() (string, error) {
	if base := os.Getenv("XDG_STATE_HOME"); base != "" {
		return filepath.Join(base, "ww", "state.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "ww", "state.json"), nil
}
