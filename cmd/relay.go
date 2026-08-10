package cmd

import (
	"fmt"
	"strings"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newRelayCmd() *cobra.Command {
	var status, message string
	cmd := &cobra.Command{
		Use:   "relay <name>",
		Short: "Record a Worker completion-relay outcome",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			status = strings.TrimSpace(status)
			message = strings.TrimSpace(message)
			if status != "attempted" && status != "delivered" && status != "failed" && status != "unavailable" {
				return fmt.Errorf("--status must be attempted, delivered, failed, or unavailable")
			}
			if message == "" {
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
				if ws.Handoff == nil {
					return fmt.Errorf("workspace %q has no Handoff to relay", ws.Name)
				}
				if status != "unavailable" && (ws.Master == nil || ws.Master.TaskID == "" || !ws.Master.RelaySupported) {
					return fmt.Errorf("workspace %q has no Master Endpoint that supports Completion Relay; record --status unavailable instead", ws.Name)
				}
				if status == "delivered" || status == "failed" {
					if !hasRelayAttempt(ws, ws.Handoff.Commit) {
						return fmt.Errorf("workspace %q must record completion relay --status attempted for handoff %s before %s", ws.Name, ws.Handoff.Commit, status)
					}
				}
				appendEvent(ws, "completion_relay_"+status, message, ws.Handoff.Commit)
				return nil
			}); err != nil {
				return err
			}
			p.AppendLog(fmt.Sprintf("relay name=%s status=%s", args[0], status))
			ui.Success("Recorded completion relay for %q as %s", args[0], status)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "delivery status: attempted, delivered, failed, or unavailable (required)")
	cmd.Flags().StringVarP(&message, "message", "m", "", "delivery summary (required)")
	return cmd
}

func hasRelayAttempt(ws *workspace.Workspace, handoffCommit string) bool {
	for i := len(ws.Events) - 1; i >= 0; i-- {
		event := ws.Events[i]
		if event.Type == "completion_relay_attempted" && event.Commit == handoffCommit {
			return true
		}
	}
	return false
}
