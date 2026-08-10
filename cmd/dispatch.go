package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newDispatchCmd() *cobra.Command {
	var runner string
	cmd := &cobra.Command{
		Use:   "dispatch <name>",
		Short: "Prepare a Worker task packet and optionally start an external runner",
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
			if ws.Status != workspace.StatusActive && ws.Status != workspace.StatusChangesReq {
				return fmt.Errorf("workspace %q cannot be dispatched while status is %s", ws.Name, ws.Status)
			}
			wsPath := ""
			if executionKind(ws) == workspace.ExecutionAgentSpace {
				wsPath, err = executionPath(p, ws)
				if err != nil {
					return err
				}
				if err := writeDispatchContext(wsPath, buildContext(p, ws)); err != nil {
					return err
				}
			} else if strings.TrimSpace(runner) != "" {
				return fmt.Errorf("--runner is only supported for agentspace execution; Codex Workers attach their own worktree")
			}
			eventType := "dispatch_prepared"
			if strings.TrimSpace(runner) != "" {
				eventType = "runner_started"
			}
			if err := p.WithLock(func(s *workspace.Store) error {
				cur := s.Find(ws.Name)
				if cur == nil || (cur.Status != workspace.StatusActive && cur.Status != workspace.StatusChangesReq) {
					return fmt.Errorf("workspace %q is no longer dispatchable", ws.Name)
				}
				cur.Task.DispatchedAt = workspace.TimestampNow()
				appendEvent(cur, eventType, runner, "")
				return nil
			}); err != nil {
				return err
			}
			p.AppendLog("dispatch name=" + ws.Name)
			if strings.TrimSpace(runner) == "" {
				ui.Success("Prepared Worker task packet for %q", ws.Name)
				if executionKind(ws) == workspace.ExecutionCodex {
					ui.Plain("  Create a user-owned Codex Worker worktree and task, then run link-task and attach from that Worker directory.")
				} else {
					ui.Plain("  path: %s", wsPath)
					ui.Plain("  Create a user-owned Codex Worker task, then run: agentspace link-task %s --task-id <id> --title <title>", ws.Name)
				}
				return nil
			}
			ui.Success("Starting runner for %q in %s", ws.Name, wsPath)
			runnerCmd := shellCommand(runner)
			runnerCmd.Dir = wsPath
			runnerCmd.Stdin = os.Stdin
			runnerCmd.Stdout = os.Stdout
			runnerCmd.Stderr = os.Stderr
			if err := runnerCmd.Run(); err != nil {
				_ = recordEvent(p, ws.Name, "runner_failed", err.Error(), "")
				return fmt.Errorf("runner exited: %w", err)
			}
			return recordEvent(p, ws.Name, "runner_exited", "", "")
		},
	}
	cmd.Flags().StringVar(&runner, "runner", "", "external command to run in the workspace after writing AGENT_CONTEXT.md")
	return cmd
}

func writeDispatchContext(dir, context string) error {
	_ = excludeAgentContext(dir)
	return os.WriteFile(filepath.Join(dir, "AGENT_CONTEXT.md"), []byte(context), 0o644)
}
