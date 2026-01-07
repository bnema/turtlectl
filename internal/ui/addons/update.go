package addons

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bnema/turtlectl/internal/addons"
	uiprogress "github.com/bnema/turtlectl/internal/ui/progress"
	"github.com/bnema/turtlectl/internal/ui/styles"
)

// UpdateSingleModel is the bubbletea model for single addon update
type UpdateSingleModel struct {
	spinner   spinner.Model
	manager   *addons.Manager
	addonName string

	steps       []uiprogress.Step
	currentStep int

	done   bool
	err    error
	result *addons.UpdateResult
}

// NewUpdateSingleModel creates a new single addon update model
func NewUpdateSingleModel(manager *addons.Manager, name string) UpdateSingleModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	steps := []uiprogress.Step{
		{Name: "Checking for updates", State: uiprogress.StatePending},
		{Name: "Fetching changes", State: uiprogress.StatePending},
		{Name: "Applying updates", State: uiprogress.StatePending},
	}

	return UpdateSingleModel{
		spinner:     s,
		manager:     manager,
		addonName:   name,
		steps:       steps,
		currentStep: 0,
	}
}

type updateSingleDoneMsg struct {
	result *addons.UpdateResult
	err    error
}

// Init initializes the model
func (m UpdateSingleModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.doUpdate(),
	)
}

func (m UpdateSingleModel) doUpdate() tea.Cmd {
	return func() tea.Msg {
		result, err := m.manager.Update(m.addonName, nil)
		return updateSingleDoneMsg{result: result, err: err}
	}
}

// Update handles messages
func (m UpdateSingleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		// Animate through steps while waiting
		if !m.done && m.currentStep < len(m.steps)-1 {
			m.steps[m.currentStep].State = uiprogress.StateInProgress
		}

		return m, cmd

	case updateSingleDoneMsg:
		m.done = true
		m.err = msg.err
		m.result = msg.result

		if msg.err != nil {
			m.steps[m.currentStep].State = uiprogress.StateError
		} else {
			for i := range m.steps {
				m.steps[i].State = uiprogress.StateComplete
			}
		}

		return m, tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
			return tea.Quit()
		})
	}

	return m, nil
}

