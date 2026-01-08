package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/addons"
	"github.com/bnema/turtlectl/internal/ui/styles"
)

var addonsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed addons",
	Long:  `List all installed addons in the Interface/AddOns directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getAddonManager()
		if err != nil {
			return err
		}

		installedAddons, err := manager.ListInstalled()
		if err != nil {
			return fmt.Errorf("failed to list addons: %w", err)
		}

		if len(installedAddons) == 0 {
			fmt.Println("No addons installed")
			fmt.Println("\nInstall addons with: turtlectl addons install <git-url>")
			return nil
		}

		// Use tabwriter for aligned output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			styles.Title.Render("NAME"),
			styles.Title.Render("VERSION"),
			styles.Title.Render("AUTHOR"),
			styles.Title.Render("STATUS"),
		)

		for _, addon := range installedAddons {
			name := addon.Name
			version := addon.Version
			if version == "" {
				version = "-"
			}
			author := addon.Author
			if author == "" {
				author = "-"
			}

			// Determine status: default > tracked > untracked
			var status string
			if addons.IsDefaultAddon(addon.Name) {
				status = styles.FormatAddonStatusEx(styles.AddonStatusDefault)
			} else if addon.GitURL != "" {
				status = styles.FormatAddonStatusEx(styles.AddonStatusTracked)
			} else {
				status = styles.FormatAddonStatusEx(styles.AddonStatusUntracked)
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, version, author, status)
		}

		_ = w.Flush()

		fmt.Printf("\n%d addon(s) installed\n", len(installedAddons))
		fmt.Printf("Addons directory: %s\n", manager.GetAddonsDir())

		return nil
	},
}

func init() {
	addonsCmd.AddCommand(addonsListCmd)
}
