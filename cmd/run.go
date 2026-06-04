package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/asam264/agentspace/internal/ui"
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
			absPath := filepath.Join(p.Root, filepath.FromSlash(ws.Path))
			ctx := buildContext(p, ws)

			if agent == "" {
				// Just print context and how to start manually.
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

// launchAgent starts the given agent CLI in the workspace dir, injecting context.
// It tries passing context as a prompt argument; if the binary is missing it
// writes AGENT_CONTEXT.md and instructs the user.
func launchAgent(dir, agent, ctx string) error {
	if _, err := exec.LookPath(agent); err != nil {
		return writeContextFile(dir, ctx, fmt.Sprintf("%q not found on PATH", agent))
	}

	// claude and codex both accept an initial prompt as a positional argument.
	c := exec.Command(agent, ctx)
	c.Dir = dir
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	ui.Success("Launching %s in %s", agent, dir)
	if err := c.Run(); err != nil {
		// Fall back to context file if the invocation form was rejected.
		return writeContextFile(dir, ctx, fmt.Sprintf("%s exited: %v", agent, err))
	}
	return nil
}

// writeContextFile drops AGENT_CONTEXT.md and tells the user to read it.
func writeContextFile(dir, ctx, reason string) error {
	path := filepath.Join(dir, "AGENT_CONTEXT.md")
	if err := os.WriteFile(path, []byte(ctx), 0o644); err != nil {
		return err
	}
	ui.Warn("%s; wrote context to %s", reason, path)
	ui.Plain("请手动启动 agent 并让其阅读该文件。")
	return nil
}
