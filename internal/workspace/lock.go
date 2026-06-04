package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Lock is a simple cross-platform advisory file lock implemented via an
// exclusively-created lock file. It guards concurrent writes to workspaces.json.
type Lock struct {
	path string
}

// Acquire takes the lock for the agentspace base dir, retrying briefly if it is
// already held. Callers must call Release when done.
func (p Paths) Acquire() (*Lock, error) {
	lockPath := filepath.Join(p.Base, "workspaces.lock")
	deadline := time.Now().Add(5 * time.Second)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d", os.Getpid())
			_ = f.Close()
			return &Lock{path: lockPath}, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("acquire lock: %w", err)
		}
		// Stale lock detection: if the lock file is old, reclaim it.
		if info, statErr := os.Stat(lockPath); statErr == nil {
			if time.Since(info.ModTime()) > 30*time.Second {
				_ = os.Remove(lockPath)
				continue
			}
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("could not acquire workspaces lock (held by another process)")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Release frees the lock.
func (l *Lock) Release() {
	if l != nil && l.path != "" {
		_ = os.Remove(l.path)
	}
}

// WithLock runs fn while holding the workspaces lock, loading the store before
// and saving it after. fn may mutate the passed store.
func (p Paths) WithLock(fn func(s *Store) error) error {
	lock, err := p.Acquire()
	if err != nil {
		return err
	}
	defer lock.Release()

	s, err := p.LoadStore()
	if err != nil {
		return err
	}
	if err := fn(s); err != nil {
		return err
	}
	return p.SaveStore(s)
}
