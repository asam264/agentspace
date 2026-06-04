package cmd

import (
	"os/exec"
	"runtime"
)

// detectEditor returns a command to open files, honoring the configured editor
// first, then falling back to a preference order of known editors found on PATH.
func detectEditor(configured string) string {
	if configured != "" {
		return configured
	}
	candidates := []string{"code", "goland", "idea", "vim", "vi"}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return c
		}
		if runtime.GOOS == "windows" {
			if _, err := exec.LookPath(c + ".cmd"); err == nil {
				return c
			}
		}
	}
	return ""
}
