package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/workspace"
)

const dryRunMarkerMagic = "agentspace-merge-dryrun-v1"

func dryRunMarkerPath(worktreePath string) string {
	return filepath.Clean(worktreePath) + ".agentspace"
}

func writeDryRunMarker(p workspace.Paths, worktreePath string) (string, error) {
	commonDir, err := git.CommonDir(p.Root)
	if err != nil {
		return "", err
	}
	markerPath := dryRunMarkerPath(worktreePath)
	contents := dryRunMarkerMagic + "\n" + commonDir + "\n"
	if err := os.WriteFile(markerPath, []byte(contents), 0o600); err != nil {
		return "", err
	}
	return markerPath, nil
}

func isVerifiedDryRunMarker(p workspace.Paths, markerPath string) bool {
	contents, err := os.ReadFile(markerPath)
	if err != nil {
		return false
	}
	parts := strings.Split(strings.TrimSpace(string(contents)), "\n")
	if len(parts) != 2 || parts[0] != dryRunMarkerMagic {
		return false
	}
	commonDir, err := git.CommonDir(p.Root)
	return err == nil && samePath(parts[1], commonDir)
}
