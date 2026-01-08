package launcher

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bnema/turtlectl/internal/launcher"
	uiprogress "github.com/bnema/turtlectl/internal/ui/progress"
	"github.com/bnema/turtlectl/internal/ui/styles"
)

const (
	stepDirs = iota
	stepCheck
	stepDownload
	stepDesktop
)

// InstallModel is the bubbletea model for launcher installation
type InstallModel struct {
	spinner     spinner.Model
	progressBar progress.Model
	launcher    *launcher.Launcher

	steps       []uiprogress.Step
	currentStep int
	subProgress float64
	subDetail   string

	done         bool
	err          error
	updateResult *launcher.UpdateResult
	width        int
}

// NewInstallModel creates a new installation progress model
func NewInstallModel(l *launcher.Launcher) InstallModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(30),
		progress.WithoutPercentage(),
	)

	steps := []uiprogress.Step{
		{Name: "Creating directories", State: uiprogress.StatePending},
		{Name: "Checking for updates", State: uiprogress.StatePending},
		{Name: "Downloading launcher", State: uiprogress.StatePending},
		{Name: "Installing desktop entry", State: uiprogress.StatePending},
	}

	return InstallModel{
		spinner:     s,
		progressBar: p,
		launcher:    l,
		steps:       steps,
		currentStep: 0,
		width:       80,
	}
}

// Messages
type (
	stepDoneMsg struct {
		step   int
		result *launcher.UpdateResult
	}
	progressMsg struct {
		downloaded int64
		total      int64
	}
	errorMsg struct{ err error }
)

// Init initializes the model
func (m InstallModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.WindowSize(),
		m.startDirs(),
	)
}

func (m InstallModel) startDirs() tea.Cmd {
	return func() tea.Msg {
		if err := m.launcher.EnsureLauncherDirs(); err != nil {
			return errorMsg{err: err}
		}
		return stepDoneMsg{step: stepDirs}
	}
}

func (m InstallModel) startCheck() tea.Cmd {
	return func() tea.Msg {
		result, err := m.launcher.UpdateAppImageWithProgress(nil)
		if err != nil {
			return errorMsg{err: err}
		}
		return stepDoneMsg{step: stepCheck, result: result}
	}
}

func (m InstallModel) startDownload() tea.Cmd {
	return func() tea.Msg {
		_, err := m.launcher.UpdateAppImageWithProgress(nil)
		if err != nil {
			return errorMsg{err: err}
		}
		return stepDoneMsg{step: stepDownload}
	}
}

func (m InstallModel) startDesktop() tea.Cmd {
	return func() tea.Msg {
		if err := m.launcher.InstallDesktop(); err != nil {
			return errorMsg{err: err}
		}
		return stepDoneMsg{step: stepDesktop}
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

	case stepDoneMsg:
		m.steps[msg.step].State = uiprogress.StateComplete
		m.subProgress = 0
		m.subDetail = ""

		switch msg.step {
		case stepDirs:
			m.steps[stepCheck].State = uiprogress.StateInProgress
			m.currentStep = stepCheck
			return m, m.startCheck()

		case stepCheck:
			m.updateResult = msg.result
			if msg.result != nil && msg.result.AlreadyLatest {
				// Skip download step
				m.steps[stepDownload].State = uiprogress.StateComplete
				m.steps[stepDownload].Name = "Already up to date"
				m.steps[stepDesktop].State = uiprogress.StateInProgress
				m.currentStep = stepDesktop
				return m, m.startDesktop()
			}
			// Need to download
			m.steps[stepDownload].State = uiprogress.StateInProgress
			m.currentStep = stepDownload
			return m, m.startDownload()

		case stepDownload:
			m.steps[stepDesktop].State = uiprogress.StateInProgress
			m.currentStep = stepDesktop
			return m, m.startDesktop()

		case stepDesktop:
			m.done = true
			return m, tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
				return tea.Quit()
			})
		}

	case progressMsg:
		if msg.total > 0 {
			m.subProgress = float64(msg.downloaded) / float64(msg.total) * 100
			m.subDetail = fmt.Sprintf("%s / %s", formatBytes(msg.downloaded), formatBytes(msg.total))
		}
		return m, m.progressBar.SetPercent(m.subProgress / 100)

	case errorMsg:
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

	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Text).
		Bold(true)
	b.WriteString(titleStyle.Render("Installing Turtle WoW Launcher"))
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

		// Show progress bar for download step
		if i == stepDownload && step.State == uiprogress.StateInProgress && m.subProgress > 0 {
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
		} else {
			b.WriteString(uiprogress.FormatSuccess("Installation complete! Launch from your app menu."))
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

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
