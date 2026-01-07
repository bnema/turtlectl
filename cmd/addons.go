package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/addons"
	"github.com/bnema/turtlectl/internal/launcher"
	"github.com/bnema/turtlectl/internal/logger"
	addonsui "github.com/bnema/turtlectl/internal/ui/addons"
)

var addonManager *addons.Manager

var addonsCmd = &cobra.Command{
	Use:   "addons",
	Short: "Manage WoW addons",
	Long: `Manage World of Warcraft addons for Turtle WoW.

When run without subcommands, opens an interactive TUI for managing addons.

Examples:
  turtlectl addons                    # Interactive TUI
  turtlectl addons list               # List installed addons
  turtlectl addons install <git-url>  # Install addon from git URL
  turtlectl addons remove <name>      # Remove addon
  turtlectl addons update [name]      # Update specific or all addons
  turtlectl addons info <name>        # Show addon details
  turtlectl addons repair             # Sync metadata and fix issues`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize manager
		l := launcher.New(getLogger())
		manager := addons.NewManager(l.GameDir, l.DataDir, getLogger())

		if err := manager.Load(); err != nil {
			logger.Warn("Failed to load addon store", "error", err)
		}

		if err := manager.EnsureAddonsDir(); err != nil {
			return fmt.Errorf("failed to ensure addons directory: %w", err)
		}

		// Start interactive TUI
		model := addonsui.NewModel(manager)
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}

		return nil
	},
}

// getAddonManager returns the shared addon manager, initializing it if needed
func getAddonManager() (*addons.Manager, error) {
	if addonManager != nil {
		return addonManager, nil
	}

	l := launcher.New(getLogger())
	addonManager = addons.NewManager(l.GameDir, l.DataDir, getLogger())

	if err := addonManager.Load(); err != nil {
		logger.Warn("Failed to load addon store", "error", err)
	}

	if err := addonManager.EnsureAddonsDir(); err != nil {
		return nil, fmt.Errorf("failed to ensure addons directory: %w", err)
	}

	return addonManager, nil
}

// saveAddonManager saves the addon store
func saveAddonManager() {
	if addonManager != nil {
		if err := addonManager.Save(); err != nil {
			logger.Warn("Failed to save addon store", "error", err)
		}
	}
}

func init() {
	rootCmd.AddCommand(addonsCmd)
}
