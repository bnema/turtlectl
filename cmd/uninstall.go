package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/bnema/turtle-wow-launcher/internal/launcher"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove desktop file (keeps game data)",
	Run: func(cmd *cobra.Command, args []string) {
		l := launcher.New(logger)

		if err := l.UninstallDesktop(); err != nil {
			log.Fatal("Failed to uninstall", "error", err)
		}

		log.Warn("Desktop file removed. AppImage and config kept",
			"cache", l.CacheDir,
			"data", l.DataDir,
		)
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
