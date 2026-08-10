package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Run executes a git command in the given directory (empty dir = current).
// It returns trimmed stdout. On failure it returns an error that includes the
// raw git stderr output so callers can surface it verbatim.
func Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var out strings.Builder
	var errBuf strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		stderr := strings.TrimSpace(errBuf.String())
		if stderr == "" {
			stderr = err.Error()
		}
		return strings.TrimSpace(out.String()), fmt.Errorf("git %s: %s", strings.Join(args, " "), stderr)
	}
	return strings.TrimSpace(out.String()), nil
}

// RunPassthrough runs a git command streaming stdout/stderr directly to the
// provided writers (used for diff with color). Returns the exit error if any.
func RunPassthrough(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// IsRepo reports whether dir (or cwd if empty) is inside a git work tree.
func IsRepo(dir string) bool {
	out, err := Run(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// RepoRoot returns the absolute path of the repository root.
func RepoRoot(dir string) (string, error) {
	return Run(dir, "rev-parse", "--show-toplevel")
}

// MainRoot returns the main working tree root, even when called from inside a
// linked worktree. It derives the path from the common git dir so that
// agentspace state (which lives in the main repo) is always found.
func MainRoot(dir string) (string, error) {
	commonDir, err := Run(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		// Older git without --path-format: fall back to show-toplevel.
		return RepoRoot(dir)
	}
	commonDir = filepath.ToSlash(commonDir)
	// commonDir is typically "<root>/.git"; the main root is its parent.
	if strings.HasSuffix(commonDir, "/.git") {
		return filepath.FromSlash(strings.TrimSuffix(commonDir, "/.git")), nil
	}
	// Bare or unusual layout: use the parent directory.
	return filepath.Dir(filepath.FromSlash(commonDir)), nil
}

// CurrentBranch returns the current branch name.
func CurrentBranch(dir string) (string, error) {
	return Run(dir, "rev-parse", "--abbrev-ref", "HEAD")
}

// CommonDir returns the absolute path of the shared git common dir (the main
// repo's .git), which is shared by all linked worktrees.
func CommonDir(dir string) (string, error) {
	out, err := Run(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	return filepath.FromSlash(out), nil
}

// RevParse resolves a ref to a short commit hash.
func RevParseShort(dir, ref string) (string, error) {
	return Run(dir, "rev-parse", "--short", ref)
}

// WorktreeAdd creates a new worktree at path on a new branch, based on from.
func WorktreeAdd(dir, path, branch, from string) (string, error) {
	return Run(dir, "worktree", "add", path, "-b", branch, from)
}

// WorktreeAddDetached creates an isolated temporary worktree without creating
// or checking out another branch in the caller's main worktree.
func WorktreeAddDetached(dir, path, from string) (string, error) {
	return Run(dir, "worktree", "add", "--detach", path, from)
}

// WorktreeRemove removes a worktree (force).
func WorktreeRemove(dir, path string) (string, error) {
	return Run(dir, "worktree", "remove", path, "--force")
}

// BranchDelete force-deletes a branch.
func BranchDelete(dir, branch string) (string, error) {
	return Run(dir, "branch", "-D", branch)
}

// BranchExists reports whether a branch ref exists.
func BranchExists(dir, branch string) bool {
	_, err := Run(dir, "rev-parse", "--verify", "refs/heads/"+branch)
	return err == nil
}
