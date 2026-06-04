package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// Dir is the project subdirectory holding all agentspace state.
	Dir = ".agentspace"
)

// ErrNotInitialized is returned when .agentspace is missing.
var ErrNotInitialized = errors.New("agentspace not initialized in this repository (run 'agentspace init')")

// Paths resolves the standard agentspace paths relative to a repo root.
type Paths struct {
	Root          string // repo root
	Base          string // <root>/.agentspace
	ConfigFile    string // <root>/.agentspace/config.json
	StoreFile     string // <root>/.agentspace/workspaces.json
	LogsDir       string // <root>/.agentspace/logs
	WorkspacesDir string // <root>/.agentspace/workspaces
}

// NewPaths builds the Paths for a given repo root.
func NewPaths(root string) Paths {
	base := filepath.Join(root, Dir)
	return Paths{
		Root:          root,
		Base:          base,
		ConfigFile:    filepath.Join(base, "config.json"),
		StoreFile:     filepath.Join(base, "workspaces.json"),
		LogsDir:       filepath.Join(base, "logs"),
		WorkspacesDir: filepath.Join(base, "workspaces"),
	}
}

// Initialized reports whether .agentspace exists at the repo root.
func (p Paths) Initialized() bool {
	info, err := os.Stat(p.Base)
	return err == nil && info.IsDir()
}

// LoadConfig reads config.json.
func (p Paths) LoadConfig() (*Config, error) {
	data, err := os.ReadFile(p.ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotInitialized
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config.json: %w", err)
	}
	return &c, nil
}

// SaveConfig writes config.json (pretty-printed).
func (p Paths) SaveConfig(c *Config) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.ConfigFile, data, 0o644)
}

// LoadStore reads workspaces.json. A missing file yields an empty store.
func (p Paths) LoadStore() (*Store, error) {
	data, err := os.ReadFile(p.StoreFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &Store{}, nil
		}
		return nil, err
	}
	var s Store
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parse workspaces.json: %w", err)
		}
	}
	return &s, nil
}

// SaveStore writes workspaces.json atomically (temp file + rename).
func (p Paths) SaveStore(s *Store) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := p.StoreFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p.StoreFile)
}

// NewSnapshotID returns a snapshot id based on the existing count.
func NewSnapshotID(w *Workspace) string {
	return fmt.Sprintf("snap-%03d", len(w.Snapshots)+1)
}

// AppendLog appends a timestamped line to the operations log.
func (p Paths) AppendLog(line string) {
	_ = os.MkdirAll(p.LogsDir, 0o755)
	f, err := os.OpenFile(filepath.Join(p.LogsDir, "agentspace.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339), line)
}
