package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/bnema/turtle-wow-launcher/internal/launcher"
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
		l := launcher.New(logger)

		if err := l.Clean(cleanAll); err != nil {
			log.Fatal("Failed to clean", "error", err)
		}
	},
}

var resetCredentialsCmd = &cobra.Command{
	Use:   "reset-credentials",
	Short: "Reset saved login credentials only",
	Run: func(cmd *cobra.Command, args []string) {
		l := launcher.New(logger)

		if err := l.ResetCredentials(); err != nil {
			log.Fatal("Failed to reset credentials", "error", err)
		}
	},
}

func init() {
	cleanCmd.Flags().BoolVarP(&cleanAll, "all", "a", false, "Also remove game files (full purge)")
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(resetCredentialsCmd)
}
