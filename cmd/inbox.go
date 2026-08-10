package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

type inboxItem struct {
	Name       string `json:"name"`
	Category   string `json:"category"`
	Status     string `json:"status"`
	TaskID     string `json:"task_id,omitempty"`
	TaskTitle  string `json:"task_title,omitempty"`
	NextAction string `json:"next_action"`
}

func newInboxCmd() *cobra.Command {
	var asJSON, includeTerminal bool
	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Group workspaces by the next Master or User Owner action",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			store, err := p.LoadStore()
			if err != nil {
				return err
			}
			items := make([]inboxItem, 0, len(store.Workspaces))
			for i := range store.Workspaces {
				ws := &store.Workspaces[i]
				if !includeTerminal && isTerminalStatus(ws.Status) {
					continue
				}
				item := inboxItem{Name: ws.Name, Category: inboxCategory(store, ws), Status: ws.Status, NextAction: nextAction(store, ws)}
				if ws.WorkerTask != nil {
					item.TaskID = ws.WorkerTask.ID
					item.TaskTitle = ws.WorkerTask.Title
				}
				items = append(items, item)
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(items)
			}
			if len(items) == 0 {
				fmt.Println("No workspaces in the inbox.")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "CATEGORY\tNAME\tSTATUS\tTASK\tNEXT ACTION")
			for _, item := range items {
				task := item.TaskTitle
				if task == "" {
					task = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", item.Category, item.Name, item.Status, task, item.NextAction)
			}
			return w.Flush()
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output inbox as JSON")
	cmd.Flags().BoolVar(&includeTerminal, "all", false, "include merged, cancelled, and failed workspaces")
	return cmd
}

func inboxCategory(store *workspace.Store, ws *workspace.Workspace) string {
	if isTerminalStatus(ws.Status) {
		return "terminal"
	}
	if ws.Status == workspace.StatusPaused {
		return "paused"
	}
	if executionKind(ws) == workspace.ExecutionCodex && !hasWorkerTask(ws) {
		return "needs_link"
	}
	if !hasAttachedExecution(ws) {
		return "needs_attach"
	}
	if hasPendingOverride(ws) {
		return "needs_attention"
	}
	switch ws.Status {
	case workspace.StatusSubmitted:
		return "needs_review"
	case workspace.StatusChangesReq:
		return "changes_requested"
	case workspace.StatusAccepted:
		if len(dependencyBlockers(store, ws)) > 0 {
			return "blocked"
		}
		return "ready_to_merge"
	case workspace.StatusActive:
		if ws.Task.DispatchedAt == "" {
			return "ready_to_dispatch"
		}
		return "working"
	default:
		return "needs_attention"
	}
}

func isTerminalStatus(status string) bool {
	return status == workspace.StatusMerged || status == workspace.StatusCancelled || status == workspace.StatusFailed
}
