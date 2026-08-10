package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newAttachCmd() *cobra.Command {
	var taskID string
	cmd := &cobra.Command{
		Use:   "attach <name>",
		Short: "Bind a Codex Worker task to its current worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID = strings.TrimSpace(taskID)
			if taskID == "" {
				return fmt.Errorf("--task-id is required")
			}
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			currentRoot, err := git.RepoRoot("")
			if err != nil {
				return err
			}
			currentRoot = filepath.Clean(currentRoot)
			if samePath(currentRoot, p.Root) {
				return fmt.Errorf("attach must run inside a Codex Worker worktree, not the main worktree")
			}
			commonDir, err := git.CommonDir("")
			if err != nil {
				return err
			}
			mainCommonDir, err := git.CommonDir(p.Root)
			if err != nil {
				return err
			}
			if !samePath(commonDir, mainCommonDir) {
				return fmt.Errorf("current worktree is not part of this AgentSpace repository")
			}
			worktrees, err := git.WorktreePaths(p.Root)
			if err != nil {
				return err
			}
			if !containsPath(worktrees, currentRoot) {
				return fmt.Errorf("current directory is not a registered Git worktree for this repository")
			}
			head, err := git.Run(currentRoot, "rev-parse", "HEAD")
			if err != nil {
				return err
			}

			var updated workspace.Workspace
			if err := p.WithLock(func(s *workspace.Store) error {
				ws := s.Find(args[0])
				if ws == nil {
					return fmt.Errorf("workspace %q not found", args[0])
				}
				if executionKind(ws) != workspace.ExecutionCodex {
					return fmt.Errorf("workspace %q uses %q execution and cannot attach a Codex worktree", ws.Name, executionKind(ws))
				}
				if !hasWorkerTask(ws) || ws.WorkerTask.ID != taskID {
					return fmt.Errorf("--task-id does not match the linked Worker task for %q", ws.Name)
				}
				if ws.Execution.Path != "" && !samePath(ws.Execution.Path, currentRoot) {
					return fmt.Errorf("workspace %q is already attached to %s", ws.Name, ws.Execution.Path)
				}
				boundHead, err := git.RevParseShort(currentRoot, "HEAD")
				if err != nil {
					return err
				}
				if boundHead != ws.BaseCommit {
					return fmt.Errorf("current worktree HEAD must equal workspace source commit %s before attach (got %s)", ws.BaseCommit, boundHead)
				}
				ws.Execution.Path = currentRoot
				ws.Execution.CommonDir = commonDir
				ws.Execution.AttachedAt = workspace.TimestampNow()
				ws.Execution.BoundHead = head
				ws.Execution.TaskID = taskID
				appendEvent(ws, "execution_attached", currentRoot, head)
				updated = *ws
				return nil
			}); err != nil {
				return err
			}
			if err := writeDispatchContext(currentRoot, buildContext(p, &updated)); err != nil {
				return err
			}
			p.AppendLog(fmt.Sprintf("attach name=%s task=%s path=%s", updated.Name, taskID, currentRoot))
			ui.Success("Attached Codex Worker worktree for %q", updated.Name)
			ui.Plain("  path: %s", currentRoot)
			return nil
		},
	}
	cmd.Flags().StringVar(&taskID, "task-id", "", "linked Codex Worker task identity (required)")
	return cmd
}

func samePath(a, b string) bool {
	a, _ = filepath.Abs(a)
	b, _ = filepath.Abs(b)
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func containsPath(paths []string, want string) bool {
	for _, path := range paths {
		if samePath(path, want) {
			return true
		}
	}
	return false
}
