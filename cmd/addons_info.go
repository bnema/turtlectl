package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/addons"
	"github.com/bnema/turtlectl/internal/ui/styles"
)

var addonsInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show addon details",
	Long: `Show detailed information about an installed addon.

Displays information from the .toc file and tracking metadata.

Examples:
  turtlectl addons info pfQuest
  turtlectl addons info ShaguTweaks`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		addonName := args[0]

		manager, err := getAddonManager()
		if err != nil {
			return err
		}

		addon, err := manager.GetInfo(addonName)
		if err != nil {
			return fmt.Errorf("addon not found: %s", addonName)
		}

		printAddonInfo(addon)

		// Check for backups
		backups, err := manager.GetBackupManager().ListBackups(addonName)
		if err == nil && len(backups) > 0 {
			fmt.Printf("\nBackups: %d available (latest: %s)\n", len(backups), backups[0])
		}

		// Check git status
		if addons.IsGitRepo(addon.Path) {
			if commit, err := addons.GetCurrentCommit(addon.Path); err == nil {
				fmt.Printf("Commit:    %s\n", commit)
			}
		}

		return nil
	},
}

func printAddonInfo(addon *addons.Addon) {
	// Name/Title
	fmt.Println(styles.Title.Render(addon.Name))
	if addon.Title != "" && addon.Title != addon.Name {
		fmt.Println(styles.MutedText.Render(addon.Title))
	}
	fmt.Println()

	// Basic info
	printField("Path", addon.Path)

	if addon.Version != "" {
		printField("Version", addon.Version)
	}

	if addon.Author != "" {
		printField("Author", addon.Author)
	}

	if addon.Notes != "" {
		printField("Notes", addon.Notes)
	}

	// Git/tracking info
	if addon.GitURL != "" {
		printField("Git URL", addon.GitURL)
		fmt.Printf("Status:    %s\n", styles.FormatAddonStatus(true))
	} else {
		fmt.Printf("Status:    %s\n", styles.FormatAddonStatus(false))
	}

	// Timestamps
	if !addon.InstalledAt.IsZero() {
		printField("Installed", addon.InstalledAt.Format("2006-01-02 15:04:05"))
	}

	if !addon.UpdatedAt.IsZero() {
		printField("Updated", addon.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
}

func printField(label, value string) {
	fmt.Printf("%-10s %s\n", label+":", value)
}

func init() {
	addonsCmd.AddCommand(addonsInfoCmd)
}
