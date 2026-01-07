package progress

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bnema/turtlectl/internal/ui/styles"
)

// Model is the bubbletea model for multi-step progress display
type Model struct {
	progress    *Progress
	spinner     spinner.Model
	progressBar progress.Model
	done        bool
	err         error
	width       int
}

// NewModel creates a new progress model with the given title and steps
func NewModel(title string, stepNames ...string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(30),
		progress.WithoutPercentage(),
	)

	return Model{
		progress:    NewProgress(title, stepNames...),
		spinner:     s,
		progressBar: p,
		width:       80,
	}
}

// Progress messages for updating state
type (
	// StartStepMsg signals to start the current step
	StartStepMsg struct{}

	// CompleteStepMsg signals the current step is complete
	CompleteStepMsg struct{}

	// FailStepMsg signals the current step failed
	FailStepMsg struct{ Err error }

	// SubProgressMsg updates the sub-progress within current step
	SubProgressMsg struct {
		Percent float64
		Detail  string
	}

	// DoneMsg signals the entire operation is complete
	DoneMsg struct{ Err error }
)

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tea.WindowSize())
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.progressBar.Width = minInt(msg.Width-10, 40)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		progressModel, cmd := m.progressBar.Update(msg)
		m.progressBar = progressModel.(progress.Model)
		return m, cmd

	case StartStepMsg:
		m.progress.StartStep()
		return m, nil

	case CompleteStepMsg:
		m.progress.CompleteStep()
		// Check if all done
		if m.progress.IsComplete() {
			m.done = true
			return m, tea.Quit
		}
		return m, nil

	case FailStepMsg:
		m.progress.FailStep(msg.Err)
		m.err = msg.Err
		m.done = true
		return m, tea.Quit

	case SubProgressMsg:
		m.progress.SetSubProgress(msg.Percent, msg.Detail)
		return m, m.progressBar.SetPercent(msg.Percent / 100)

	case DoneMsg:
		m.done = true
		m.err = msg.Err
		return m, tea.Quit
	}

	return m, nil
}

// View renders the progress display
func (m Model) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Text).
		Bold(true).
		MarginBottom(1)
	b.WriteString(titleStyle.Render(m.progress.Title))
	b.WriteString("\n\n")

	// Steps
	indent := "  "
	for _, step := range m.progress.Steps {
		icon := StyledIcon(step.State)
		textStyle := StepStyle(step.State)

		// For in-progress step, use spinner
		if step.State == StateInProgress {
			icon = m.spinner.View()
		}

		// Step line
		line := fmt.Sprintf("%s%s %s", indent, icon, textStyle.Render(step.Name))
		b.WriteString(line)

		// Add detail if present and step is in progress
		if step.State == StateInProgress && step.Detail != "" {
			detailStyle := lipgloss.NewStyle().Foreground(styles.Muted)
			b.WriteString(detailStyle.Render(" - " + step.Detail))
		}
		b.WriteString("\n")

		// Show sub-progress bar for current in-progress step
		if step.State == StateInProgress && m.progress.SubProgress > 0 {
			// Sub-detail line (git output)
			if m.progress.SubDetail != "" {
				subDetailStyle := lipgloss.NewStyle().Foreground(styles.Muted)
				b.WriteString(indent + "    " + subDetailStyle.Render(m.progress.SubDetail) + "\n")
			}
			// Progress bar
			b.WriteString(indent + "  " + m.progressBar.View() + "\n")
		}

	}

	// Final newline
	b.WriteString("\n")

	return b.String()
}

// GetError returns any error that occurred
func (m Model) GetError() error {
	return m.err
}

// IsDone returns true if the operation is complete
func (m Model) IsDone() bool {
	return m.done
}

// GetProgress returns the underlying progress state
func (m Model) GetProgress() *Progress {
	return m.progress
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
