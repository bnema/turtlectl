package progress

import (
	"os"

	"github.com/charmbracelet/lipgloss"

	"github.com/bnema/turtlectl/internal/ui/styles"
)

// State represents the current state of a step
type State int

const (
	StatePending State = iota
	StateInProgress
	StateComplete
	StateError
)

// Step represents a single step in a multi-step operation
type Step struct {
	Name   string // Display name (e.g., "Cloning repository")
	State  State
	Detail string // Optional detail text (e.g., "Receiving objects: 67%")
	Error  error  // Error if State == StateError
}

// Icons - Nerd Font with ASCII fallback
type Icons struct {
	Check   string
	Cross   string
	Arrow   string
	Pending string
	Warning string
	Spinner string
}

var (
	// NerdFontIcons uses Nerd Font glyphs
	NerdFontIcons = Icons{
		Check:   "\uf00c", //
		Cross:   "\uf00d", //
		Arrow:   "\uf061", //
		Pending: "\uf111", //
		Warning: "\uf071", //
		Spinner: "\uf110", //
	}

	// ASCIIIcons uses simple ASCII characters
	ASCIIIcons = Icons{
		Check:   "+",
		Cross:   "x",
		Arrow:   "->",
		Pending: "o",
		Warning: "!",
		Spinner: "*",
	}
)

// GetIcons returns the appropriate icon set based on environment
func GetIcons() Icons {
	if os.Getenv("TURTLECTL_NERD_FONTS") == "1" {
		return NerdFontIcons
	}
	return ASCIIIcons
}

// Icon styles
var (
	IconStyleCheck   = lipgloss.NewStyle().Foreground(styles.Success)
	IconStyleCross   = lipgloss.NewStyle().Foreground(styles.Error)
	IconStyleArrow   = lipgloss.NewStyle().Foreground(styles.Primary)
	IconStylePending = lipgloss.NewStyle().Foreground(styles.Muted)
	IconStyleWarning = lipgloss.NewStyle().Foreground(styles.Warning)
	IconStyleSpinner = lipgloss.NewStyle().Foreground(styles.Primary)
)

// StyledIcon returns a styled icon string for the given state
func StyledIcon(state State) string {
	icons := GetIcons()
	switch state {
	case StateComplete:
		return IconStyleCheck.Render(icons.Check)
	case StateError:
		return IconStyleCross.Render(icons.Cross)
	case StateInProgress:
		return IconStyleSpinner.Render(icons.Spinner)
	default:
		return IconStylePending.Render(icons.Pending)
	}
}

// StepStyle returns the appropriate text style for a step based on state
func StepStyle(state State) lipgloss.Style {
	switch state {
	case StateComplete:
		return styles.SuccessText
	case StateError:
		return styles.ErrorText
	case StateInProgress:
		return styles.NormalText.Bold(true)
	default:
		return styles.MutedText
	}
}

// Progress holds the overall progress information
type Progress struct {
	Title       string // Operation title (e.g., "Installing pfQuest")
	Steps       []Step
	CurrentStep int
	SubProgress float64 // 0-100, for progress bar within current step
	SubDetail   string  // Git output detail (disappears when step completes)
}

// NewProgress creates a new Progress with the given title and step names
func NewProgress(title string, stepNames ...string) *Progress {
	steps := make([]Step, len(stepNames))
	for i, name := range stepNames {
		steps[i] = Step{Name: name, State: StatePending}
	}
	return &Progress{
		Title:       title,
		Steps:       steps,
		CurrentStep: 0,
	}
}

// StartStep marks the current step as in progress
func (p *Progress) StartStep() {
	if p.CurrentStep < len(p.Steps) {
		p.Steps[p.CurrentStep].State = StateInProgress
		p.SubProgress = 0
		p.SubDetail = ""
	}
}

// CompleteStep marks the current step as complete and advances
func (p *Progress) CompleteStep() {
	if p.CurrentStep < len(p.Steps) {
		p.Steps[p.CurrentStep].State = StateComplete
		p.Steps[p.CurrentStep].Detail = ""
		p.SubProgress = 0
		p.SubDetail = ""
		p.CurrentStep++
	}
}

// FailStep marks the current step as failed with an error
func (p *Progress) FailStep(err error) {
	if p.CurrentStep < len(p.Steps) {
		p.Steps[p.CurrentStep].State = StateError
		p.Steps[p.CurrentStep].Error = err
	}
}

// SetSubProgress updates the sub-progress percentage and detail
func (p *Progress) SetSubProgress(percent float64, detail string) {
	p.SubProgress = percent
	p.SubDetail = detail
}

// SetDetail sets the detail text for the current step
func (p *Progress) SetDetail(detail string) {
	if p.CurrentStep < len(p.Steps) {
		p.Steps[p.CurrentStep].Detail = detail
	}
}

// IsComplete returns true if all steps are complete
func (p *Progress) IsComplete() bool {
	for _, step := range p.Steps {
		if step.State != StateComplete {
			return false
		}
	}
	return true
}

// HasError returns true if any step has an error
func (p *Progress) HasError() bool {
	for _, step := range p.Steps {
		if step.State == StateError {
			return true
		}
	}
	return false
}
