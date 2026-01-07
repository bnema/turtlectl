package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/launcher"
	"github.com/bnema/turtlectl/internal/ui/progress"
)

var cleanAll bool

var cleanCmd = &cobra.Command{
	Use:     "clean",
	Aliases: []string{"c"},
	Short:   "Nuclear clean: remove all config, cache, and AppImage (keeps game files)",
	Long: `Completely removes all launcher data including:
  - Config and preferences (~/.local/share/turtle-wow)
  - Cache and AppImage (~/.cache/turtle-wow)
  - Desktop file and icon

Game files in ~/Games/turtle-wow are preserved by default.
Use --all to also remove game files (full purge).`,
	Run: func(cmd *cobra.Command, args []string) {
		l := launcher.New(getLogger())

		if cleanAll {
			progress.PrintTitle("Full Purge")
			progress.PrintWarning("Removing ALL data including game files")
		} else {
			progress.PrintTitle("Cleaning Launcher Data")
		}

		progress.PrintInProgress("Removing data")
		if err := l.Clean(cleanAll); err != nil {
			progress.PrintError("Failed to clean: " + err.Error())
			os.Exit(1)
		}

		progress.PrintComplete("Data directory removed")
		progress.PrintComplete("Cache directory removed")
		progress.PrintComplete("Desktop integration removed")

		if cleanAll {
			progress.PrintComplete("Game files removed")
		} else {
			progress.PrintDetail("Game files preserved at: " + l.GameDir)
		}

		progress.PrintNewline()
		progress.PrintSuccess("Clean complete")
	},
}

var resetCredentialsCmd = &cobra.Command{
	Use:   "reset-credentials",
	Short: "Reset saved login credentials only",
	Run: func(cmd *cobra.Command, args []string) {
		l := launcher.New(getLogger())

		progress.PrintTitle("Resetting Credentials")

		progress.PrintInProgress("Removing credential files")
		if err := l.ResetCredentials(); err != nil {
			progress.PrintError("Failed to reset: " + err.Error())
			os.Exit(1)
		}

		progress.PrintComplete("Credentials reset")
		progress.PrintNewline()
		progress.PrintSuccess("You will need to log in again")
	},
}

func init() {
	cleanCmd.Flags().BoolVarP(&cleanAll, "all", "a", false, "Also remove game files (full purge)")
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(resetCredentialsCmd)
}
