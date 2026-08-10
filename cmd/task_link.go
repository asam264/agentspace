package cmd

import (
	"fmt"
	"strings"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newLinkTaskCmd() *cobra.Command {
	var taskID, title, taskStatus, masterTaskID, masterHostID string
	var masterRelaySupported bool
	var replace bool
	cmd := &cobra.Command{
		Use:   "link-task <name>",
		Short: "Link a workspace to its user-owned Codex Worker task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID = strings.TrimSpace(taskID)
			title = strings.TrimSpace(title)
			taskStatus = strings.TrimSpace(taskStatus)
			masterTaskID = strings.TrimSpace(masterTaskID)
			masterHostID = strings.TrimSpace(masterHostID)
			if taskID == "" || title == "" {
				return fmt.Errorf("--task-id and --title are required")
			}
			if masterHostID != "" && masterTaskID == "" {
				return fmt.Errorf("--master-host-id requires --master-task-id")
			}
			if masterRelaySupported && masterTaskID == "" {
				return fmt.Errorf("--master-relay-supported requires --master-task-id")
			}
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			var updated workspace.Workspace
			if err := p.WithLock(func(s *workspace.Store) error {
				ws := s.Find(args[0])
				if ws == nil {
					return fmt.Errorf("workspace %q not found", args[0])
				}
				if ws.Task.DispatchedAt == "" {
					return fmt.Errorf("workspace %q must be dispatched before linking a Worker task", ws.Name)
				}
				expectedTitlePrefix := fmt.Sprintf("【#%s】", ws.Name)
				if !strings.HasPrefix(title, expectedTitlePrefix) {
					return fmt.Errorf("--title must start with %q", expectedTitlePrefix)
				}
				if strings.TrimSpace(strings.TrimPrefix(title, expectedTitlePrefix)) == "" {
					return fmt.Errorf("--title must include a work title after %q", expectedTitlePrefix)
				}
				if ws.WorkerTask != nil && ws.WorkerTask.ID != taskID && !replace {
					return fmt.Errorf("workspace %q is already linked to task %q; use --replace after confirming the old task is no longer active", ws.Name, ws.WorkerTask.ID)
				}
				linkedAt := workspace.TimestampNow()
				if ws.WorkerTask != nil && ws.WorkerTask.ID == taskID {
					linkedAt = ws.WorkerTask.LinkedAt
				}
				ws.WorkerTask = &workspace.WorkerTask{
					ID:              taskID,
					Title:           title,
					LinkedAt:        linkedAt,
					LastKnownStatus: taskStatus,
				}
				if taskStatus != "" {
					ws.WorkerTask.ObservedAt = workspace.TimestampNow()
				}
				if masterTaskID != "" {
					ws.Master = &workspace.MasterEndpoint{TaskID: masterTaskID, HostID: masterHostID, RelaySupported: masterRelaySupported}
					appendEvent(ws, "master_endpoint_set", fmt.Sprintf("%s relay_supported=%t", masterTaskID, masterRelaySupported), "")
				}
				appendEvent(ws, "task_linked", title, "")
				updated = *ws
				return nil
			}); err != nil {
				return err
			}
			if executionKind(&updated) == workspace.ExecutionAgentSpace {
				wsPath, err := executionPath(p, &updated)
				if err != nil {
					return err
				}
				if err := writeDispatchContext(wsPath, buildContext(p, &updated)); err != nil {
					return err
				}
			}
			p.AppendLog(fmt.Sprintf("link-task name=%s task=%s", updated.Name, taskID))
			ui.Success("Linked %q to Worker task %q", updated.Name, title)
			return nil
		},
	}
	cmd.Flags().StringVar(&taskID, "task-id", "", "Codex Worker task identity (required)")
	cmd.Flags().StringVar(&title, "title", "", "Worker task title (required)")
	cmd.Flags().StringVar(&taskStatus, "task-status", "", "last observed task status")
	cmd.Flags().StringVar(&masterTaskID, "master-task-id", "", "Master task identity for Worker completion relays")
	cmd.Flags().StringVar(&masterHostID, "master-host-id", "", "Master task host identity for Worker completion relays")
	cmd.Flags().BoolVar(&masterRelaySupported, "master-relay-supported", false, "Master endpoint accepts Completion Relay task messages")
	cmd.Flags().BoolVar(&replace, "replace", false, "replace an existing task link after confirming the old task is inactive")
	return cmd
}

func newObserveTaskCmd() *cobra.Command {
	var taskStatus, message string
	cmd := &cobra.Command{
		Use:   "observe-task <name>",
		Short: "Record a Master observation about a linked Worker task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskStatus = strings.TrimSpace(taskStatus)
			message = strings.TrimSpace(message)
			if taskStatus == "" || message == "" {
				return fmt.Errorf("--task-status and --message/-m are required")
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
				if !hasWorkerTask(ws) {
					return fmt.Errorf("workspace %q has no linked Worker task", ws.Name)
				}
				ws.WorkerTask.LastKnownStatus = taskStatus
				ws.WorkerTask.ObservedAt = workspace.TimestampNow()
				appendEvent(ws, "task_observed", message, "")
				return nil
			}); err != nil {
				return err
			}
			ui.Success("Recorded Worker task observation for %q", args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&taskStatus, "task-status", "", "observed Codex task status (required)")
	cmd.Flags().StringVarP(&message, "message", "m", "", "observation summary (required)")
	return cmd
}
