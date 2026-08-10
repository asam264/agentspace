package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var branch string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize agentspace in the current git repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := resolvePaths()
			if err != nil {
				return err
			}
			if p.Initialized() {
				// Upgrade repositories initialized by earlier releases, which only
				// excluded the nested worktree directory and left control files dirty.
				if err := excludeAgentspaceState(p.Root); err != nil {
					return err
				}
				if err := excludeAgentContext(p.Root); err != nil {
					return err
				}
				ui.Warn("agentspace already initialized at %s", p.Base)
				return nil
			}

			// Determine base branch: --branch override or current branch.
			baseBranch := branch
			if baseBranch == "" {
				baseBranch, err = git.CurrentBranch(p.Root)
				if err != nil {
					return err
				}
			}

			// Create directory structure.
			for _, dir := range []string{p.Base, p.LogsDir, p.WorkspacesDir} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return err
				}
			}

			cfg := &workspace.Config{
				Version:       "1",
				BaseBranch:    baseBranch,
				CreatedAt:     workspace.Timestamp(time.Now()),
				MergeStrategy: "squash",
				Editor:        "",
			}
			if err := p.SaveConfig(cfg); err != nil {
				return err
			}
			if err := p.SaveStore(&workspace.Store{Workspaces: []workspace.Workspace{}}); err != nil {
				return err
			}

			if err := excludeAgentspaceState(p.Root); err != nil {
				return err
			}
			if err := excludeAgentContext(p.Root); err != nil {
				return err
			}

			p.AppendLog("init base_branch=" + baseBranch)
			ui.Success("Initialized agentspace (base branch: %s)", baseBranch)
			ui.Plain("  config:     %s", p.ConfigFile)
			ui.Plain("  workspaces: %s", p.WorkspacesDir)
			return nil
		},
	}
	cmd.Flags().StringVar(&branch, "branch", "", "base branch (defaults to current branch)")
	return cmd
}

// excludeAgentspaceState keeps AgentSpace's local control files out of every
// shared worktree without changing the user's tracked .gitignore.
func excludeAgentspaceState(root string) error {
	return addGitExclude(root, ".agentspace/")
}

// excludeAgentContext adds AGENT_CONTEXT.md to the repo's .git/info/exclude so
// the agent context file is never staged, committed, or merged. The exclude
// file lives in the shared common dir, so it applies to all worktrees.
func excludeAgentContext(root string) error {
	return addGitExclude(root, "AGENT_CONTEXT.md")
}

func addGitExclude(root, entry string) error {
	commonDir, err := git.CommonDir(root)
	if err != nil {
		return nil // best-effort: skip if we cannot resolve the git dir
	}
	infoDir := filepath.Join(commonDir, "info")
	if err := os.MkdirAll(infoDir, 0o755); err != nil {
		return nil
	}
	excludePath := filepath.Join(infoDir, "exclude")

	data, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return nil
	}
	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == entry {
			return nil // already excluded
		}
	}

	f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil
	}
	defer f.Close()
	prefix := ""
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		prefix = "\n"
	}
	_, _ = f.WriteString(prefix + entry + "\n")
	return nil
}
