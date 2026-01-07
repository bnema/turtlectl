package cmd

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// Version info set via ldflags at build time
var (
	version = "dev"
	commit  = "unknown"
)

var (
	verbose bool
	logger  *log.Logger
)

var rootCmd = &cobra.Command{
	Use:     "turtlectl",
	Short:   "Turtle WoW CLI for Linux",
	Version: version + " (" + commit + ")",
	Long: `A Go CLI tool to manage and run Turtle WoW on Linux.
Handles AppImage management, config issues, and Wayland compatibility.

Quick start:
  turtlectl install    Download AppImage and create desktop entry
  turtlectl launch     Start the game`,
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
