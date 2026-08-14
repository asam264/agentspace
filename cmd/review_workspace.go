package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newReviewWorkspaceCmd() *cobra.Command {
	var open bool
	cmd := &cobra.Command{
		Use:   "review <name>",
		Short: "Create a fixed Handoff review worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, cfg, err := requireInit()
			if err != nil {
				return err
			}
			_, ws, err := loadWorkspace(p, args[0])
			if err != nil {
				return err
			}
			reviewSpace, created, err := ensureReviewWorkspace(p, ws)
			if err != nil {
				return err
			}
			verb := "Reused"
			if created {
				verb = "Created"
			}
			ui.Success("%s review workspace for %q", verb, ws.Name)
			ui.Plain("  path:    %s", reviewSpace.Path)
			ui.Plain("  handoff: %s", reviewSpace.Commit)
			ui.Plain("  source:  %s", ws.BaseBranch)
			ui.Plain("  base:    %s", ws.BaseCommit)
			ui.Plain("  compare: git -C %q diff %s...HEAD", reviewSpace.Path, ws.BaseCommit)
			if !open {
				ui.Plain("  open:    agentspace review %s --open", ws.Name)
				return nil
			}
			if err := openReviewWorkspace(cfg, reviewSpace.Path); err != nil {
				return err
			}
			if err := recordEvent(p, ws.Name, "review_workspace_opened", reviewSpace.Path, reviewSpace.Commit); err != nil {
				return err
			}
			ui.Success("Requested GoLand to open the review workspace")
			return nil
		},
	}
	cmd.Flags().BoolVar(&open, "open", false, "open the review workspace in GoLand")
	return cmd
}

func ensureReviewWorkspace(p workspace.Paths, ws *workspace.Workspace) (*workspace.ReviewWorkspace, bool, error) {
	if ws.Handoff == nil || ws.Handoff.Commit == "" {
		return nil, false, fmt.Errorf("workspace %q has no submitted handoff to review", ws.Name)
	}
	path, err := reviewWorkspacePath(p, ws.Name, ws.Handoff.Commit)
	if err != nil {
		return nil, false, err
	}
	if existing := reviewWorkspaceForCommit(ws, ws.Handoff.Commit); existing != nil && !samePath(existing.Path, path) {
		return nil, false, fmt.Errorf("workspace %q has a review workspace path outside the expected review directory", ws.Name)
	}

	registered, err := git.WorktreePaths(p.Root)
	if err != nil {
		return nil, false, err
	}
	_, statErr := os.Lstat(path)
	created := false
	switch {
	case statErr == nil:
		if !containsPath(registered, path) {
			return nil, false, fmt.Errorf("review path %s exists but is not a registered Git worktree; refusing to overwrite it", path)
		}
	case os.IsNotExist(statErr):
		if containsPath(registered, path) {
			return nil, false, fmt.Errorf("review path %s is registered but missing on disk; run git worktree prune after resolving the missing path", path)
		}
		if err := os.MkdirAll(p.ReviewsDir, 0o755); err != nil {
			return nil, false, err
		}
		if _, err := git.WorktreeAddDetached(p.Root, path, ws.Handoff.Commit); err != nil {
			return nil, false, err
		}
		created = true
	default:
		return nil, false, statErr
	}

	if err := validateReviewWorkspace(p, path, ws.Handoff.Commit); err != nil {
		if created {
			_, _ = git.WorktreeRemove(p.Root, path)
		}
		return nil, false, err
	}

	reviewSpace := workspace.ReviewWorkspace{Path: path, Commit: ws.Handoff.Commit, CreatedAt: workspace.TimestampNow()}
	err = p.WithLock(func(s *workspace.Store) error {
		cur := s.Find(ws.Name)
		if cur == nil || cur.Handoff == nil || cur.Handoff.Commit != ws.Handoff.Commit {
			return fmt.Errorf("workspace %q handoff changed while creating its review workspace", ws.Name)
		}
		if existing := reviewWorkspaceForCommit(cur, ws.Handoff.Commit); existing != nil {
			reviewSpace = *existing
			return nil
		}
		cur.ReviewSpaces = append(cur.ReviewSpaces, reviewSpace)
		appendEvent(cur, "review_workspace_created", path, ws.Handoff.Commit)
		return nil
	})
	if err != nil {
		if created {
			_, _ = git.WorktreeRemove(p.Root, path)
		}
		return nil, false, err
	}
	p.AppendLog(fmt.Sprintf("review-workspace name=%s commit=%s path=%s created=%t", ws.Name, reviewSpace.Commit, reviewSpace.Path, created))
	return &reviewSpace, created, nil
}

