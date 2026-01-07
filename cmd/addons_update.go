package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/addons"
	uiaddons "github.com/bnema/turtlectl/internal/ui/addons"
)

var addonsUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update addon(s)",
	Long: `Update a specific addon or all addons.

Uses git fast-forward to update addons. If local modifications exist,
the update will fail (use remove + install to force).

Examples:
  turtlectl addons update          # Update all addons
  turtlectl addons update pfQuest  # Update specific addon`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getAddonManager()
		if err != nil {
			return err
		}

		var addonName string
		if len(args) > 0 {
			addonName = args[0]
		}

		if addonName == "" {
			return updateAllAddons(manager)
		}
		return updateSingleAddon(manager, addonName)
	},
}

func updateSingleAddon(manager *addons.Manager, name string) error {
	m := uiaddons.NewUpdateSingleModel(manager, name)

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	fm := finalModel.(uiaddons.UpdateSingleModel)
	if fm.GetError() != nil {
		return fm.GetError()
	}

	saveAddonManager()
	return nil
}

func updateAllAddons(manager *addons.Manager) error {
	m := uiaddons.NewUpdateAllModel(manager)

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	fm := finalModel.(uiaddons.UpdateAllModel)
	if fm.GetError() != nil {
		return fm.GetError()
	}

	saveAddonManager()
	return nil
}

func init() {
	addonsCmd.AddCommand(addonsUpdateCmd)
}
