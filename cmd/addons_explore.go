package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/launcher"
	addonsui "github.com/bnema/turtlectl/internal/ui/addons"
	"github.com/bnema/turtlectl/internal/ui/styles"
	"github.com/bnema/turtlectl/internal/wiki"
)

var addonsExploreCmd = &cobra.Command{
	Use:   "explore",
	Short: "Browse and discover addons from the Turtle WoW wiki",
	Long: `Browse the curated list of addons from the Turtle WoW wiki.

The addon registry is maintained centrally and cached locally for 24 hours.
New addons are marked with [NEW] for 7 days after being added to the registry.

Examples:
  turtlectl addons explore              # Interactive TUI
  turtlectl addons explore --refresh    # Force refresh from registry
  turtlectl addons explore --list       # Plain text list
  turtlectl addons explore --json       # JSON output for scripting`,
	RunE: runExplore,
}

func init() {
	addonsCmd.AddCommand(addonsExploreCmd)

	addonsExploreCmd.Flags().BoolP("refresh", "r", false, "Force refresh the registry cache")
	addonsExploreCmd.Flags().BoolP("list", "l", false, "Output as plain text list (non-interactive)")
	addonsExploreCmd.Flags().Bool("json", false, "Output as JSON (non-interactive)")
}

func runExplore(cmd *cobra.Command, args []string) error {
	refresh, _ := cmd.Flags().GetBool("refresh")
	listOutput, _ := cmd.Flags().GetBool("list")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Get launcher for paths
	l := launcher.New(getLogger())

	// Initialize registry
	registry := wiki.NewRegistry(l.CacheDir, getLogger())

	// Non-interactive modes
	if listOutput || jsonOutput {
		return runExploreNonInteractive(registry, refresh, jsonOutput)
	}

	// Interactive TUI mode
	return runExploreTUI(registry, refresh, l)
}

// runExploreNonInteractive handles --list and --json output modes
func runExploreNonInteractive(registry *wiki.Registry, refresh, jsonOutput bool) error {
	addons, err := registry.GetAddons(refresh)
	if err != nil {
		return fmt.Errorf("failed to load addons: %w", err)
	}

	// Sort addons
	wiki.SortAddons(addons)

	info := registry.GetInfo()

	if jsonOutput {
		return outputJSON(addons, info)
	}

	return outputTable(addons, info)
}

// outputJSON outputs addons as JSON
func outputJSON(addons []wiki.WikiAddon, info wiki.RegistryInfo) error {
	output := struct {
		Addons      []wiki.WikiAddon `json:"addons"`
		Total       int              `json:"total"`
		NewCount    int              `json:"new_count"`
		CacheAge    string           `json:"cache_age,omitempty"`
		GeneratedAt string           `json:"generated_at,omitempty"`
	}{
		Addons:   addons,
		Total:    len(addons),
		NewCount: info.NewAddons,
	}

	if !info.GeneratedAt.IsZero() {
		output.GeneratedAt = info.GeneratedAt.Format("2006-01-02T15:04:05Z07:00")
		output.CacheAge = info.Age.Round(1000000000).String() // Round to seconds
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// outputTable outputs addons as a formatted table
func outputTable(addons []wiki.WikiAddon, info wiki.RegistryInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header
	_, _ = fmt.Fprintln(w, "NAME\tAUTHOR\tSTARS\tSTATUS\tDESCRIPTION")
	_, _ = fmt.Fprintln(w, "----\t------\t-----\t------\t-----------")

	// Rows
	for _, addon := range addons {
		status := ""
		if addon.IsNew() {
			status = "NEW"
		}
		if addon.IsInstalled {
			if status != "" {
				status += ", "
			}
			status += "installed"
		}

		// Truncate description
		desc := addon.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		// Format stars
		stars := ""
		if addon.Stars > 0 {
			if addon.Stars >= 1000 {
				stars = fmt.Sprintf("%.1fk", float64(addon.Stars)/1000)
			} else {
				stars = fmt.Sprintf("%d", addon.Stars)
			}
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			addon.Name,
			addon.Author,
			stars,
			status,
			desc,
		)
	}

	_ = w.Flush()

	// Summary
	fmt.Println()
	summary := fmt.Sprintf("Total: %d addons", len(addons))
	if info.NewAddons > 0 {
		summary += fmt.Sprintf(" (%d new)", info.NewAddons)
	}
	fmt.Println(summary)

	// Cache info
	if info.IsStale {
		days := int(info.Age.Hours() / 24)
		fmt.Println(styles.FormatWarning(fmt.Sprintf("Cache is %d day(s) old. Use --refresh to update.", days)))
	}

	return nil
}

// runExploreTUI runs the interactive TUI
func runExploreTUI(registry *wiki.Registry, refresh bool, l *launcher.Launcher) error {
	// Get addon manager for install functionality
	manager, err := getAddonManager()
	if err != nil {
		return err
	}

	// Create and run TUI
	model := addonsui.NewExploreModel(manager, registry, refresh)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}
