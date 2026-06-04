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

			if err := ensureGitignore(p.Root); err != nil {
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

// ensureGitignore appends the workspaces dir to .gitignore if not present.
func ensureGitignore(root string) error {
	const entry = ".agentspace/workspaces/"
	path := filepath.Join(root, ".gitignore")

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == entry || strings.TrimSpace(line) == ".agentspace/workspaces" {
			return nil // already ignored
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	prefix := ""
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		prefix = "\n"
	}
	_, err = f.WriteString(prefix + "\n# agentspace worktrees\n" + entry + "\n")
	return err
}
