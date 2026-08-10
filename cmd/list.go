package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all workspaces",
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

			if asJSON {
				data, err := json.MarshalIndent(store, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			if len(store.Workspaces) == 0 {
				ui.Warn("no workspaces yet (create one with 'agentspace new <name>')")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tDESCRIPTION\tBASE\tSTATUS\tDEPS\tSCOPE\tPATH")
			for _, ws := range store.Workspaces {
				desc := ws.Description
				if desc == "" {
					desc = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					ws.Name, desc, ws.BaseBranch, ws.Status, orDash(strings.Join(ws.Task.DependsOn, ",")), orDash(strings.Join(ws.Task.FileScope, ",")), ws.Path)
			}
			return w.Flush()
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

// loadWorkspace is a small helper used by several commands.
func loadWorkspace(p workspace.Paths, name string) (*workspace.Store, *workspace.Workspace, error) {
	store, err := p.LoadStore()
	if err != nil {
		return nil, nil, err
	}
	ws := store.Find(name)
	if ws == nil {
		return nil, nil, fmt.Errorf("workspace %q not found", name)
	}
	return store, ws, nil
}
