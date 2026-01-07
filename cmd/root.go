package cmd

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	verbose bool
	logger  *log.Logger
)

var rootCmd = &cobra.Command{
	Use:   "turtle-wow",
	Short: "Turtle WoW Launcher for Linux",
	Long: `A clean wrapper for the Turtle WoW AppImage launcher on Linux.
Handles AppImage management, config issues, and Wayland compatibility.

Quick start:
  turtle-wow install    Download AppImage and create desktop entry
  turtle-wow launch     Start the game`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		} else {
			log.SetLevel(log.InfoLevel)
		}
		logger = log.Default()
	}
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose/debug logging")
}
