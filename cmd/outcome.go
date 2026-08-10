package cmd

import (
	"fmt"
	"strings"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newCancelCmd() *cobra.Command {
	return newOutcomeCmd("cancel", workspace.StatusCancelled, "Cancel a workspace and record the reason")
}

func newFailCmd() *cobra.Command {
	return newOutcomeCmd("fail", workspace.StatusFailed, "Mark a workspace failed and record the reason")
}

func newOutcomeCmd(use, nextStatus, short string) *cobra.Command {
	var message string
	cmd := &cobra.Command{
		Use:   use + " <name>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(message) == "" {
				return fmt.Errorf("--message/-m is required")
			}
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			if err := p.WithLock(func(s *workspace.Store) error {
				ws := s.Find(args[0])
				if ws == nil {
					return fmt.Errorf("workspace %q not found", args[0])
				}
				if ws.Status == workspace.StatusMerged || ws.Status == workspace.StatusCancelled || ws.Status == workspace.StatusFailed {
					return fmt.Errorf("workspace %q cannot transition from terminal status %s", ws.Name, ws.Status)
				}
				ws.Status = nextStatus
				appendEvent(ws, nextStatus, message, "")
				return nil
			}); err != nil {
				return err
			}
			p.AppendLog(fmt.Sprintf("%s name=%s", use, args[0]))
			ui.Success("Marked %q as %s", args[0], nextStatus)
			return nil
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "reason (required)")
	return cmd
}
