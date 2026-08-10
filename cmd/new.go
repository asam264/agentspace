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
	var from, into, execution, desc, promptStr, promptFile string
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
			execution = strings.TrimSpace(execution)
			if execution != workspace.ExecutionAgentSpace && execution != workspace.ExecutionCodex {
				return fmt.Errorf("--execution must be %q or %q", workspace.ExecutionCodex, workspace.ExecutionAgentSpace)
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

			// Resolve the source ref and merge target independently. A plain
			// invocation follows the branch currently checked out in the main
			// worktree rather than the branch recorded at init time.
			fromRef := strings.TrimSpace(from)
			targetBranch := strings.TrimSpace(into)
			if (fromRef == "") != (targetBranch == "") {
				return fmt.Errorf("--from and --into must be supplied together")
			}
			if fromRef == "" || targetBranch == "" {
				currentBranch, err := git.CurrentBranch(p.Root)
				if err != nil {
					return err
				}
				if currentBranch == "HEAD" {
					return fmt.Errorf("main worktree is detached; pass --into <local-branch> and --from <ref> explicitly")
				}
				if fromRef == "" {
					fromRef = currentBranch
				}
				if targetBranch == "" {
					targetBranch = currentBranch
				}
			}
			if !git.BranchExists(p.Root, targetBranch) {
				return fmt.Errorf("--into %q must name an existing local branch", targetBranch)
			}
			baseCommit, err := git.RevParseShort(p.Root, fromRef)
			if err != nil {
				return err
			}

			branch := ""
			relPath := ""
			absPath := ""
			if execution == workspace.ExecutionAgentSpace {
				branch = "agentspace/" + name
				relPath = filepath.ToSlash(filepath.Join(workspace.Dir, "workspaces", name))
				absPath = filepath.Join(p.WorkspacesDir, name)
			}

			err = p.WithLock(func(s *workspace.Store) error {
				if s.Find(name) != nil {
					return fmt.Errorf("workspace %q already exists", name)
				}
				if branch != "" && git.BranchExists(p.Root, branch) {
					return fmt.Errorf("branch %q already exists", branch)
				}
				for _, dependency := range dependencies {
					if s.Find(dependency) == nil {
						return fmt.Errorf("dependency workspace %q does not exist", dependency)
					}
				}
				if execution == workspace.ExecutionAgentSpace {
					if _, err := git.WorktreeAdd(p.Root, absPath, branch, fromRef); err != nil {
						return err
					}
				}
				createdAt := workspace.Timestamp(time.Now())
				ws := workspace.Workspace{
					Name:         name,
					Description:  desc,
					Prompt:       prompt,
					Branch:       branch,
					BaseCommit:   baseCommit,
					BaseBranch:   fromRef,
					TargetBranch: targetBranch,
					Status:       workspace.StatusActive,
					CreatedAt:    createdAt,
					Path:         relPath,
					Snapshots:    []workspace.Snapshot{},
					Task: workspace.TaskManifest{
						AcceptanceCriteria: acceptance,
						FileScope:          scope,
						DependsOn:          dependencies,
					},
					Execution: workspace.Execution{Kind: execution},
				}
				if execution == workspace.ExecutionAgentSpace {
					ws.Execution.Path = absPath
					ws.Execution.AttachedAt = createdAt
					ws.Execution.BoundHead = baseCommit
				}
				appendEvent(&ws, "created", "workspace created", "")
				s.Workspaces = append(s.Workspaces, ws)
				return nil
			})
			if err != nil {
				return err
			}

			p.AppendLog(fmt.Sprintf("new name=%s execution=%s from=%s into=%s commit=%s", name, execution, fromRef, targetBranch, baseCommit))
			if execution == workspace.ExecutionCodex {
				ui.Success("Created Codex workspace manifest %q (source %s -> target %s)", name, fromRef, targetBranch)
				ui.Plain("  Next: create the Codex Worker worktree, link its task, then run agentspace attach %s inside that Worker.", name)
			} else {
				ui.Success("Created workspace %q on branch %s (source %s -> target %s)", name, branch, fromRef, targetBranch)
				ui.Plain("  path: %s", absPath)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "source ref for the workspace (default: current local branch)")
	cmd.Flags().StringVar(&into, "into", "", "existing local branch to receive the merge (default: current local branch)")
	cmd.Flags().StringVar(&execution, "execution", workspace.ExecutionAgentSpace, "execution mode: codex or agentspace")
	cmd.Flags().StringVar(&desc, "desc", "", "short description shown in list (defaults to first line of prompt)")
	cmd.Flags().StringVar(&promptStr, "prompt", "", "full task description")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "read full task description from file")
	cmd.Flags().BoolVar(&editFlag, "edit", false, "open editor to write task description interactively")
	cmd.Flags().StringArrayVar(&acceptance, "acceptance", nil, "acceptance criterion (repeatable)")
	cmd.Flags().StringArrayVar(&scope, "scope", nil, "repository-relative file or directory path owned by this workspace (repeatable)")
	cmd.Flags().StringArrayVar(&dependencies, "depends-on", nil, "workspace that must merge before this one (repeatable)")
	return cmd
}
