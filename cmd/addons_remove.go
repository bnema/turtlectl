package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/ui/styles"
)

var (
	removeForce    bool
	removeNoBackup bool
)

var addonsRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete", "uninstall"},
	Short:   "Remove an installed addon",
	Long: `Remove an installed addon from the Interface/AddOns directory.

By default, a backup is created before removal.
Use --no-backup to skip backup creation.
Use --force to skip confirmation prompt.

Examples:
  turtlectl addons remove pfQuest
  turtlectl addons remove pfQuest --force
  turtlectl addons remove pfQuest --no-backup`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		addonName := args[0]

		manager, err := getAddonManager()
		if err != nil {
			return err
		}

		// Check addon exists
		addon, err := manager.GetInfo(addonName)
		if err != nil {
			return fmt.Errorf("addon not found: %s", addonName)
		}

		// Confirm removal
		if !removeForce {
			fmt.Printf("Remove addon %s?\n", styles.Highlighted.Render(addon.Name))
			if addon.Title != "" && addon.Title != addon.Name {
				fmt.Printf("  Title: %s\n", addon.Title)
			}
			if addon.Path != "" {
				fmt.Printf("  Path: %s\n", addon.Path)
			}
			if !removeNoBackup {
				fmt.Println("  A backup will be created.")
			} else {
				fmt.Println(styles.FormatWarning("No backup will be created!"))
			}

			fmt.Print("\nConfirm? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "y" && response != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		// Remove addon
		createBackup := !removeNoBackup
		if err := manager.Remove(addonName, createBackup); err != nil {
			return fmt.Errorf("failed to remove addon: %w", err)
		}

		saveAddonManager()

		if createBackup {
			fmt.Println(styles.FormatSuccess(fmt.Sprintf("Addon %s removed (backup created)", addonName)))
		} else {
			fmt.Println(styles.FormatSuccess(fmt.Sprintf("Addon %s removed", addonName)))
		}

		return nil
	},
}

func init() {
	addonsRemoveCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Skip confirmation prompt")
	addonsRemoveCmd.Flags().BoolVar(&removeNoBackup, "no-backup", false, "Skip backup creation")
	addonsCmd.AddCommand(addonsRemoveCmd)
}
