package cmd

import (
	"fmt"
	"strings"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <name>",
		Short: "Show details and changes for a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			_, ws, err := loadWorkspace(p, name)
			if err != nil {
				return err
			}
			wsPath, pathErr := executionPath(p, ws)

			ui.Bold("Workspace: %s", ws.Name)
			ui.Plain("  Description: %s", orDash(ws.Description))
			ui.Plain("  Branch:      %s", ws.Branch)
			ui.Plain("  Source:      %s (commit %s)", ws.BaseBranch, ws.BaseCommit)
			ui.Plain("  Merge into:  %s", targetBranch(ws))
			ui.Plain("  Status:      %s", ws.Status)
			ui.Plain("  Created:     %s", ws.CreatedAt)
			if pathErr != nil {
				ui.Plain("  Path:        (waiting for Codex Worker attach)")
			} else {
				ui.Plain("  Path:        %s", wsPath)
			}
			ui.Plain("  Execution:   %s", executionKind(ws))
			if ws.Handoff != nil {
				ui.Plain("  Handoff:     %s (%s)", ws.Handoff.Commit, formatTimestamp(ws.Handoff.SubmittedAt))
				if reviewSpace := reviewWorkspaceForCommit(ws, ws.Handoff.Commit); reviewSpace != nil {
					ui.Plain("  Review path:  %s", reviewSpace.Path)
				} else {
					ui.Plain("  Review:      agentspace review %s --open", ws.Name)
				}
			}
			if ws.Review != nil {
				ui.Plain("  Review:      %s (%s)", ws.Review.Decision, formatTimestamp(ws.Review.ReviewedAt))
			}
			if len(ws.Task.AcceptanceCriteria) > 0 {
				ui.Plain("  Acceptance: %s", strings.Join(ws.Task.AcceptanceCriteria, "; "))
			}
			if len(ws.Task.FileScope) > 0 {
				ui.Plain("  Scope:      %s", strings.Join(ws.Task.FileScope, ", "))
			}
			if len(ws.Task.DependsOn) > 0 {
				ui.Plain("  Depends on: %s", strings.Join(ws.Task.DependsOn, ", "))
			}
			if ws.Task.DispatchedAt != "" {
				ui.Plain("  Dispatched:  %s", formatTimestamp(ws.Task.DispatchedAt))
			}
			if ws.WorkerTask != nil {
				ui.Plain("  Worker task: %s (%s)", ws.WorkerTask.Title, ws.WorkerTask.ID)
				if ws.WorkerTask.LastKnownStatus != "" {
					ui.Plain("  Task status: %s (%s)", ws.WorkerTask.LastKnownStatus, formatTimestamp(ws.WorkerTask.ObservedAt))
				}
			}
			if ws.Status == workspace.StatusPaused {
				ui.Plain("  Paused:      %s", formatTimestamp(ws.Control.PausedAt))
				ui.Plain("  Pause reason: %s", ws.Control.PauseReason)
			}
			if hasPendingOverride(ws) {
				ui.Plain("  Override:    %s", ws.Control.Override.Reason)
			}
			ui.Plain("  Events:      %d", len(ws.Events))

			// Diff stat vs base commit.
			ui.Bold("\nChanges since base (%s):", ws.BaseCommit)
			stat := ""
			if pathErr == nil {
				var err error
				stat, err = git.Run(wsPath, "diff", "--stat", ws.BaseCommit)
				if err != nil {
					return err
				}
			}
			if stat == "" {
				ui.Plain("  (no committed changes)")
			} else {
				fmt.Println(indent(stat))
			}

			// Uncommitted changes.
			ui.Bold("Uncommitted changes:")
			short := ""
			if pathErr == nil {
				var err error
				short, err = git.Run(wsPath, "status", "--short")
				if err != nil {
					return err
				}
			}
			if short == "" {
				ui.Plain("  (clean)")
			} else {
				fmt.Println(indent(short))
			}

			// Snapshots.
			ui.Bold("Snapshots: %d", len(ws.Snapshots))
			for _, s := range ws.Snapshots {
				ui.Plain("  %s  %s  %s", s.ID, s.Commit, s.Message)
			}
			return nil
		},
	}
	return cmd
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
