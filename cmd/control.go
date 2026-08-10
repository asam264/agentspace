package cmd

import (
	"fmt"
	"strings"

	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newPauseCmd() *cobra.Command {
	var message string
	cmd := &cobra.Command{
		Use:   "pause <name>",
		Short: "Pause a workspace under User Owner control",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(message) == "" {
				return fmt.Errorf("--message/-m is required")
			}
			return pauseWorkspace(args[0], message)
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "pause reason (required)")
	return cmd
}

func pauseWorkspace(name, message string) error {
	p, _, err := requireInit()
	if err != nil {
		return err
	}
	if err := p.WithLock(func(s *workspace.Store) error {
		ws := s.Find(name)
		if ws == nil {
			return fmt.Errorf("workspace %q not found", name)
		}
		if ws.Status == workspace.StatusPaused {
			return fmt.Errorf("workspace %q is already paused", ws.Name)
		}
		if !isOpenWorkspaceStatus(ws.Status) {
			return fmt.Errorf("workspace %q cannot be paused from terminal status %s", ws.Name, ws.Status)
		}
		if ws.Status == workspace.StatusConflicted {
			return fmt.Errorf("workspace %q has an in-progress merge; abort or continue that merge before pausing", ws.Name)
		}
		ws.Control.PausedFrom = ws.Status
		ws.Control.PauseReason = strings.TrimSpace(message)
		ws.Control.PausedAt = workspace.TimestampNow()
		ws.Status = workspace.StatusPaused
		appendEvent(ws, "paused", ws.Control.PauseReason, "")
		return nil
	}); err != nil {
		return err
	}
	p.AppendLog("pause name=" + name)
	ui.Success("Paused %q", name)
	return nil
}

func newContinueCmd() *cobra.Command {
	var message string
	cmd := &cobra.Command{
		Use:   "continue <name>",
		Short: "Resume a User Owner-paused workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(message) == "" {
				return fmt.Errorf("--message/-m is required")
			}
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			if err := p.WithLock(func(s *workspace.Store) error {
				ws := s.Find(args[0])
				if ws == nil {
					return fmt.Errorf("workspace %q not found", args[0])
				}
				if ws.Status != workspace.StatusPaused {
					return fmt.Errorf("workspace %q is not paused", ws.Name)
				}
				nextStatus := ws.Control.PausedFrom
				if hasPendingOverride(ws) || nextStatus == "" {
					nextStatus = workspace.StatusActive
				}
				ws.Status = nextStatus
				ws.Control.PausedFrom = ""
				ws.Control.PauseReason = ""
				ws.Control.PausedAt = ""
				appendEvent(ws, "continued", strings.TrimSpace(message), "")
				return nil
			}); err != nil {
				return err
			}
			p.AppendLog("continue name=" + args[0])
			ui.Success("Continued %q", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "resume reason (required)")
	return cmd
}

func newOverrideCmd() *cobra.Command {
	var message string
	cmd := &cobra.Command{
		Use:   "override <name>",
		Short: "Record a User Owner correction and require a fresh handoff",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(message) == "" {
				return fmt.Errorf("--message/-m is required")
			}
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			if err := p.WithLock(func(s *workspace.Store) error {
				ws := s.Find(args[0])
				if ws == nil {
					return fmt.Errorf("workspace %q not found", args[0])
				}
				if !isOpenWorkspaceStatus(ws.Status) || ws.Status == workspace.StatusConflicted {
					return fmt.Errorf("workspace %q cannot be overridden while status is %s", ws.Name, ws.Status)
				}
				oldCommit := ""
				if ws.Handoff != nil {
					oldCommit = ws.Handoff.Commit
				}
				ws.Handoff = nil
				ws.Review = nil
				ws.Control.Override = &workspace.ManualOverride{Reason: strings.TrimSpace(message), At: workspace.TimestampNow()}
				if ws.Status == workspace.StatusPaused {
					ws.Control.PausedFrom = workspace.StatusActive
				} else {
					ws.Status = workspace.StatusActive
				}
				appendEvent(ws, "manual_override", ws.Control.Override.Reason, oldCommit)
				return nil
			}); err != nil {
				return err
			}
			p.AppendLog("override name=" + args[0])
			ui.Success("Recorded User Owner override for %q; a fresh handoff is required", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "user correction (required)")
	return cmd
}

func newNoteCmd() *cobra.Command {
	var message, from string
	cmd := &cobra.Command{
		Use:   "note <name>",
		Short: "Append a Worker, Master, or User Owner workflow note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			message = strings.TrimSpace(message)
			from = strings.ToLower(strings.TrimSpace(from))
			if message == "" {
				return fmt.Errorf("--message/-m is required")
			}
			if from != "worker" && from != "master" && from != "user" {
				return fmt.Errorf("--from must be worker, master, or user")
			}
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			if err := recordEvent(p, args[0], from+"_note", message, ""); err != nil {
				return err
			}
			ui.Success("Recorded %s note for %q", from, args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "note text (required)")
	cmd.Flags().StringVar(&from, "from", "worker", "note author: worker, master, or user")
	return cmd
}
