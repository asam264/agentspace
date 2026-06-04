package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// promptTemplate is the pre-filled content for --edit mode.
const promptTemplate = `# 任务说明（以 # 开头的行会被过滤，保存后关闭编辑器）
#
# 请在下方填写完整的任务说明：

`

// resolvePrompt returns the final prompt string from the three input sources.
// Priority: --edit > --prompt-file > --prompt
// desc is the short description; if prompt is set and desc is empty, desc is
// derived from the first line of prompt (capped at 60 chars).
func resolvePrompt(promptStr, promptFile string, editFlag bool, editorCfg string) (prompt string, err error) {
	switch {
	case editFlag:
		prompt, err = editPromptInteractive(editorCfg)
	case promptFile != "":
		data, rerr := os.ReadFile(promptFile)
		if rerr != nil {
			return "", fmt.Errorf("read prompt file: %w", rerr)
		}
		prompt = strings.TrimSpace(string(data))
	default:
		prompt = strings.TrimSpace(promptStr)
	}
	return
}

// editPromptInteractive opens an editor on a temp file and returns its contents
// (comment lines stripped). editorCfg is the configured editor; empty means auto-detect.
func editPromptInteractive(editorCfg string) (string, error) {
	tmp, err := os.CreateTemp("", fmt.Sprintf("agentspace-prompt-%d-*.md", time.Now().Unix()))
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(promptTemplate); err != nil {
		_ = tmp.Close()
		return "", err
	}
	_ = tmp.Close()

	editor := detectEditorCmd(editorCfg)
	if editor.bin == "" {
		// Fall back to $EDITOR env var.
		editor = editorCmd{bin: os.Getenv("EDITOR")}
	}
	if editor.bin == "" {
		return "", fmt.Errorf("no editor found; set 'editor' in config.json or $EDITOR")
	}

	args := append(editor.extraFlags, tmp.Name())
	c := exec.Command(editor.bin, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return "", err
	}

	// Filter comment lines and trim.
	var lines []string
	for _, l := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "#") {
			lines = append(lines, l)
		}
	}
	result := strings.TrimSpace(strings.Join(lines, "\n"))
	if result == "" {
		return "", fmt.Errorf("prompt is empty after editing")
	}
	return result, nil
}

// descFromPrompt derives a short description from the first line of prompt.
func descFromPrompt(prompt string) string {
	first := strings.SplitN(strings.TrimSpace(prompt), "\n", 2)[0]
	if len([]rune(first)) > 60 {
		return string([]rune(first)[:60])
	}
	return first
}
