package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/launcher"
	"github.com/bnema/turtlectl/internal/ui/progress"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"u"},
	Short:   "Update the launcher AppImage only",
	Run: func(cmd *cobra.Command, args []string) {
		l := launcher.New(getLogger())

		progress.PrintTitle("Updating Turtle WoW Launcher")

		progress.PrintInProgress("Creating directories")
		if err := l.EnsureLauncherDirs(); err != nil {
			progress.PrintError("Failed to create directories: " + err.Error())
			os.Exit(1)
		}
		progress.PrintComplete("Directories ready")

		progress.PrintInProgress("Checking for updates")
		result, err := l.UpdateAppImageWithProgress(nil)
		if err != nil {
			progress.PrintError("Failed to update: " + err.Error())
			os.Exit(1)
		}

		if result != nil && result.AlreadyLatest {
			progress.PrintComplete("Already up to date")
		} else {
			progress.PrintComplete("Launcher updated")
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
