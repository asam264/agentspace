package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

// mergeDryRun tests the merge in a detached temporary worktree. It never checks
// out another branch in the user's main worktree.
func mergeDryRun(p workspace.Paths, ws *workspace.Workspace) (resultErr error) {
	if ws.Handoff == nil || ws.Handoff.Commit == "" {
		return fmt.Errorf("workspace %q has no submitted handoff to test", ws.Name)
	}
	if err := os.MkdirAll(p.TmpDir, 0o755); err != nil {
		return err
	}
	tmpPath := filepath.Join(p.TmpDir, fmt.Sprintf("merge-dryrun-%d", time.Now().UnixNano()))
	target := targetBranch(ws)
	if _, err := git.WorktreeAddDetached(p.Root, tmpPath, target); err != nil {
		return err
	}
	markerPath, err := writeDryRunMarker(p, tmpPath)
	if err != nil {
		if _, removeErr := git.WorktreeRemove(p.Root, tmpPath); removeErr != nil {
			cleanupErr := fmt.Errorf("write marker: %v; remove worktree: %w", err, removeErr)
			recordDryRunCleanupFailure(p, ws, tmpPath, cleanupErr)
			ui.Warn("merge dry-run temporary worktree could not be initialized or removed: %s", tmpPath)
			return fmt.Errorf("create merge dry-run marker failed and temporary worktree remains at %s; close processes using it, then run git -C %q worktree remove --force %q: %w", tmpPath, p.Root, tmpPath, cleanupErr)
		}
		return err
	}
	defer func() {
		_, _ = git.Run(tmpPath, "merge", "--abort")
		var cleanupErr error
		cleanupPath := tmpPath
		if _, err := git.WorktreeRemove(p.Root, tmpPath); err != nil {
			cleanupErr = err
		} else if err := os.Remove(markerPath); err != nil && !os.IsNotExist(err) {
			cleanupErr = err
			cleanupPath = markerPath
		}
		if cleanupErr != nil {
			recordDryRunCleanupFailure(p, ws, cleanupPath, cleanupErr)
			ui.Warn("merge dry-run temporary files could not be fully cleaned: %s", cleanupPath)
			if resultErr == nil {
				resultErr = fmt.Errorf("merge dry-run completed but temporary cleanup failed; run agentspace clean --temp after closing any process using %s: %w", cleanupPath, cleanupErr)
			}
		}
	}()

	_, mergeErr := git.Run(tmpPath, "merge", "--no-commit", "--no-ff", ws.Handoff.Commit)
	if mergeErr == nil {
		ui.Success("No conflicts: %q merges cleanly into %s", ws.Name, target)
		return nil
	}

	conflicts, _ := git.Run(tmpPath, "diff", "--name-only", "--diff-filter=U")
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

func recordDryRunCleanupFailure(p workspace.Paths, ws *workspace.Workspace, path string, cleanupErr error) {
	p.AppendLog(fmt.Sprintf("merge-dryrun-cleanup-failed path=%s error=%v", path, cleanupErr))
	if eventErr := recordEvent(p, ws.Name, "merge_dryrun_cleanup_failed", path+": "+cleanupErr.Error(), ws.Handoff.Commit); eventErr != nil {
		p.AppendLog(fmt.Sprintf("merge-dryrun-cleanup-event-failed path=%s error=%v", path, eventErr))
	}
}

// mergeRun performs the real squash merge into the workspace target branch.
func mergeRun(p workspace.Paths, cfg *workspace.Config, ws *workspace.Workspace) error {
	if err := ensureMergeReady(p, ws); err != nil {
		return err
	}
	_, mergeErr := git.Run(p.Root, "merge", "--squash", ws.Handoff.Commit)
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

	msg := fmt.Sprintf("agentspace: merge %s\n\n%s", ws.Name, mergeSummary(ws))
	if _, err := git.Run(p.Root, "commit", "--no-verify", "-m", msg); err != nil {
		return err
	}
	_ = setStatus(p, ws.Name, workspace.StatusMerged)
	p.AppendLog("merge name=" + ws.Name)
	ui.Success("Merged %q into %s", ws.Name, targetBranch(ws))
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
	if blockers := workflowControlBlockers(ws); len(blockers) > 0 {
		return fmt.Errorf("workspace %q cannot continue merge: %s", ws.Name, strings.Join(blockers, "; "))
	}
	if _, err := git.Run(p.Root, "add", "-A"); err != nil {
		return err
	}
	msg := fmt.Sprintf("agentspace: merge %s\n\n%s", ws.Name, mergeSummary(ws))
	if _, err := git.Run(p.Root, "commit", "--no-verify", "-m", msg); err != nil {
		return err
	}
	_ = setStatus(p, name, workspace.StatusMerged)
	p.AppendLog("merge-continue name=" + name)
	ui.Success("Completed merge of %q", name)
	return nil
}

// mergeAbort aborts an in-progress merge and restores accepted status.
func mergeAbort(p workspace.Paths, name string) error {
	if _, err := git.Run(p.Root, "merge", "--abort"); err != nil {
		// For squash conflicts there may be no MERGE_HEAD; fall back to reset.
		if _, rerr := git.Run(p.Root, "reset", "--hard", "HEAD"); rerr != nil {
			return err
		}
	}
	_ = setStatus(p, name, workspace.StatusAccepted)
	p.AppendLog("merge-abort name=" + name)
	ui.Success("Aborted merge of %q; target branch restored", name)
	return nil
}

// ensureMergeReady protects the main worktree and makes merge operate on the
// exact commit the master reviewed, rather than a later workspace mutation.
func ensureMergeReady(p workspace.Paths, ws *workspace.Workspace) error {
	if ws.Status != workspace.StatusAccepted || ws.Handoff == nil || ws.Review == nil || ws.Review.Decision != "accepted" {
		return fmt.Errorf("workspace %q must be approved before it can be merged", ws.Name)
	}
	if ws.Review.Commit != ws.Handoff.Commit {
		return fmt.Errorf("workspace %q review does not match its submitted handoff", ws.Name)
	}
	if blockers := workflowControlBlockers(ws); len(blockers) > 0 {
		return fmt.Errorf("workspace %q cannot be merged: %s", ws.Name, strings.Join(blockers, "; "))
	}
	branch, err := git.CurrentBranch(p.Root)
	if err != nil {
		return err
	}
	if branch != targetBranch(ws) {
		return fmt.Errorf("main worktree is on %q; switch to workspace target branch %q before merging", branch, targetBranch(ws))
	}
	dirty, err := git.Run(p.Root, "status", "--porcelain")
	if err != nil {
		return err
	}
	if dirty != "" {
		return fmt.Errorf("main worktree has uncommitted changes; commit, stash, or clean it before merging:\n%s", dirty)
	}
	wsPath, err := executionPath(p, ws)
	if err != nil {
		return err
	}
	head, err := git.Run(wsPath, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if head != ws.Handoff.Commit {
		return fmt.Errorf("workspace %q changed after approval; it must be resubmitted and reviewed", ws.Name)
	}
	store, err := p.LoadStore()
	if err != nil {
		return err
	}
	if blockers := dependencyBlockers(store, ws); len(blockers) > 0 {
		return fmt.Errorf("workspace %q has unmet dependencies: %s", ws.Name, strings.Join(blockers, "; "))
	}
	return nil
}

func mergeSummary(ws *workspace.Workspace) string {
	if ws.Handoff != nil && ws.Handoff.Summary != "" {
		return ws.Handoff.Summary
	}
	return ws.Description
}

// setStatus updates a workspace status under lock.
func setStatus(p workspace.Paths, name, status string) error {
	return p.WithLock(func(s *workspace.Store) error {
		cur := s.Find(name)
		if cur == nil {
			return fmt.Errorf("workspace %q not found", name)
		}
		cur.Status = status
		appendEvent(cur, status, "", "")
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
