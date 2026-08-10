package cmd

import (
	"fmt"
	"time"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newSnapshotCmd() *cobra.Command {
	var message string
	cmd := &cobra.Command{
		Use:   "snapshot <name>",
		Short: "Save a snapshot (commit) of the workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if message == "" {
				return fmt.Errorf("--message/-m is required")
			}
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

			if _, err := git.Run(wsPath, "add", "-A"); err != nil {
				return err
			}
			// Only commit if there is something staged.
			if _, err := git.Run(wsPath, "diff", "--cached", "--quiet"); err == nil {
				ui.Warn("nothing to snapshot (no changes)")
				return nil
			}
			if _, err := git.Run(wsPath, "commit", "--no-verify", "-m", "[snapshot] "+message); err != nil {
				return err
			}
			commit, err := git.RevParseShort(wsPath, "HEAD")
			if err != nil {
				return err
			}

			var snapID string
			err = p.WithLock(func(s *workspace.Store) error {
				cur := s.Find(name)
				if cur == nil {
					return fmt.Errorf("workspace %q not found", name)
				}
				snapID = workspace.NewSnapshotID(cur)
				cur.Snapshots = append(cur.Snapshots, workspace.Snapshot{
					ID:        snapID,
					Commit:    commit,
					Message:   message,
					CreatedAt: workspace.Timestamp(time.Now()),
				})
				return nil
			})
			if err != nil {
				return err
			}

			p.AppendLog(fmt.Sprintf("snapshot name=%s id=%s commit=%s", name, snapID, commit))
			ui.Success("Saved snapshot %s (%s): %s", snapID, commit, message)
			return nil
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "snapshot description (required)")
	return cmd
}
