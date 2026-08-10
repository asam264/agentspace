package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

type resumeItem struct {
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	NextAction string   `json:"next_action"`
	DependsOn  []string `json:"depends_on"`
}

func newResumeCmd() *cobra.Command {
	var asJSON, includeTerminal bool
	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Show the next Master action for unfinished workspaces",
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
			items := make([]resumeItem, 0, len(store.Workspaces))
			for i := range store.Workspaces {
				ws := &store.Workspaces[i]
				if !includeTerminal && (ws.Status == workspace.StatusMerged || ws.Status == workspace.StatusCancelled || ws.Status == workspace.StatusFailed) {
					continue
				}
				items = append(items, resumeItem{Name: ws.Name, Status: ws.Status, NextAction: nextAction(store, ws), DependsOn: ws.Task.DependsOn})
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(items)
			}
			if len(items) == 0 {
				fmt.Println("No unfinished workspaces.")
				return nil
			}
			for _, item := range items {
				fmt.Printf("%s [%s]: %s\n", item.Name, item.Status, item.NextAction)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output resume queue as JSON")
	cmd.Flags().BoolVar(&includeTerminal, "all", false, "include merged, cancelled, and failed workspaces")
	return cmd
}

func nextAction(store *workspace.Store, ws *workspace.Workspace) string {
	switch ws.Status {
	case workspace.StatusActive:
		return "dispatch a Worker: agentspace dispatch " + ws.Name
	case workspace.StatusChangesReq:
		return "send review feedback to the Worker, then submit a new handoff"
	case workspace.StatusSubmitted:
		return "review handoff: agentspace handoff " + ws.Name
	case workspace.StatusAccepted:
		if blockers := dependencyBlockers(store, ws); len(blockers) > 0 {
			return "wait for dependencies: " + blockers[0]
		}
		return "preflight and merge: agentspace preflight " + ws.Name
	case workspace.StatusConflicted:
		return "resolve the conflict, then run: agentspace merge --continue " + ws.Name
	case workspace.StatusFailed:
		return "inspect events and decide whether to dispatch a replacement workspace"
	case workspace.StatusCancelled:
		return "no action; create a new workspace if work should resume"
	default:
		return "inspect workspace status"
	}
}
