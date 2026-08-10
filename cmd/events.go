package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/spf13/cobra"
)

func newEventsCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "events <name>",
		Short: "Show append-only workflow events for a workspace",
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
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(ws.Events)
			}
			if len(ws.Events) == 0 {
				ui.Plain("(no events)")
				return nil
			}
			for _, event := range ws.Events {
				fmt.Printf("%s  %s", formatTimestamp(event.At), event.Type)
				if event.Commit != "" {
					fmt.Printf("  %s", event.Commit)
				}
				if event.Summary != "" {
					fmt.Printf("  %s", event.Summary)
				}
				fmt.Println()
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output events as JSON")
	return cmd
}
