package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newCleanCmd() *cobra.Command {
	var yes, temp bool
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove all merged workspaces",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			if temp {
				return cleanTemporaryWorktrees(p, yes)
			}
			store, err := p.LoadStore()
			if err != nil {
				return err
			}

			var merged []workspace.Workspace
			for _, ws := range store.Workspaces {
				if ws.Status == workspace.StatusMerged {
					merged = append(merged, ws)
				}
			}
			if len(merged) == 0 {
				ui.Warn("no merged workspaces to clean")
				return nil
			}

			ui.Bold("Merged workspaces to remove:")
			for _, ws := range merged {
				ui.Plain("  %s (%s)", ws.Name, executionKind(&ws))
			}

			if !yes {
				fmt.Printf("Remove these %d workspace(s)? [y/N]: ", len(merged))
				ans, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				if strings.ToLower(strings.TrimSpace(ans)) != "y" {
					ui.Plain("Aborted.")
					return nil
				}
			}

			for i := range merged {
				ws := merged[i]
				if err := removeWorkspace(p, &ws); err != nil {
					ui.Error("failed to remove %q: %v", ws.Name, err)
					continue
				}
				ui.Success("Removed %q", ws.Name)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	cmd.Flags().BoolVar(&temp, "temp", false, "remove verified orphaned merge dry-run directories")
	return cmd
}

func cleanTemporaryWorktrees(p workspace.Paths, yes bool) error {
	entries, err := os.ReadDir(p.TmpDir)
	if os.IsNotExist(err) {
		ui.Warn("no temporary worktree directory to clean")
		return nil
	}
	if err != nil {
		return err
	}
	registered, err := git.WorktreePaths(p.Root)
	if err != nil {
		return err
	}
	type temporaryCandidate struct {
		path       string
		markerPath string
	}
	candidates := make([]temporaryCandidate, 0)
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "merge-dryrun-") {
			path := filepath.Join(p.TmpDir, entry.Name())
			if containsPath(registered, path) {
				ui.Warn("skipping registered temporary worktree: %s", path)
				continue
			}
			if !isVerifiedDryRunMarker(p, dryRunMarkerPath(path)) {
				ui.Warn("skipping unverified temporary directory: %s", path)
				continue
			}
			candidates = append(candidates, temporaryCandidate{path: path, markerPath: dryRunMarkerPath(path)})
			continue
		}
		if !entry.Type().IsRegular() || !strings.HasPrefix(entry.Name(), "merge-dryrun-") || !strings.HasSuffix(entry.Name(), ".agentspace") {
			continue
		}
		markerPath := filepath.Join(p.TmpDir, entry.Name())
		worktreePath := strings.TrimSuffix(markerPath, ".agentspace")
		if _, err := os.Lstat(worktreePath); !os.IsNotExist(err) || containsPath(registered, worktreePath) {
			continue
		}
		if isVerifiedDryRunMarker(p, markerPath) {
			candidates = append(candidates, temporaryCandidate{path: markerPath})
		}
	}
	if len(candidates) == 0 {
		ui.Warn("no verified orphaned merge dry-run directories to clean")
		return nil
	}
	ui.Bold("Orphaned merge dry-run temporary entries to remove:")
	for _, candidate := range candidates {
		ui.Plain("  %s", candidate.path)
	}
	if !yes {
		fmt.Printf("Remove these %d temporary item(s)? [y/N]: ", len(candidates))
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			ui.Plain("Aborted.")
			return nil
		}
	}
	for _, candidate := range candidates {
		if err := os.RemoveAll(candidate.path); err != nil {
			return err
		}
		if candidate.markerPath != "" {
			if err := os.Remove(candidate.markerPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removed temporary worktree %s but could not remove marker %s: %w", candidate.path, candidate.markerPath, err)
			}
		}
		p.AppendLog("clean-temp path=" + candidate.path)
		ui.Success("Removed orphaned merge dry-run temporary entry %s", candidate.path)
	}
	return nil
}
