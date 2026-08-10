package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

const maxCheckOutput = 4096

func newSubmitCmd() *cobra.Command {
	var message string
	var checks []string
	cmd := &cobra.Command{
		Use:   "submit <name>",
		Short: "Commit a worker result and submit it for master review",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(message) == "" {
				return fmt.Errorf("--message/-m is required")
			}
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			_, ws, err := loadWorkspace(p, args[0])
			if err != nil {
				return err
			}
			if ws.Status != workspace.StatusActive && ws.Status != workspace.StatusChangesReq {
				return fmt.Errorf("workspace %q cannot be submitted while status is %s", ws.Name, ws.Status)
			}

			wsPath := filepath.Join(p.Root, filepath.FromSlash(ws.Path))
			if err := commitWorkspaceChanges(wsPath, message); err != nil {
				return err
			}
			results, err := runChecks(wsPath, checks)
			if err != nil {
				return err
			}
			if dirty, err := git.Run(wsPath, "status", "--porcelain"); err != nil {
				return err
			} else if dirty != "" {
				return fmt.Errorf("checks left uncommitted changes in workspace %q; clean them and submit again:\n%s", ws.Name, dirty)
			}

			head, err := git.Run(wsPath, "rev-parse", "HEAD")
			if err != nil {
				return err
			}
			files, err := changedFiles(wsPath, ws.BaseCommit, head)
			if err != nil {
				return err
			}
			handoff := &workspace.Handoff{
				Commit:      head,
				Summary:     strings.TrimSpace(message),
				Files:       files,
				Checks:      results,
				SubmittedAt: workspace.Timestamp(time.Now()),
			}
			if err := p.WithLock(func(s *workspace.Store) error {
				cur := s.Find(ws.Name)
				if cur == nil {
					return fmt.Errorf("workspace %q not found", ws.Name)
				}
				if cur.Status != workspace.StatusActive && cur.Status != workspace.StatusChangesReq {
					return fmt.Errorf("workspace %q changed status to %s before submission", cur.Name, cur.Status)
				}
				cur.Handoff = handoff
				cur.Review = nil
				cur.Status = workspace.StatusSubmitted
				appendEvent(cur, "submitted", handoff.Summary, head)
				return nil
			}); err != nil {
				return err
			}
			p.AppendLog(fmt.Sprintf("submit name=%s commit=%s", ws.Name, head))
			ui.Success("Submitted %q for review at %s", ws.Name, head)
			ui.Plain("  changed files: %d; checks: %d", len(files), len(results))
			return nil
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "handoff summary (required)")
	cmd.Flags().StringArrayVar(&checks, "test", nil, "test command to run before submission (repeatable)")
	return cmd
}

func commitWorkspaceChanges(dir, message string) error {
	dirty, err := git.Run(dir, "status", "--porcelain")
	if err != nil {
		return err
	}
	if dirty == "" {
		return nil
	}
	if _, err := git.Run(dir, "add", "-A"); err != nil {
		return err
	}
	if _, err := git.Run(dir, "diff", "--cached", "--quiet"); err == nil {
		return nil
	}
	_, err = git.Run(dir, "commit", "--no-verify", "-m", "[handoff] "+strings.TrimSpace(message))
	return err
}

func runChecks(dir string, commands []string) ([]workspace.CheckResult, error) {
	results := make([]workspace.CheckResult, 0, len(commands))
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			return nil, fmt.Errorf("--test cannot be empty")
		}
		c := shellCommand(command)
		c.Dir = dir
		output, err := c.CombinedOutput()
		result := workspace.CheckResult{Command: command, Passed: err == nil, Output: truncateOutput(string(output))}
		results = append(results, result)
		if err != nil {
			return nil, fmt.Errorf("check failed: %s\n%s", command, result.Output)
		}
	}
	return results, nil
}

func shellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", command)
	}
	return exec.Command("sh", "-c", command)
}

func truncateOutput(output string) string {
	output = strings.TrimSpace(output)
	if len(output) <= maxCheckOutput {
		return output
	}
	return output[:maxCheckOutput] + "\n... (truncated)"
}

func changedFiles(dir, baseCommit, head string) ([]string, error) {
	output, err := git.Run(dir, "diff", "--name-only", baseCommit+".."+head)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(output, "\n"), nil
}
