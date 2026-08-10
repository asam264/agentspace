package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <name> [agent]",
		Short: "Launch an AI agent (claude|codex) in a workspace with context",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			agent := ""
			if len(args) == 2 {
				agent = args[1]
			}
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			_, ws, err := loadWorkspace(p, name)
			if err != nil {
				return err
			}
			if executionKind(ws) != workspace.ExecutionAgentSpace {
				return fmt.Errorf("run is only supported for agentspace execution; create a Codex Worker task for %q instead", ws.Name)
			}
			absPath, err := executionPath(p, ws)
			if err != nil {
				return err
			}
			ctx := buildContext(p, ws)

			if agent == "" {
				fmt.Print(ctx)
				ui.Plain("\n手动启动：cd %s 然后运行 claude 或 codex", absPath)
				return nil
			}

			switch agent {
			case "claude", "codex":
				return launchAgent(absPath, agent, ctx)
			default:
				return fmt.Errorf("unknown agent %q (use 'claude' or 'codex')", agent)
			}
		},
	}
	return cmd
}

// launchAgent writes context to AGENT_CONTEXT.md, then launches the agent with
// a short trigger prompt so the agent reads the file and starts working.
func launchAgent(dir, agent, ctx string) error {
	if _, err := exec.LookPath(agent); err != nil {
		return writeContextFile(dir, ctx, fmt.Sprintf("%q not found on PATH", agent))
	}

	ctxFile := filepath.Join(dir, "AGENT_CONTEXT.md")
	_ = excludeAgentContext(dir)
	if err := os.WriteFile(ctxFile, []byte(ctx), 0o644); err != nil {
		return err
	}

	// Short trigger prompt — agent reads the file and starts working immediately.
	trigger := fmt.Sprintf("请阅读工作目录下的 AGENT_CONTEXT.md，然后按其中【任务说明】立即开始实现，无需等待进一步指令。")

	c := exec.Command(agent, trigger)
	c.Dir = dir
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	ui.Success("Launching %s in %s", agent, dir)
	if err := c.Run(); err != nil {
		return writeContextFile(dir, ctx, fmt.Sprintf("%s exited: %v", agent, err))
	}
	return nil
}

func writeContextFile(dir, ctx, reason string) error {
	path := filepath.Join(dir, "AGENT_CONTEXT.md")
	_ = excludeAgentContext(dir)
	if err := os.WriteFile(path, []byte(ctx), 0o644); err != nil {
		return err
	}
	ui.Warn("%s; wrote context to %s", reason, path)
	ui.Plain("请手动启动 agent 并让其阅读该文件。")
	return nil
}
