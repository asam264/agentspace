package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newNewCmd() *cobra.Command {
	var from, desc, promptStr, promptFile string
	var acceptance, scope, dependencies []string
	var editFlag bool
	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new workspace (git worktree + branch)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("workspace name cannot be empty")
			}
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
			acceptance, err = normalizeTextList(acceptance, "acceptance criterion")
			if err != nil {
				return err
			}
			scope, err = normalizeScope(scope)
			if err != nil {
				return err
			}
			dependencies, err = normalizeTextList(dependencies, "dependency")
			if err != nil {
				return err
			}
			for _, dependency := range dependencies {
				if dependency == name {
					return fmt.Errorf("workspace %q cannot depend on itself", name)
				}
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
				for _, dependency := range dependencies {
					if s.Find(dependency) == nil {
						return fmt.Errorf("dependency workspace %q does not exist", dependency)
					}
				}
				if _, err := git.WorktreeAdd(p.Root, absPath, branch, fromRef); err != nil {
					return err
				}
				createdAt := workspace.Timestamp(time.Now())
				ws := workspace.Workspace{
					Name:        name,
					Description: desc,
					Prompt:      prompt,
					Branch:      branch,
					BaseCommit:  baseCommit,
					BaseBranch:  fromRef,
					Status:      workspace.StatusActive,
					CreatedAt:   createdAt,
					Path:        relPath,
					Snapshots:   []workspace.Snapshot{},
					Task: workspace.TaskManifest{
						AcceptanceCriteria: acceptance,
						FileScope:          scope,
						DependsOn:          dependencies,
					},
				}
				appendEvent(&ws, "created", "workspace created", "")
				s.Workspaces = append(s.Workspaces, ws)
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
	cmd.Flags().StringArrayVar(&acceptance, "acceptance", nil, "acceptance criterion (repeatable)")
	cmd.Flags().StringArrayVar(&scope, "scope", nil, "repository-relative file or directory path owned by this workspace (repeatable)")
	cmd.Flags().StringArrayVar(&dependencies, "depends-on", nil, "workspace that must merge before this one (repeatable)")
	return cmd
}
