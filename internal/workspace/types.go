package workspace

import "time"

// Config is the project-level configuration stored in .agentspace/config.json.
type Config struct {
	Version       string `json:"version"`
	BaseBranch    string `json:"base_branch"`
	CreatedAt     string `json:"created_at"`
	MergeStrategy string `json:"merge_strategy"`
	Editor        string `json:"editor"`
}

// Snapshot is a single saved commit point within a workspace.
type Snapshot struct {
	ID        string `json:"id"`
	Commit    string `json:"commit"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

// Workspace is the metadata for one git-worktree-backed workspace.
type Workspace struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Branch      string     `json:"branch"`
	BaseCommit  string     `json:"base_commit"`
	BaseBranch  string     `json:"base_branch"`
	Status      string     `json:"status"`
	CreatedAt   string     `json:"created_at"`
	Path        string     `json:"path"`
	Snapshots   []Snapshot `json:"snapshots"`
}

// Store is the on-disk shape of workspaces.json.
type Store struct {
	Workspaces []Workspace `json:"workspaces"`
}

// Workspace status values.
const (
	StatusActive     = "active"
	StatusConflicted = "conflicted"
	StatusMerged     = "merged"
)

// Timestamp formats a time in the RFC3339 layout used throughout the store.
func Timestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// Find returns a pointer to the workspace with the given name, or nil.
func (s *Store) Find(name string) *Workspace {
	for i := range s.Workspaces {
		if s.Workspaces[i].Name == name {
			return &s.Workspaces[i]
		}
	}
	return nil
}

// Remove deletes the workspace with the given name. Returns true if removed.
func (s *Store) Remove(name string) bool {
	for i := range s.Workspaces {
		if s.Workspaces[i].Name == name {
			s.Workspaces = append(s.Workspaces[:i], s.Workspaces[i+1:]...)
			return true
		}
	}
	return false
}
