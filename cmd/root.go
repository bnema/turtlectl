package cmd

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/logger"
)

// Version info set via ldflags at build time
var (
	version = "dev"
	commit  = "unknown"
)

var verbose bool

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
	logger.Close()
}

func init() {
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		_ = logger.Init(verbose)
	}
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose/debug logging")
}

// getLogger returns the global logger for use in commands
func getLogger() *log.Logger {
	return logger.Log
}
