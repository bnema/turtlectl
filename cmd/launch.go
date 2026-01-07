package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/launcher"
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
		l := launcher.New(logger)

		log.Info("Starting Turtle WoW launcher")

		if err := l.EnsureAllDirs(); err != nil {
			log.Fatal("Failed to create directories", "error", err)
		}

		if err := l.UpdateAppImage(); err != nil {
			log.Fatal("Failed to update AppImage", "error", err)
		}

		if err := l.CleanConfig(); err != nil {
			log.Warn("Config cleanup issue", "error", err)
		}

		l.SetupEnvironment()

		if err := l.InitPreferences(); err != nil {
			log.Warn("Failed to initialize preferences", "error", err)
		}

		if err := l.Launch(args); err != nil {
			log.Fatal("Failed to launch", "error", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(launchCmd)
}