func reviewWorkspacePath(p workspace.Paths, name, commit string) (string, error) {
	shortCommit := commit
	if len(shortCommit) > 12 {
		shortCommit = shortCommit[:12]
	}
	path := filepath.Clean(filepath.Join(p.ReviewsDir, name+"-"+shortCommit))
	if !pathInside(p.ReviewsDir, path) {
		return "", fmt.Errorf("workspace name %q cannot be used for a review workspace path", name)
	}
	return path, nil
}

func pathInside(base, candidate string) bool {
	rel, err := filepath.Rel(base, candidate)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func reviewWorkspaceForCommit(ws *workspace.Workspace, commit string) *workspace.ReviewWorkspace {
	for i := range ws.ReviewSpaces {
		if ws.ReviewSpaces[i].Commit == commit {
			return &ws.ReviewSpaces[i]
		}
	}
	return nil
}

func validateReviewWorkspace(p workspace.Paths, path, commit string) error {
	if err := validateReviewWorkspaceIdentity(p, path, commit); err != nil {
		return err
	}
	dirty, err := git.Run(path, "status", "--porcelain")
	if err != nil {
		return err
	}
	if dirty != "" {
		return fmt.Errorf("review workspace %s has local changes and no longer represents an exact Handoff snapshot; discard or move those changes before reviewing", path)
	}
	return nil
}

func validateReviewWorkspaceIdentity(p workspace.Paths, path, commit string) error {
	if !pathInside(p.ReviewsDir, path) {
		return fmt.Errorf("review workspace path %s is outside %s", path, p.ReviewsDir)
	}
	root, err := git.RepoRoot(path)
	if err != nil {
		return err
	}
	if !samePath(root, path) {
		return fmt.Errorf("review workspace path %s is not a Git worktree root", path)
	}
	commonDir, err := git.CommonDir(path)
	if err != nil {
		return err
	}
	mainCommonDir, err := git.CommonDir(p.Root)
	if err != nil {
		return err
	}
	if !samePath(commonDir, mainCommonDir) {
		return fmt.Errorf("review workspace path %s belongs to a different Git repository", path)
	}
	head, err := git.Run(path, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if head != commit {
		return fmt.Errorf("review workspace %s is at %s, not handoff commit %s", path, head, commit)
	}
	return nil
}

func removeReviewWorkspaces(p workspace.Paths, ws *workspace.Workspace) error {
	if len(ws.ReviewSpaces) == 0 {
		return nil
	}
	registered, err := git.WorktreePaths(p.Root)
	if err != nil {
		return err
	}
	for _, reviewSpace := range ws.ReviewSpaces {
		if !pathInside(p.ReviewsDir, reviewSpace.Path) {
			return fmt.Errorf("refusing to remove review workspace outside %s: %s", p.ReviewsDir, reviewSpace.Path)
		}
		_, statErr := os.Lstat(reviewSpace.Path)
		if os.IsNotExist(statErr) && !containsPath(registered, reviewSpace.Path) {
			continue
		}
		if statErr != nil {
			return statErr
		}
		if !containsPath(registered, reviewSpace.Path) {
			return fmt.Errorf("refusing to remove unregistered review path %s", reviewSpace.Path)
		}
		if err := validateReviewWorkspaceIdentity(p, reviewSpace.Path, reviewSpace.Commit); err != nil {
			return err
		}
		if _, err := git.WorktreeRemove(p.Root, reviewSpace.Path); err != nil {
			return err
		}
	}
	return nil
}

func openReviewWorkspace(cfg *workspace.Config, path string) error {
	launcher := reviewLauncher(cfg)
	if launcher == "" {
		return fmt.Errorf("GoLand launcher not found; open %s manually, add goland64.exe to PATH, or set AGENTSPACE_REVIEW_EDITOR", path)
	}
	if err := exec.Command(launcher, path).Start(); err != nil {
		return fmt.Errorf("launch GoLand for review workspace: %w", err)
	}
	return nil
}

func reviewLauncher(cfg *workspace.Config) string {
	candidates := []string{strings.TrimSpace(os.Getenv("AGENTSPACE_REVIEW_EDITOR"))}
	if cfg != nil && strings.Contains(strings.ToLower(cfg.Editor), "goland") {
		candidates = append(candidates, strings.Fields(cfg.Editor)[0])
	}
	candidates = append(candidates, "goland64.exe", "goland.bat", "goland")
	for _, candidate := range candidates {
		candidate = strings.Trim(strings.TrimSpace(candidate), "\"")
		if candidate == "" {
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return ""
}
