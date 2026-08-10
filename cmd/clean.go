package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newCleanCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove all merged workspaces",
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

			var merged []workspace.Workspace
			for _, ws := range store.Workspaces {
				if ws.Status == workspace.StatusMerged {
					merged = append(merged, ws)
				}
			}
			if len(merged) == 0 {
				ui.Warn("no merged workspaces to clean")
				return nil
			}

			ui.Bold("Merged workspaces to remove:")
			for _, ws := range merged {
				ui.Plain("  %s (%s)", ws.Name, executionKind(&ws))
			}

			if !yes {
				fmt.Printf("Remove these %d workspace(s)? [y/N]: ", len(merged))
				ans, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				if strings.ToLower(strings.TrimSpace(ans)) != "y" {
					ui.Plain("Aborted.")
					return nil
				}
			}

			for i := range merged {
				ws := merged[i]
				if err := removeWorkspace(p, &ws); err != nil {
					ui.Error("failed to remove %q: %v", ws.Name, err)
					continue
				}
				ui.Success("Removed %q", ws.Name)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
