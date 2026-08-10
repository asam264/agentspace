package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
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
			wsPath := filepath.Join(p.Root, filepath.FromSlash(ws.Path))

			ui.Bold("Workspace: %s", ws.Name)
			ui.Plain("  Description: %s", orDash(ws.Description))
			ui.Plain("  Branch:      %s", ws.Branch)
			ui.Plain("  Base:        %s (commit %s)", ws.BaseBranch, ws.BaseCommit)
			ui.Plain("  Status:      %s", ws.Status)
			ui.Plain("  Created:     %s", ws.CreatedAt)
			ui.Plain("  Path:        %s", wsPath)
			if ws.Handoff != nil {
				ui.Plain("  Handoff:     %s (%s)", ws.Handoff.Commit, formatTimestamp(ws.Handoff.SubmittedAt))
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
			ui.Plain("  Events:      %d", len(ws.Events))

			// Diff stat vs base commit.
			ui.Bold("\nChanges since base (%s):", ws.BaseCommit)
			stat, err := git.Run(wsPath, "diff", "--stat", ws.BaseCommit)
			if err != nil {
				return err
			}
			if stat == "" {
				ui.Plain("  (no committed changes)")
			} else {
				fmt.Println(indent(stat))
			}

			// Uncommitted changes.
			ui.Bold("Uncommitted changes:")
			short, err := git.Run(wsPath, "status", "--short")
			if err != nil {
				return err
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
