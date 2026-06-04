package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/spf13/cobra"
)

func newSnapshotsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshots <name>",
		Short: "List all snapshots of a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			_, ws, err := loadWorkspace(p, name)
			if err != nil {
				return err
			}
			if len(ws.Snapshots) == 0 {
				ui.Warn("no snapshots for %q", name)
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tCOMMIT\tMESSAGE\tTIME")
			for _, s := range ws.Snapshots {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.Commit, s.Message, s.CreatedAt)
			}
			return w.Flush()
		},
	}
	return cmd
}
