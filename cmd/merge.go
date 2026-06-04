package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newMergeCmd() *cobra.Command {
	var dryRun, doContinue, doAbort bool
	cmd := &cobra.Command{
		Use:   "merge <name>",
		Short: "Merge a workspace branch into the base branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			p, cfg, err := requireInit()
			if err != nil {
				return err
			}
			_, ws, err := loadWorkspace(p, name)
			if err != nil {
				return err
			}

			switch {
			case doContinue:
				return mergeContinue(p, name)
			case doAbort:
				return mergeAbort(p, name)
			case dryRun:
				return mergeDryRun(p, ws)
			default:
				return mergeRun(p, cfg, ws)
			}
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "only detect conflicts, do not merge")
	cmd.Flags().BoolVar(&doContinue, "continue", false, "complete a conflicted merge")
	cmd.Flags().BoolVar(&doAbort, "abort", false, "abort an in-progress merge")
	return cmd
}

// mergeDryRun creates a temp branch off base, tries to merge, reports conflicts,
// then aborts and deletes the temp branch leaving no trace.
func mergeDryRun(p workspace.Paths, ws *workspace.Workspace) error {
	cfg, err := p.LoadConfig()
	if err != nil {
		return err
	}
	tmpBranch := "agentspace-dryrun/" + ws.Name

	// Create temp branch at base branch tip and check it out.
	if _, err := git.Run(p.Root, "branch", "-f", tmpBranch, cfg.BaseBranch); err != nil {
		return err
	}
	cleanup := func() {
		_, _ = git.Run(p.Root, "merge", "--abort")
		_, _ = git.Run(p.Root, "checkout", cfg.BaseBranch)
		_, _ = git.Run(p.Root, "branch", "-D", tmpBranch)
	}

	if _, err := git.Run(p.Root, "checkout", tmpBranch); err != nil {
		_, _ = git.Run(p.Root, "branch", "-D", tmpBranch)
		return err
	}

	_, mergeErr := git.Run(p.Root, "merge", "--no-commit", "--no-ff", ws.Branch)
	if mergeErr == nil {
		cleanup()
		ui.Success("No conflicts: %q merges cleanly into %s", ws.Name, cfg.BaseBranch)
		return nil
	}

	conflicts, _ := git.Run(p.Root, "diff", "--name-only", "--diff-filter=U")
	cleanup()
	if conflicts == "" {
		ui.Warn("merge would not apply cleanly")
		return nil
	}
	ui.Warn("Conflicts detected in:")
	for _, f := range strings.Split(conflicts, "\n") {
		ui.Plain("  %s", f)
	}
	return nil
}

// mergeRun performs the real squash merge into the base branch.
func mergeRun(p workspace.Paths, cfg *workspace.Config, ws *workspace.Workspace) error {
	if _, err := git.Run(p.Root, "checkout", cfg.BaseBranch); err != nil {
		return err
	}
	_, mergeErr := git.Run(p.Root, "merge", "--squash", ws.Branch)
	if mergeErr != nil {
		// Conflict path.
		conflicts, _ := git.Run(p.Root, "diff", "--name-only", "--diff-filter=U")
		ui.Error("Merge conflict in workspace %q", ws.Name)
		if conflicts != "" {
			ui.Warn("Conflicting files:")
			for _, f := range strings.Split(conflicts, "\n") {
				ui.Plain("  %s", f)
			}
		}
		_ = setStatus(p, ws.Name, workspace.StatusConflicted)
		maybeOpenEditor(p, cfg, conflicts)
		ui.Plain("Resolve conflicts, then run: agentspace merge --continue %s", ws.Name)
		ui.Plain("Or abort with:            agentspace merge --abort %s", ws.Name)
		return fmt.Errorf("merge stopped due to conflicts")
	}

	msg := fmt.Sprintf("agentspace: merge %s\n\n%s", ws.Name, ws.Description)
	if _, err := git.Run(p.Root, "commit", "--no-verify", "-m", msg); err != nil {
		return err
	}
	_ = setStatus(p, ws.Name, workspace.StatusMerged)
	p.AppendLog("merge name=" + ws.Name)
	ui.Success("Merged %q into %s", ws.Name, cfg.BaseBranch)
	return nil
}

// mergeContinue finalizes a conflicted squash merge.
func mergeContinue(p workspace.Paths, name string) error {
	_, ws, err := loadWorkspace(p, name)
	if err != nil {
		return err
	}
	if ws.Status != workspace.StatusConflicted {
		return fmt.Errorf("workspace %q is not in a conflicted state", name)
	}
	if _, err := git.Run(p.Root, "add", "-A"); err != nil {
		return err
	}
	msg := fmt.Sprintf("agentspace: merge %s\n\n%s", ws.Name, ws.Description)
	if _, err := git.Run(p.Root, "commit", "--no-verify", "-m", msg); err != nil {
		return err
	}
	_ = setStatus(p, name, workspace.StatusMerged)
	p.AppendLog("merge-continue name=" + name)
	ui.Success("Completed merge of %q", name)
	return nil
}

// mergeAbort aborts an in-progress merge and restores active status.
func mergeAbort(p workspace.Paths, name string) error {
	if _, err := git.Run(p.Root, "merge", "--abort"); err != nil {
		// For squash conflicts there may be no MERGE_HEAD; fall back to reset.
		if _, rerr := git.Run(p.Root, "reset", "--hard", "HEAD"); rerr != nil {
			return err
		}
	}
	_ = setStatus(p, name, workspace.StatusActive)
	p.AppendLog("merge-abort name=" + name)
	ui.Success("Aborted merge of %q; base branch restored", name)
	return nil
}

// setStatus updates a workspace status under lock.
func setStatus(p workspace.Paths, name, status string) error {
	return p.WithLock(func(s *workspace.Store) error {
		cur := s.Find(name)
		if cur == nil {
			return fmt.Errorf("workspace %q not found", name)
		}
		cur.Status = status
		return nil
	})
}

// maybeOpenEditor prompts the user to open conflict files in an editor.
func maybeOpenEditor(p workspace.Paths, cfg *workspace.Config, conflicts string) {
	if conflicts == "" {
		return
	}
	fmt.Print("Open conflict files in an editor? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(ans)) != "y" {
		return
	}
	editor := detectEditor(cfg.Editor)
	if editor == "" {
		ui.Warn("no editor found (set 'editor' in config.json)")
		return
	}
	files := strings.Split(conflicts, "\n")
	args := append([]string{}, files...)
	c := exec.Command(editor, args...)
	c.Dir = p.Root
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Start(); err != nil {
		ui.Warn("failed to launch %s: %v", editor, err)
	}
}
