package styles

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Color palette - coherent with charmbracelet style
var (
	Primary   = lipgloss.Color("#7D56F4") // Purple (charmbracelet brand)
	Secondary = lipgloss.Color("#FF79C6") // Pink accent
	Success   = lipgloss.Color("#50FA7B") // Green
	Warning   = lipgloss.Color("#FFB86C") // Orange
	Error     = lipgloss.Color("#FF5555") // Red
	Muted     = lipgloss.Color("#6272A4") // Muted blue-gray
	Text      = lipgloss.Color("#F8F8F2") // Light text
	Subtle    = lipgloss.Color("#44475A") // Dark background accent
)

// Base styles
var (
	// Title style for headers
	Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFDF5")).
		Background(Primary).
		Padding(0, 1).
		Bold(true)

	// Subtitle style
	Subtitle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	// Normal text
	NormalText = lipgloss.NewStyle().
			Foreground(Text)

	// Muted text
	MutedText = lipgloss.NewStyle().
			Foreground(Muted)

	// Success text
	SuccessText = lipgloss.NewStyle().
			Foreground(Success)

	// Warning text
	WarningText = lipgloss.NewStyle().
			Foreground(Warning)

	// Error text
	ErrorText = lipgloss.NewStyle().
			Foreground(Error)

	// Selected item
	Selected = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	// Highlighted (focused)
	Highlighted = lipgloss.NewStyle().
			Foreground(Secondary).
			Bold(true)

	// App container
	App = lipgloss.NewStyle().
		Padding(1, 2)

	// Box border
	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Subtle).
		Padding(0, 1)

	// Status bar
	StatusBar = lipgloss.NewStyle().
			Foreground(Text).
			Background(Subtle).
			Padding(0, 1)

	// Help text
	Help = lipgloss.NewStyle().
		Foreground(Muted)

	// Spinner
	Spinner = lipgloss.NewStyle().
		Foreground(Primary)
)

// Symbols
var (
	CheckMark = lipgloss.NewStyle().Foreground(Success).SetString("✓")
	CrossMark = lipgloss.NewStyle().Foreground(Error).SetString("✗")
	Bullet    = lipgloss.NewStyle().Foreground(Primary).SetString("•")
	Arrow     = lipgloss.NewStyle().Foreground(Primary).SetString("→")
)

// AddonItem styles for list display
var (
	AddonName = lipgloss.NewStyle().
			Foreground(Text).
			Bold(true)

	AddonVersion = lipgloss.NewStyle().
			Foreground(Muted)

	AddonAuthor = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	AddonTracked = lipgloss.NewStyle().
			Foreground(Success)

	AddonUntracked = lipgloss.NewStyle().
			Foreground(Warning)

	AddonDefault = lipgloss.NewStyle().
			Foreground(Muted)
)

// AddonStatusType represents the tracking status of an addon
type AddonStatusType int

const (
	AddonStatusTracked AddonStatusType = iota
	AddonStatusUntracked
	AddonStatusDefault
)

// FormatAddonStatus returns a styled status indicator
func FormatAddonStatus(tracked bool) string {
	if tracked {
		return AddonTracked.Render("tracked")
	}
	return AddonUntracked.Render("untracked")
}

// FormatAddonStatusEx returns a styled status indicator with default support
func FormatAddonStatusEx(status AddonStatusType) string {
	switch status {
	case AddonStatusTracked:
		return AddonTracked.Render("tracked")
	case AddonStatusDefault:
		return AddonDefault.Render("default")
	default:
		return AddonUntracked.Render("untracked")
	}
}

// FormatUpdateAvailable returns a styled "update available" indicator
func FormatUpdateAvailable() string {
	style := lipgloss.NewStyle().Foreground(Primary).Bold(true)
	return style.Render("↑ update")
}

// FormatSuccess formats a success message
func FormatSuccess(msg string) string {
	return CheckMark.String() + " " + SuccessText.Render(msg)
}

// FormatError formats an error message
func FormatError(msg string) string {
	return CrossMark.String() + " " + ErrorText.Render(msg)
}

// FormatWarning formats a warning message
func FormatWarning(msg string) string {
	return WarningText.Render("! " + msg)
}

// Explore view styles
var (
	// NewBadge for newly discovered addons
	NewBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(Success).
			Bold(true).
			Padding(0, 1)

	// InstalledBadge for already installed addons
	InstalledBadge = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	// StarCount for GitHub stars
	StarCount = lipgloss.NewStyle().
			Foreground(Warning)

	// CategoryBadge for A-Z category
	CategoryBadge = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)
)

// FormatNewBadge returns a styled "NEW" badge
func FormatNewBadge() string {
	return NewBadge.Render("NEW")
}

// FormatInstalledBadge returns a styled "installed" indicator
func FormatInstalledBadge() string {
	return InstalledBadge.Render("installed")
}

// FormatStars formats star count with icon
func FormatStars(count int) string {
	if count <= 0 {
		return ""
	}
	if count >= 1000 {
		return StarCount.Render(fmt.Sprintf("★ %.1fk", float64(count)/1000))
	}
	return StarCount.Render(fmt.Sprintf("★ %d", count))
}

// FormatCategory formats a category letter
func FormatCategory(cat string) string {
	if cat == "" {
		return ""
	}
	return CategoryBadge.Render("[" + cat + "]")
}
