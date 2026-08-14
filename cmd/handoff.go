package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newHandoffCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "handoff <name>",
		Short: "Show the worker handoff submitted for master review",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			_, ws, err := loadWorkspace(p, args[0])
			if err != nil {
				return err
			}
			if ws.Handoff == nil {
				return fmt.Errorf("workspace %q has no submitted handoff", ws.Name)
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(ws.Handoff)
			}
			printHandoff(ws.Handoff)
			if reviewSpace := reviewWorkspaceForCommit(ws, ws.Handoff.Commit); reviewSpace != nil {
				ui.Plain("Review workspace: %s", reviewSpace.Path)
			} else {
				ui.Plain("Review in GoLand: agentspace review %s --open", ws.Name)
			}
			if ws.Review != nil {
				ui.Bold("Review:")
				ui.Plain("  %s at %s", ws.Review.Decision, formatTimestamp(ws.Review.ReviewedAt))
				ui.Plain("  %s", ws.Review.Summary)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output handoff as JSON")
	return cmd
}

func printHandoff(handoff *workspace.Handoff) {
	ui.Bold("Handoff:")
	ui.Plain("  Commit:    %s", handoff.Commit)
	ui.Plain("  Submitted: %s", formatTimestamp(handoff.SubmittedAt))
	ui.Plain("  Summary:   %s", handoff.Summary)
	ui.Plain("  Files:     %s", orDash(strings.Join(handoff.Files, ", ")))
	if len(handoff.Checks) == 0 {
		ui.Plain("  Checks:    (none recorded)")
		return
	}
	ui.Plain("  Checks:")
	for _, check := range handoff.Checks {
		state := "passed"
		if !check.Passed {
			state = "failed"
		}
		ui.Plain("    [%s] %s", state, check.Command)
	}
}

func formatTimestamp(value string) string {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.Local().Format("2006-01-02 15:04:05")
	}
	return value
}
