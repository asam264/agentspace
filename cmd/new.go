package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newNewCmd() *cobra.Command {
	var from, desc, promptStr, promptFile string
	var editFlag bool
	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new workspace (git worktree + branch)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			p, cfg, err := requireInit()
			if err != nil {
				return err
			}

			// Resolve prompt text.
			prompt, err := resolvePrompt(promptStr, promptFile, editFlag, cfg.Editor)
			if err != nil {
				return err
			}

			// Derive description from prompt if --desc not given.
			if desc == "" && prompt != "" {
				desc = descFromPrompt(prompt)
			}

			// Resolve the base ref.
			fromRef := from
			if fromRef == "" {
				fromRef = cfg.BaseBranch
			}
			baseCommit, err := git.RevParseShort(p.Root, fromRef)
			if err != nil {
				return err
			}

			branch := "agentspace/" + name
			relPath := filepath.ToSlash(filepath.Join(workspace.Dir, "workspaces", name))
			absPath := filepath.Join(p.WorkspacesDir, name)

			err = p.WithLock(func(s *workspace.Store) error {
				if s.Find(name) != nil {
					return fmt.Errorf("workspace %q already exists", name)
				}
				if git.BranchExists(p.Root, branch) {
					return fmt.Errorf("branch %q already exists", branch)
				}
				if _, err := git.WorktreeAdd(p.Root, absPath, branch, fromRef); err != nil {
					return err
				}
				s.Workspaces = append(s.Workspaces, workspace.Workspace{
					Name:        name,
					Description: desc,
					Prompt:      prompt,
					Branch:      branch,
					BaseCommit:  baseCommit,
					BaseBranch:  fromRef,
					Status:      workspace.StatusActive,
					CreatedAt:   workspace.Timestamp(time.Now()),
					Path:        relPath,
					Snapshots:   []workspace.Snapshot{},
				})
				return nil
			})
			if err != nil {
				return err
			}

			p.AppendLog(fmt.Sprintf("new name=%s from=%s commit=%s", name, fromRef, baseCommit))
			ui.Success("Created workspace %q on branch %s (base %s)", name, branch, baseCommit)
			ui.Plain("  path: %s", absPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "branch or commit to base the workspace on (default: config base_branch)")
	cmd.Flags().StringVar(&desc, "desc", "", "short description shown in list (defaults to first line of prompt)")
	cmd.Flags().StringVar(&promptStr, "prompt", "", "full task description")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "read full task description from file")
	cmd.Flags().BoolVar(&editFlag, "edit", false, "open editor to write task description interactively")
	return cmd
}
