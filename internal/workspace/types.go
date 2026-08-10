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

// CheckResult captures one command the worker ran before submitting work.
// Output is intentionally capped by the caller so workspace metadata remains small.
type CheckResult struct {
	Command string `json:"command"`
	Passed  bool   `json:"passed"`
	Output  string `json:"output,omitempty"`
}

// Handoff is the immutable worker result that a master reviews. Commit must
// remain the workspace HEAD until the handoff is approved and merged.
type Handoff struct {
	Commit      string        `json:"commit"`
	Summary     string        `json:"summary"`
	Files       []string      `json:"files"`
	Checks      []CheckResult `json:"checks"`
	SubmittedAt string        `json:"submitted_at"`
}

// Review records the master's decision for a submitted handoff.
type Review struct {
	Commit     string `json:"commit"`
	Decision   string `json:"decision"`
	Summary    string `json:"summary"`
	ReviewedAt string `json:"reviewed_at"`
}

// TaskManifest contains the durable coordination details for a workspace.
// It intentionally does not retain a short-lived provider-specific agent ID.
type TaskManifest struct {
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	FileScope          []string `json:"file_scope"`
	DependsOn          []string `json:"depends_on"`
	DispatchedAt       string   `json:"dispatched_at,omitempty"`
}

// Event is an append-only record of a workflow transition or runner outcome.
type Event struct {
	At      string `json:"at"`
	Type    string `json:"type"`
	Summary string `json:"summary,omitempty"`
	Commit  string `json:"commit,omitempty"`
}

// Workspace is the metadata for one git-worktree-backed workspace.
type Workspace struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Prompt      string       `json:"prompt"`
	Branch      string       `json:"branch"`
	BaseCommit  string       `json:"base_commit"`
	BaseBranch  string       `json:"base_branch"`
	Status      string       `json:"status"`
	CreatedAt   string       `json:"created_at"`
	Path        string       `json:"path"`
	Snapshots   []Snapshot   `json:"snapshots"`
	Handoff     *Handoff     `json:"handoff,omitempty"`
	Review      *Review      `json:"review,omitempty"`
	Task        TaskManifest `json:"task"`
	Events      []Event      `json:"events,omitempty"`
}

// Store is the on-disk shape of workspaces.json.
type Store struct {
	Workspaces []Workspace `json:"workspaces"`
}

// Workspace status values.
const (
	StatusActive     = "active"
	StatusSubmitted  = "submitted"
	StatusChangesReq = "changes_requested"
	StatusAccepted   = "accepted"
	StatusConflicted = "conflicted"
	StatusMerged     = "merged"
	StatusCancelled  = "cancelled"
	StatusFailed     = "failed"
)

// Timestamp formats a time in the RFC3339 layout used throughout the store.
func Timestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// TimestampNow returns the current timestamp in the store's canonical format.
func TimestampNow() string {
	return Timestamp(time.Now())
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
