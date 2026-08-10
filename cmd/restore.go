package cmd

import (
	"fmt"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <name> <snapshot-id>",
		Short: "Reset a workspace to a snapshot and drop later snapshots",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, snapID := args[0], args[1]
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			_, ws, err := loadWorkspace(p, name)
			if err != nil {
				return err
			}
			wsPath, err := ensureCurrentExecution(p, ws)
			if err != nil {
				return err
			}

			// Find snapshot index + commit.
			idx := -1
			var commit string
			for i, s := range ws.Snapshots {
				if s.ID == snapID {
					idx = i
					commit = s.Commit
					break
				}
			}
			if idx < 0 {
				return fmt.Errorf("snapshot %q not found in workspace %q", snapID, name)
			}

			if _, err := git.Run(wsPath, "reset", "--hard", commit); err != nil {
				return err
			}

			err = p.WithLock(func(s *workspace.Store) error {
				cur := s.Find(name)
				if cur == nil {
					return fmt.Errorf("workspace %q not found", name)
				}
				// Keep snapshots up to and including idx.
				cur.Snapshots = cur.Snapshots[:idx+1]
				return nil
			})
			if err != nil {
				return err
			}

			p.AppendLog(fmt.Sprintf("restore name=%s id=%s commit=%s", name, snapID, commit))
			ui.Success("Restored %q to %s (%s); later snapshots removed", name, snapID, commit)
			return nil
		},
	}
	return cmd
}
