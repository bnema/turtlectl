package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/launcher"
	"github.com/bnema/turtlectl/internal/ui/progress"
)

var launchCmd = &cobra.Command{
	Use:     "launch",
	Aliases: []string{"start", "run", "play"},
	Short:   "Launch Turtle WoW",
	Long: `Launches the Turtle WoW game client.

This will:
  1. Create necessary directories
  2. Check for launcher updates
  3. Clean any problematic config
  4. Setup environment (Wayland, GPU optimizations)
  5. Start the AppImage launcher`,
	Run: func(cmd *cobra.Command, args []string) {
		l := launcher.New(getLogger())

		progress.PrintTitle("Launching Turtle WoW")

		progress.PrintInProgress("Creating directories")
		if err := l.EnsureAllDirs(); err != nil {
			progress.PrintError("Failed to create directories: " + err.Error())
			os.Exit(1)
		}
		progress.PrintComplete("Directories ready")

		progress.PrintInProgress("Checking for updates")
		if err := l.UpdateAppImage(); err != nil {
			progress.PrintError("Failed to update AppImage: " + err.Error())
			os.Exit(1)
		}
		progress.PrintComplete("Launcher ready")

		if err := l.CleanConfig(); err != nil {
			progress.PrintWarning("Config cleanup issue: " + err.Error())
		}

		l.SetupEnvironment()

		if err := l.InitPreferences(); err != nil {
			progress.PrintWarning("Failed to initialize preferences: " + err.Error())
		}

		progress.PrintComplete("Starting game...")
		progress.PrintNewline()

		if err := l.Launch(args); err != nil {
			progress.PrintError("Failed to launch: " + err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(launchCmd)
}
