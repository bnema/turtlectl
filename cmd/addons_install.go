package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/addons"
	uiaddons "github.com/bnema/turtlectl/internal/ui/addons"
)

var addonsInstallCmd = &cobra.Command{
	Use:   "install <git-url>",
	Short: "Install an addon from a git repository",
	Long: `Install an addon from a git repository URL.

The addon will be cloned to the Interface/AddOns directory.
The folder name will be derived from the .toc file if present.

Examples:
  turtlectl addons install https://github.com/shagu/pfQuest
  turtlectl addons install https://github.com/shagu/ShaguTweaks.git`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gitURL := args[0]

		manager, err := getAddonManager()
		if err != nil {
			return err
		}

		// Validate URL first
		if err := addons.ValidateGitURL(gitURL); err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}

		// Extract addon name for display
		addonName := addons.ExtractRepoName(gitURL)

		// Run multi-step progress TUI
		m := uiaddons.NewInstallModel(manager, gitURL, addonName)

		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		fm := finalModel.(uiaddons.InstallModel)
		if fm.GetError() != nil {
			return fm.GetError()
		}

		saveAddonManager()
		return nil
	},
}

func init() {
	addonsCmd.AddCommand(addonsInstallCmd)
}
