package cmd

import (
	"os/exec"
	"runtime"
	"strings"
)

// editorCmd holds the editor executable and any extra flags needed to make it
// block until the file is closed (e.g. --wait for VSCode).
type editorCmd struct {
	bin        string
	extraFlags []string
}

// guiEditors maps known GUI editors to the flag that makes them wait.
var guiEditors = map[string][]string{
	"code":   {"--wait"},
	"goland": {"--wait"},
	"idea":   {"--wait"},
	"cursor": {"--wait"},
}

// detectEditorCmd returns an editorCmd for the configured/detected editor.
// For GUI editors it appends the blocking flag automatically.
func detectEditorCmd(configured string) editorCmd {
	bin := configured
	if bin == "" {
		for _, c := range []string{"code", "goland", "idea", "cursor", "vim", "vi"} {
			if _, err := exec.LookPath(c); err == nil {
				bin = c
				break
			}
			if runtime.GOOS == "windows" {
				if _, err := exec.LookPath(c + ".cmd"); err == nil {
					bin = c
					break
				}
			}
		}
	}
	// Strip any flags the user may have embedded in the configured string.
	parts := strings.Fields(bin)
	if len(parts) > 1 {
		bin = parts[0]
	}
	return editorCmd{bin: bin, extraFlags: guiEditors[strings.ToLower(bin)]}
}

// detectEditor returns just the binary name (used by merge conflict opening).
func detectEditor(configured string) string {
	return detectEditorCmd(configured).bin
}
