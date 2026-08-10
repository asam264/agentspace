package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/ui"
	"github.com/asam264/agentspace/internal/workspace"
	"github.com/spf13/cobra"
)

func newApproveCmd() *cobra.Command {
	return newReviewCmd("approve", "accepted", workspace.StatusAccepted, "Approve a submitted handoff for merge")
}

func newRequestChangesCmd() *cobra.Command {
	return newReviewCmd("request-changes", "changes_requested", workspace.StatusChangesReq, "Return a submitted handoff to the worker with review feedback")
}

func newReviewCmd(use, decision, nextStatus, short string) *cobra.Command {
	var message string
	cmd := &cobra.Command{
		Use:   use + " <name>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(message) == "" {
				return fmt.Errorf("--message/-m is required")
			}
			p, _, err := requireInit()
			if err != nil {
				return err
			}
			_, ws, err := loadWorkspace(p, args[0])
			if err != nil {
				return err
			}
			if ws.Status != workspace.StatusSubmitted || ws.Handoff == nil {
				return fmt.Errorf("workspace %q is not awaiting review", ws.Name)
			}
			wsPath := filepath.Join(p.Root, filepath.FromSlash(ws.Path))
			if dirty, err := git.Run(wsPath, "status", "--porcelain"); err != nil {
				return err
			} else if dirty != "" {
				return fmt.Errorf("workspace %q changed after submission; submit a new handoff before review", ws.Name)
			}
			head, err := git.Run(wsPath, "rev-parse", "HEAD")
			if err != nil {
				return err
			}
			if head != ws.Handoff.Commit {
				return fmt.Errorf("workspace %q HEAD changed after submission; submit a new handoff before review", ws.Name)
			}
			if err := p.WithLock(func(s *workspace.Store) error {
				cur := s.Find(ws.Name)
				if cur == nil || cur.Status != workspace.StatusSubmitted || cur.Handoff == nil {
					return fmt.Errorf("workspace %q is no longer awaiting review", ws.Name)
				}
				if cur.Handoff.Commit != head {
					return fmt.Errorf("workspace %q handoff changed before review", ws.Name)
				}
				cur.Review = &workspace.Review{
					Commit:     head,
					Decision:   decision,
					Summary:    strings.TrimSpace(message),
					ReviewedAt: workspace.Timestamp(time.Now()),
				}
				cur.Status = nextStatus
				appendEvent(cur, decision, strings.TrimSpace(message), head)
				return nil
			}); err != nil {
				return err
			}
			p.AppendLog(fmt.Sprintf("review name=%s decision=%s commit=%s", ws.Name, decision, head))
			ui.Success("Recorded %s for %q", decision, ws.Name)
			return nil
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "review summary (required)")
	return cmd
}
