package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/bnema/turtle-wow-launcher/internal/launcher"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"u"},
	Short:   "Update the launcher AppImage only",
	Run: func(cmd *cobra.Command, args []string) {
		l := launcher.New(logger)

		log.Info("Checking for updates")

		if err := l.EnsureLauncherDirs(); err != nil {
			log.Fatal("Failed to create directories", "error", err)
		}

		if err := l.UpdateAppImage(); err != nil {
			log.Fatal("Failed to update AppImage", "error", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
