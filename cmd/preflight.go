package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

type preflightReport struct {
	Workspace string   `json:"workspace"`
	Ready     bool     `json:"ready"`
	Blockers  []string `json:"blockers"`
	Overlaps  []string `json:"overlaps"`
}

func newPreflightCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "preflight <name>",
		Short: "Check dependency order and declared file-scope overlap before merge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			store, err := p.LoadStore()
			if err != nil {
				return err
			}
			ws := store.Find(args[0])
			if ws == nil {
				return fmt.Errorf("workspace %q not found", args[0])
			}
			report := buildPreflightReport(store, ws)
			if asJSON {
				if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
					return err
				}
			} else {
				printPreflightReport(report)
			}
			if !report.Ready {
				return fmt.Errorf("workspace %q is not ready to merge", ws.Name)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output preflight report as JSON")
	return cmd
}

func buildPreflightReport(store *workspace.Store, ws *workspace.Workspace) preflightReport {
	report := preflightReport{Workspace: ws.Name, Ready: true, Blockers: []string{}, Overlaps: scopeOverlapWarnings(store, ws)}
	if ws.Status != workspace.StatusAccepted {
		report.Blockers = append(report.Blockers, fmt.Sprintf("workspace status is %s, not accepted", ws.Status))
	}
	report.Blockers = append(report.Blockers, dependencyBlockers(store, ws)...)
	report.Ready = len(report.Blockers) == 0
	return report
}

func scopeOverlapWarnings(store *workspace.Store, ws *workspace.Workspace) []string {
	warnings := make([]string, 0)
	for i := range store.Workspaces {
		other := &store.Workspaces[i]
		if other.Name == ws.Name || !isOpenWorkspaceStatus(other.Status) {
			continue
		}
		for _, left := range ws.Task.FileScope {
			for _, right := range other.Task.FileScope {
				if scopeOverlaps(left, right) {
					warnings = append(warnings, fmt.Sprintf("%s overlaps %s scope %q", left, other.Name, right))
				}
			}
		}
	}
	return warnings
}

func printPreflightReport(report preflightReport) {
	if report.Ready {
		ui.Success("%q is ready to merge", report.Workspace)
	} else {
		ui.Error("%q is not ready to merge", report.Workspace)
		for _, blocker := range report.Blockers {
			ui.Plain("  blocker: %s", blocker)
		}
	}
	for _, warning := range report.Overlaps {
		ui.Warn("scope overlap: %s", warning)
	}
}
