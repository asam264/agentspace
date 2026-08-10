package cmd

import (
	"fmt"

	"github.com/asam264/agentspace/internal/git"
	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	var vs string
	cmd := &cobra.Command{
		Use:   "diff <name>",
		Short: "Show the diff of a workspace vs its source commit or another workspace",
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
			wsPath, err := executionPath(p, ws)
			if err != nil {
				return err
			}

			// Determine the comparison ref.
			target := vs
			if target == "" {
				target = ws.BaseCommit
			}

			var compareRef string
			store, err := p.LoadStore()
			if err != nil {
				return err
			}
			if other := store.Find(target); other != nil {
				if other.Handoff != nil && other.Handoff.Commit != "" {
					compareRef = other.Handoff.Commit
				} else if other.Branch != "" {
					compareRef = other.Branch
				} else {
					return fmt.Errorf("workspace %q has no attached handoff commit to compare", other.Name)
				}
			} else {
				// Treat target as a branch/commit ref.
				compareRef = target
			}

			// git -C <wsPath> diff --color <compareRef>...HEAD
			// Use two-dot diff against the ref so it shows ws changes relative to it.
			err = git.RunPassthrough(wsPath, "diff", "--color=auto", compareRef)
			if err != nil {
				return fmt.Errorf("git diff failed: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&vs, "vs", "", "compare against this branch/commit or another workspace name (default: source commit)")
	return cmd
}
