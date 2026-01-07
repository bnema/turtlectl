package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/bnema/turtle-wow-launcher/internal/launcher"
)

var installCmd = &cobra.Command{
	Use:     "install",
	Aliases: []string{"i"},
	Short:   "Install/update AppImage and create desktop file",
	Run: func(cmd *cobra.Command, args []string) {
		l := launcher.New(logger)

		log.Info("Starting installation")

		if err := l.EnsureLauncherDirs(); err != nil {
			log.Fatal("Failed to create directories", "error", err)
		}

		if err := l.UpdateAppImage(); err != nil {
			log.Fatal("Failed to update AppImage", "error", err)
		}

		if err := l.InstallDesktop(); err != nil {
			log.Fatal("Failed to install desktop file", "error", err)
		}

		log.Info("Installation complete! You can now launch Turtle WoW from your app menu.")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
