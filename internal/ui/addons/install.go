package addons

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bnema/turtlectl/internal/addons"
	uiprogress "github.com/bnema/turtlectl/internal/ui/progress"
	"github.com/bnema/turtlectl/internal/ui/styles"
)

// Install step indices
const (
	installStepValidate = iota
	installStepClone
	installStepParse
	installStepFinalize
)

// InstallModel is the bubbletea model for addon installation progress
type InstallModel struct {
	spinner     spinner.Model
	progressBar progress.Model
	manager     *addons.Manager
	gitURL      string
	addonName   string

	steps       []uiprogress.Step
	currentStep int
	subProgress float64
	subDetail   string

	done   bool
	err    error
	result *addons.InstallResult
	width  int
}

// NewInstallModel creates a new addon installation progress model
func NewInstallModel(manager *addons.Manager, gitURL, addonName string) InstallModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(30),
		progress.WithoutPercentage(),
	)

	steps := []uiprogress.Step{
		{Name: "Validating URL", State: uiprogress.StatePending},
		{Name: "Cloning repository", State: uiprogress.StatePending},
		{Name: "Parsing metadata", State: uiprogress.StatePending},
		{Name: "Finalizing", State: uiprogress.StatePending},
	}

	return InstallModel{
		spinner:     s,
		progressBar: p,
		manager:     manager,
		gitURL:      gitURL,
		addonName:   addonName,
		steps:       steps,
		currentStep: 0,
		width:       80,
	}
}

// Messages
type (
	installStepDoneMsg struct{ step int }
	installProgressMsg struct {
		percent float64
		detail  string
	}
	installCompleteMsg struct{ result *addons.InstallResult }
	installErrorMsg    struct{ err error }
)

// Init initializes the model
func (m InstallModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.WindowSize(),
		m.startValidation(),
	)
}

func (m InstallModel) startValidation() tea.Cmd {
	return func() tea.Msg {
		// Validation already happened in command, just a visual step
		time.Sleep(50 * time.Millisecond)
		return installStepDoneMsg{step: installStepValidate}
	}
}

func (m InstallModel) startClone() tea.Cmd {
	return func() tea.Msg {
		result, err := m.manager.Install(m.gitURL, nil)
		if err != nil {
			return installErrorMsg{err: err}
		}
		return installCompleteMsg{result: result}
	}
}

// Update handles messages
func (m InstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case installStepDoneMsg:
		m.steps[msg.step].State = uiprogress.StateComplete
		m.subProgress = 0
		m.subDetail = ""

		switch msg.step {
		case installStepValidate:
			m.steps[installStepClone].State = uiprogress.StateInProgress
			m.currentStep = installStepClone
			return m, m.startClone()
		}
		return m, nil

	case installProgressMsg:
		m.subProgress = msg.percent
		m.subDetail = msg.detail
		return m, m.progressBar.SetPercent(msg.percent / 100)

	case installCompleteMsg:
		// Mark all steps as complete
		for i := range m.steps {
			m.steps[i].State = uiprogress.StateComplete
		}
		m.done = true
		m.result = msg.result
		m.addonName = msg.result.Title
		return m, tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
			return tea.Quit()
		})

	case installErrorMsg:
		m.steps[m.currentStep].State = uiprogress.StateError
		m.done = true
		m.err = msg.err
		return m, tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
			return tea.Quit()
		})
	}

	return m, nil
}

// View renders the model
func (m InstallModel) View() string {
	var b strings.Builder

	title := fmt.Sprintf("Installing %s", m.addonName)
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Text).
		Bold(true)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	indent := "  "
	for i, step := range m.steps {
		icon := uiprogress.StyledIcon(step.State)
		textStyle := uiprogress.StepStyle(step.State)

		if step.State == uiprogress.StateInProgress {
			icon = m.spinner.View()
		}

		line := fmt.Sprintf("%s%s %s", indent, icon, textStyle.Render(step.Name))
		b.WriteString(line)
		b.WriteString("\n")

		// Show sub-progress bar for clone step
		if i == installStepClone && step.State == uiprogress.StateInProgress && m.subProgress > 0 {
			if m.subDetail != "" {
				subDetailStyle := lipgloss.NewStyle().Foreground(styles.Muted)
				b.WriteString(indent + "    " + subDetailStyle.Render(m.subDetail) + "\n")
			}
			b.WriteString(indent + "  " + m.progressBar.View() + "\n")
		}
	}

	if m.done {
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(uiprogress.FormatError(m.err.Error()))
		} else if m.result != nil {
			b.WriteString(uiprogress.FormatSuccess(fmt.Sprintf("Installed %s", m.result.Title)))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// GetError returns any error that occurred
func (m InstallModel) GetError() error {
	return m.err
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
