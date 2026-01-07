package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/launcher"
	"github.com/bnema/turtlectl/internal/ui/progress"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove desktop file (keeps game data)",
	Run: func(cmd *cobra.Command, args []string) {
		l := launcher.New(getLogger())

		progress.PrintTitle("Uninstalling Desktop Integration")

		progress.PrintInProgress("Removing desktop file")
		if err := l.UninstallDesktop(); err != nil {
			progress.PrintError("Failed to uninstall: " + err.Error())
			os.Exit(1)
		}

		progress.PrintComplete("Desktop file removed")
		progress.PrintComplete("Icon removed")

		progress.PrintNewline()
		progress.PrintWarning("AppImage and config kept")
		progress.PrintDetail("Cache: " + l.CacheDir)
		progress.PrintDetail("Data: " + l.DataDir)
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
