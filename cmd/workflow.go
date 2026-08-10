package cmd

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/asam264/agentspace/internal/workspace"
)

func normalizeTextList(values []string, label string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%s cannot be empty", label)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

// normalizeScope accepts only repo-relative file or directory paths. Globs are
// deliberately not supported because overlap checks must be deterministic.
func normalizeScope(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(filepath.ToSlash(value))
		if value == "" || filepath.IsAbs(value) || path.IsAbs(value) {
			return nil, fmt.Errorf("scope must be a non-empty repository-relative path")
		}
		value = path.Clean(strings.Trim(value, "/"))
		if value == "." || value == ".." || strings.HasPrefix(value, "../") || strings.ContainsAny(value, "*?[") {
			return nil, fmt.Errorf("scope %q must be a repository-relative file or directory path, not a glob", value)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func appendEvent(ws *workspace.Workspace, eventType, summary, commit string) {
	ws.Events = append(ws.Events, workspace.Event{
		At:      workspace.Timestamp(time.Now()),
		Type:    eventType,
		Summary: strings.TrimSpace(summary),
		Commit:  commit,
	})
}

func recordEvent(p workspace.Paths, name, eventType, summary, commit string) error {
	return p.WithLock(func(s *workspace.Store) error {
		ws := s.Find(name)
		if ws == nil {
			return fmt.Errorf("workspace %q not found", name)
		}
		appendEvent(ws, eventType, summary, commit)
		return nil
	})
}

func isOpenWorkspaceStatus(status string) bool {
	switch status {
	case workspace.StatusActive, workspace.StatusSubmitted, workspace.StatusChangesReq, workspace.StatusAccepted, workspace.StatusConflicted:
		return true
	default:
		return false
	}
}

func scopeOverlaps(left, right string) bool {
	return left == right || strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}

func dependencyBlockers(store *workspace.Store, ws *workspace.Workspace) []string {
	blockers := make([]string, 0)
	for _, dependency := range ws.Task.DependsOn {
		dep := store.Find(dependency)
		if dep == nil {
			blockers = append(blockers, fmt.Sprintf("dependency %q no longer exists", dependency))
			continue
		}
		if dep.Status != workspace.StatusMerged {
			blockers = append(blockers, fmt.Sprintf("dependency %q is %s, not merged", dependency, dep.Status))
		}
	}
	return blockers
}