// View renders the model
func (m UpdateSingleModel) View() string {
	var b strings.Builder

	title := fmt.Sprintf("Updating %s", m.addonName)
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Text).
		Bold(true)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	indent := "  "
	for _, step := range m.steps {
		icon := uiprogress.StyledIcon(step.State)
		textStyle := uiprogress.StepStyle(step.State)

		if step.State == uiprogress.StateInProgress {
			icon = m.spinner.View()
		}

		line := fmt.Sprintf("%s%s %s", indent, icon, textStyle.Render(step.Name))
		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.done {
		b.WriteString("\n")
		if m.err != nil {
			b.WriteString(uiprogress.FormatError(m.err.Error()))
		} else if m.result != nil {
			if m.result.AlreadyUpToDate {
				b.WriteString(uiprogress.FormatSuccess(fmt.Sprintf("%s is already up to date", m.addonName)))
			} else {
				b.WriteString(uiprogress.FormatSuccess(fmt.Sprintf("Updated %s", m.addonName)))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// GetError returns any error that occurred
func (m UpdateSingleModel) GetError() error {
	return m.err
}

// UpdateAllModel is the bubbletea model for updating all addons
type UpdateAllModel struct {
	spinner spinner.Model
	manager *addons.Manager

	addonsList  []string
	current     int
	currentName string

	done    bool
	err     error
	result  *addons.UpdateAllResult
	errors  []string
	updated []string
	skipped []string
}

// NewUpdateAllModel creates a new update all addons model
func NewUpdateAllModel(manager *addons.Manager) UpdateAllModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	addonList := manager.GetTrackedAddons()

	return UpdateAllModel{
		spinner:    s,
		manager:    manager,
		addonsList: addonList,
		current:    0,
	}
}

type (
	updateAllStartMsg struct{}
	updateAllDoneMsg  struct {
		result *addons.UpdateAllResult
	}
	updateOneMsg struct {
		name    string
		updated bool
		skipped bool
		err     error
	}
)

// Init initializes the model
func (m UpdateAllModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg { return updateAllStartMsg{} },
	)
}

func (m UpdateAllModel) updateNext() tea.Cmd {
	if m.current >= len(m.addonsList) {
		return func() tea.Msg {
			return updateAllDoneMsg{result: &addons.UpdateAllResult{
				Updated: len(m.updated),
				Skipped: len(m.skipped),
				Failed:  len(m.errors),
				Errors:  m.errors,
			}}
		}
	}

	name := m.addonsList[m.current]
	return func() tea.Msg {
		result, err := m.manager.Update(name, nil)
		if err != nil {
			return updateOneMsg{name: name, err: err}
		}
		return updateOneMsg{
			name:    name,
			updated: result.Updated,
			skipped: result.AlreadyUpToDate,
		}
	}
}

// Update handles messages
func (m UpdateAllModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case updateAllStartMsg:
		if len(m.addonsList) == 0 {
			m.done = true
			return m, tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
				return tea.Quit()
			})
		}
		m.currentName = m.addonsList[0]
		return m, m.updateNext()

	case updateOneMsg:
		if msg.err != nil {
			m.errors = append(m.errors, fmt.Sprintf("%s: %v", msg.name, msg.err))
		} else if msg.skipped {
			m.skipped = append(m.skipped, msg.name)
		} else if msg.updated {
			m.updated = append(m.updated, msg.name)
		}

		m.current++
		if m.current < len(m.addonsList) {
			m.currentName = m.addonsList[m.current]
			return m, m.updateNext()
		}
		return m, m.updateNext() // Will trigger done

	case updateAllDoneMsg:
		m.done = true
		m.result = msg.result
		return m, tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
			return tea.Quit()
		})
	}

	return m, nil
}

// View renders the model
func (m UpdateAllModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Text).
		Bold(true)
	b.WriteString(titleStyle.Render("Updating all addons"))
	b.WriteString("\n\n")

	if len(m.addonsList) == 0 {
		b.WriteString(uiprogress.FormatWarning("No tracked addons to update"))
		b.WriteString("\n")
		return b.String()
	}

	// Progress indicator
	if !m.done {
		progress := fmt.Sprintf("%d/%d", m.current+1, len(m.addonsList))
		progressStyle := lipgloss.NewStyle().Foreground(styles.Muted)
		line := fmt.Sprintf("  %s Updating %s %s",
			m.spinner.View(),
			progressStyle.Render(progress+":"),
			styles.NormalText.Bold(true).Render(m.currentName),
		)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Results when done
	if m.done {
		for _, name := range m.updated {
			b.WriteString(uiprogress.FormatSuccess(fmt.Sprintf("Updated %s", name)))
			b.WriteString("\n")
		}

		if len(m.skipped) > 0 {
			skipStyle := lipgloss.NewStyle().Foreground(styles.Muted)
			b.WriteString(skipStyle.Render(fmt.Sprintf("  %d addon(s) already up to date", len(m.skipped))))
			b.WriteString("\n")
		}

		for _, errMsg := range m.errors {
			b.WriteString(uiprogress.FormatError(errMsg))
			b.WriteString("\n")
		}

		b.WriteString("\n")
		summary := fmt.Sprintf("Updated: %d, Skipped: %d, Failed: %d",
			len(m.updated), len(m.skipped), len(m.errors))
		summaryStyle := lipgloss.NewStyle().Foreground(styles.Muted)
		b.WriteString(summaryStyle.Render("  " + summary))
		b.WriteString("\n")
	}

	return b.String()
}

// GetError returns any error that occurred
func (m UpdateAllModel) GetError() error {
	return m.err
}
