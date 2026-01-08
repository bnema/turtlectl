package progress

import (
	"fmt"

	"github.com/bnema/turtlectl/internal/ui/styles"
)

// PrintStep prints a step with the appropriate icon and styling
func PrintStep(state State, message string) {
	icon := StyledIcon(state)
	textStyle := StepStyle(state)
	fmt.Printf("  %s %s\n", icon, textStyle.Render(message))
}

// PrintPending prints a pending step
func PrintPending(message string) {
	PrintStep(StatePending, message)
}

// PrintInProgress prints an in-progress step (without spinner animation)
func PrintInProgress(message string) {
	icons := GetIcons()
	icon := IconStyleSpinner.Render(icons.Spinner)
	fmt.Printf("  %s %s\n", icon, styles.NormalText.Bold(true).Render(message))
}

// PrintComplete prints a completed step
func PrintComplete(message string) {
	PrintStep(StateComplete, message)
}

// PrintError prints an error step
func PrintError(message string) {
	PrintStep(StateError, message)
}

// PrintSuccess prints a success message (alias for PrintComplete)
func PrintSuccess(message string) {
	PrintComplete(message)
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	icons := GetIcons()
	icon := IconStyleWarning.Render(icons.Warning)
	fmt.Printf("  %s %s\n", icon, styles.WarningText.Render(message))
}

// PrintTitle prints a title/header
func PrintTitle(title string) {
	style := styles.NormalText.Bold(true)
	fmt.Printf("%s\n\n", style.Render(title))
}

// PrintDetail prints an indented detail line
func PrintDetail(detail string) {
	fmt.Printf("      %s\n", styles.MutedText.Render(detail))
}

// PrintSummary prints a summary line with count
func PrintSummary(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("\n  %s\n", styles.MutedText.Render(message))
}

// PrintNewline prints an empty line
func PrintNewline() {
	fmt.Println()
}

// Sprintf helpers for building styled strings without printing

// FormatStep returns a formatted step string
func FormatStep(state State, message string) string {
	icon := StyledIcon(state)
	textStyle := StepStyle(state)
	return fmt.Sprintf("  %s %s", icon, textStyle.Render(message))
}

// FormatSuccess returns a formatted success string
func FormatSuccess(message string) string {
	return FormatStep(StateComplete, message)
}

// FormatError returns a formatted error string
func FormatError(message string) string {
	return FormatStep(StateError, message)
}

// FormatWarning returns a formatted warning string
func FormatWarning(message string) string {
	icons := GetIcons()
	icon := IconStyleWarning.Render(icons.Warning)
	return fmt.Sprintf("  %s %s", icon, styles.WarningText.Render(message))
}

// FormatCount formats a progress count like "3/12"
func FormatCount(current, total int) string {
	return fmt.Sprintf("%d/%d", current, total)
}

// FormatProgressLine formats a line like "Updating 3/12: pfQuest"
func FormatProgressLine(action string, current, total int, name string) string {
	icons := GetIcons()
	icon := IconStyleSpinner.Render(icons.Spinner)
	count := styles.MutedText.Render(fmt.Sprintf("%d/%d", current, total))
	return fmt.Sprintf("  %s %s %s: %s", icon, action, count, styles.NormalText.Bold(true).Render(name))
}
