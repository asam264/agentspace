package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	var promptStr, promptFile, appendStr string
	var editFlag bool
	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit the prompt/description of an existing workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			p, cfg, err := requireInit()
			if err != nil {
				return err
			}

			return p.WithLock(func(s *workspace.Store) error {
				ws := s.Find(name)
				if ws == nil {
					return fmt.Errorf("workspace %q not found", name)
				}

				switch {
				case appendStr != "":
					sep := fmt.Sprintf("\n\n---\n[%s]\n", time.Now().UTC().Format("2006-01-02 15:04:05"))
					ws.Prompt = strings.TrimSpace(ws.Prompt) + sep + strings.TrimSpace(appendStr)
					ui.Success("Appended to prompt of %q", name)

				default:
					// --edit (default), --prompt, or --prompt-file
					newPrompt, err := resolvePrompt(promptStr, promptFile, editFlag || (!cmd.Flags().Changed("prompt") && !cmd.Flags().Changed("prompt-file")), cfg.Editor)
					if err != nil {
						return err
					}
					ws.Prompt = newPrompt
					ui.Success("Updated prompt of %q", name)
				}

				// Sync description from new prompt first line if desc was auto-derived.
				if ws.Prompt != "" && ws.Description == "" {
					ws.Description = descFromPrompt(ws.Prompt)
				}

				p.AppendLog("edit name=" + name)
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&editFlag, "edit", false, "open editor to rewrite prompt (default when no other flag given)")
	cmd.Flags().StringVar(&promptStr, "prompt", "", "overwrite prompt with this string")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "overwrite prompt from file")
	cmd.Flags().StringVar(&appendStr, "append", "", "append to existing prompt")
	return cmd
}
