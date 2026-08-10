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
		Short: "Prepare a Worker context and optionally start an external runner",
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
			wsPath := filepath.Join(p.Root, filepath.FromSlash(ws.Path))
			if err := writeDispatchContext(wsPath, buildContext(p, ws)); err != nil {
				return err
			}
			if err := p.WithLock(func(s *workspace.Store) error {
				cur := s.Find(ws.Name)
				if cur == nil || (cur.Status != workspace.StatusActive && cur.Status != workspace.StatusChangesReq) {
					return fmt.Errorf("workspace %q is no longer dispatchable", ws.Name)
				}
				cur.Task.DispatchedAt = workspace.TimestampNow()
				appendEvent(cur, "dispatched", runner, "")
				return nil
			}); err != nil {
				return err
			}
			p.AppendLog("dispatch name=" + ws.Name)
			if strings.TrimSpace(runner) == "" {
				ui.Success("Prepared Worker context for %q", ws.Name)
				ui.Plain("  path: %s", wsPath)
				ui.Plain("  Master can now spawn a Codex Worker for this path.")
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
