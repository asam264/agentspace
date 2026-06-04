package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a workspace (worktree + branch + metadata)",
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
			if err := removeWorkspace(p, ws); err != nil {
				return err
			}
			ui.Success("Removed workspace %q", name)
			return nil
		},
	}
	return cmd
}

// removeWorkspace removes the worktree, deletes the branch, and drops metadata.
func removeWorkspace(p workspace.Paths, ws *workspace.Workspace) error {
	absPath := filepath.Join(p.Root, filepath.FromSlash(ws.Path))
	if _, err := git.WorktreeRemove(p.Root, absPath); err != nil {
		return err
	}
	if git.BranchExists(p.Root, ws.Branch) {
		if _, err := git.BranchDelete(p.Root, ws.Branch); err != nil {
			return err
		}
	}
	name := ws.Name
	if err := p.WithLock(func(s *workspace.Store) error {
		if !s.Remove(name) {
			return fmt.Errorf("workspace %q not found", name)
		}
		return nil
	}); err != nil {
		return err
	}
	p.AppendLog("remove name=" + name)
	return nil
}
