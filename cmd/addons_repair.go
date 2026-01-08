package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/ui/progress"
	"github.com/bnema/turtlectl/internal/ui/styles"
)

var addonsRepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Repair addon database and fix issues",
	Long: `Scan and repair the addon database.

This command will:
- Detect orphaned entries (in metadata but folder missing)
- Detect untracked addons (folder exists but no metadata)
- Verify git repository integrity
- Check if folder names match .toc files
- Auto-track addons with git remotes

Examples:
  turtlectl addons repair`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getAddonManager()
		if err != nil {
			return err
		}

		progress.PrintInProgress("Scanning addons directory...")

		result, err := manager.Repair()
		if err != nil {
			return fmt.Errorf("repair failed: %w", err)
		}

		// Print results
		fmt.Printf("\nScanned %d addon(s)\n\n", result.TotalScanned)

		if result.IssuesFound == 0 {
			fmt.Println(styles.FormatSuccess("No issues found"))
			return nil
		}

		fmt.Printf("Found %d issue(s):\n\n", result.IssuesFound)

		// Orphaned entries
		if len(result.OrphanedEntries) > 0 {
			fmt.Println(styles.WarningText.Render("Orphaned metadata entries (removed):"))
			for _, name := range result.OrphanedEntries {
				fmt.Printf("  - %s\n", name)
			}
			fmt.Println()
		}

		// Untracked addons
		if len(result.UntrackedAddons) > 0 {
			fmt.Println(styles.WarningText.Render("Untracked addons (now tracked if git repo):"))
			for _, name := range result.UntrackedAddons {
				fmt.Printf("  - %s\n", name)
			}
			fmt.Println()
		}

		// Corrupted repos
		if len(result.CorruptedRepos) > 0 {
			fmt.Println(styles.ErrorText.Render("Corrupted git repositories:"))
			for _, name := range result.CorruptedRepos {
				fmt.Printf("  - %s (re-install recommended)\n", name)
			}
			fmt.Println()
		}

		// Name mismatches
		if len(result.NameMismatches) > 0 {
			fmt.Println(styles.WarningText.Render("Folder name mismatches:"))
			for _, info := range result.NameMismatches {
				fmt.Printf("  - %s\n", info)
			}
			fmt.Println()
		}

		saveAddonManager()

		fmt.Println(styles.FormatSuccess("Repair complete"))

		return nil
	},
}

func init() {
	addonsCmd.AddCommand(addonsRepairCmd)
}
