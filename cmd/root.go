package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "agentspace",
	Short:         "Manage parallel AI-agent workspaces backed by git worktrees",
	Long:          "agentspace creates isolated git-worktree-backed workspaces so multiple AI agents can work on the same project in parallel and merge back into a chosen target branch.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command. Returns an error for main to handle.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(
		newInitCmd(),
		newNewCmd(),
		newListCmd(),
		newStatusCmd(),
		newSnapshotCmd(),
		newSnapshotsCmd(),
		newRestoreCmd(),
		newDiffCmd(),
		newMergeCmd(),
		newRemoveCmd(),
		newCleanCmd(),
		newContextCmd(),
		newRunCmd(),
		newSubmitCmd(),
		newHandoffCmd(),
		newApproveCmd(),
		newRequestChangesCmd(),
		newResumeCmd(),
		newPreflightCmd(),
		newDispatchCmd(),
		newAttachCmd(),
		newRelayCmd(),
		newLinkTaskCmd(),
		newObserveTaskCmd(),
		newPauseCmd(),
		newContinueCmd(),
		newOverrideCmd(),
		newNoteCmd(),
		newInboxCmd(),
		newEventsCmd(),
		newCancelCmd(),
		newFailCmd(),
		newEditCmd(),
	)
}

// versionString is set for the --version flag area if needed later.
var versionString = "0.3.0"

func init() {
	rootCmd.Version = versionString
	rootCmd.SetVersionTemplate(fmt.Sprintf("agentspace %s\n", versionString))
}
