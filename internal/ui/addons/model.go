package addons

import (
	"fmt"
	"strings"

	"github.com/bnema/turtlectl/internal/addons"
	"github.com/bnema/turtlectl/internal/ui/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// View states
type viewState int

const (
	viewList viewState = iota
	viewInstall
	viewConfirmRemove
	viewProgress
	viewInfo
)

// addonItem implements list.Item for bubbles/list
type addonItem struct {
	addon     *addons.Addon
	hasUpdate bool
}

func (i addonItem) Title() string {
	name := i.addon.Name
	if i.addon.Title != "" && i.addon.Title != i.addon.Name {
		name = i.addon.Title
	}
	return name
}

func (i addonItem) Description() string {
	var parts []string

	if i.addon.Version != "" {
		parts = append(parts, "v"+i.addon.Version)
	}
	if i.addon.Author != "" {
		parts = append(parts, "by "+i.addon.Author)
	}

	// Determine status: default > tracked > untracked
	if addons.IsDefaultAddon(i.addon.Name) {
		parts = append(parts, styles.FormatAddonStatusEx(styles.AddonStatusDefault))
	} else if i.addon.GitURL != "" {
		parts = append(parts, styles.FormatAddonStatusEx(styles.AddonStatusTracked))
	} else {
		parts = append(parts, styles.FormatAddonStatusEx(styles.AddonStatusUntracked))
	}

	// Show update indicator
	if i.hasUpdate {
		parts = append(parts, styles.FormatUpdateAvailable())
	}

	return strings.Join(parts, " | ")
}

func (i addonItem) FilterValue() string {
	return i.addon.Name + " " + i.addon.Title
}

// KeyMap defines keyboard shortcuts
type KeyMap struct {
	Install   key.Binding
	Remove    key.Binding
	Update    key.Binding
	UpdateAll key.Binding
	Info      key.Binding
	Repair    key.Binding
	Quit      key.Binding
	Back      key.Binding
	Confirm   key.Binding
	Help      key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Install: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "install"),
		),
		Remove: key.NewBinding(
			key.WithKeys("d", "delete"),
			key.WithHelp("d", "remove"),
		),
		Update: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "update"),
		),
		UpdateAll: key.NewBinding(
			key.WithKeys("U"),
			key.WithHelp("U", "update all"),
		),
		Info: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "info"),
		),
		Repair: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "repair"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// Model is the main TUI model
type Model struct {
	manager   *addons.Manager
	list      list.Model
	textInput textinput.Model
	spinner   spinner.Model
	keys      KeyMap

	state         viewState
	width, height int

	// For operations
	selectedAddon    *addons.Addon
	statusMsg        string
	errorMsg         string
	progressMsg      string
	updatesAvailable map[string]bool // addon name -> has update
	checkingUpdates  bool
}

// NewModel creates a new TUI model
func NewModel(manager *addons.Manager) Model {
	// Setup list
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(styles.Primary).
		BorderForeground(styles.Primary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(styles.Muted).
		BorderForeground(styles.Primary)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Addons"
	l.Styles.Title = styles.Title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	// Setup text input
	ti := textinput.New()
	ti.Placeholder = "https://github.com/user/addon.git"
	ti.CharLimit = 256
	ti.Width = 50

	// Setup spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	return Model{
		manager:          manager,
		list:             l,
		textInput:        ti,
		spinner:          s,
		keys:             DefaultKeyMap(),
		state:            viewList,
		updatesAvailable: make(map[string]bool),
		checkingUpdates:  true,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadAddons,
		m.checkUpdates,
		m.spinner.Tick,
	)
}

// loadAddons loads addons from the manager
func (m Model) loadAddons() tea.Msg {
	addons, err := m.manager.ListInstalled()
	if err != nil {
		return errMsg{err}
	}
	return addonsLoadedMsg{addons}
}

// checkUpdates checks all tracked addons for available updates
func (m Model) checkUpdates() tea.Msg {
	results := m.manager.CheckAllUpdates()
	return updatesCheckedMsg{results}
}

// Messages
type addonsLoadedMsg struct {
	addons []*addons.Addon
}

type updatesCheckedMsg struct {
	results []addons.CheckUpdatesResult
}

type errMsg struct {
	err error
}

type statusMsg string

type operationCompleteMsg struct {
	success bool
	message string
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		h, v := styles.App.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-2)
		return m, nil

	case tea.KeyMsg:
		// Handle global keys
		if key.Matches(msg, m.keys.Quit) {
			if m.state == viewList {
				return m, tea.Quit
			}
			m.state = viewList
			m.errorMsg = ""
			m.statusMsg = ""
			return m, nil
		}

		if key.Matches(msg, m.keys.Back) {
			if m.state != viewList {
				m.state = viewList
				m.errorMsg = ""
				m.statusMsg = ""
				return m, nil
			}
		}

		// State-specific handling
		switch m.state {
		case viewList:
			return m.updateList(msg)
		case viewInstall:
			return m.updateInstall(msg)
		case viewConfirmRemove:
			return m.updateConfirmRemove(msg)
		case viewInfo:
			return m.updateInfo(msg)
		}

	case addonsLoadedMsg:
		items := make([]list.Item, len(msg.addons))
		for i, addon := range msg.addons {
			items[i] = addonItem{addon: addon, hasUpdate: m.updatesAvailable[addon.Name]}
		}
		m.list.SetItems(items)
		return m, nil

	case updatesCheckedMsg:
		m.checkingUpdates = false
		m.updatesAvailable = make(map[string]bool)
		updateCount := 0
		for _, result := range msg.results {
			if result.HasUpdate && result.Error == nil {
				m.updatesAvailable[result.Name] = true
				updateCount++
			}
		}
		// Refresh list items to show update indicators
		if updateCount > 0 {
			m.statusMsg = fmt.Sprintf("%d update(s) available", updateCount)
			return m, m.loadAddons
		}
		return m, nil

	case errMsg:
		m.errorMsg = msg.err.Error()
		m.state = viewList
		return m, nil

	case statusMsg:
		m.statusMsg = string(msg)
		return m, nil

	case operationCompleteMsg:
		if msg.success {
			m.statusMsg = msg.message
		} else {
			m.errorMsg = msg.message
		}
		m.state = viewList
		return m, m.loadAddons

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Install):
		m.state = viewInstall
		m.textInput.Focus()
		m.textInput.SetValue("")
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Remove):
		if item, ok := m.list.SelectedItem().(addonItem); ok {
			m.selectedAddon = item.addon
			m.state = viewConfirmRemove
		}
		return m, nil

	case key.Matches(msg, m.keys.Update):
		if item, ok := m.list.SelectedItem().(addonItem); ok {
			m.selectedAddon = item.addon
			m.state = viewProgress
			m.progressMsg = "Updating " + item.addon.Name + "..."
			return m, m.updateAddon(item.addon.Name)
		}
		return m, nil

	case key.Matches(msg, m.keys.UpdateAll):
		m.state = viewProgress
		m.progressMsg = "Updating all addons..."
		return m, m.updateAllAddons

	case key.Matches(msg, m.keys.Info):
		if item, ok := m.list.SelectedItem().(addonItem); ok {
			m.selectedAddon = item.addon
			m.state = viewInfo
		}
		return m, nil

	case key.Matches(msg, m.keys.Repair):
		m.state = viewProgress
		m.progressMsg = "Repairing addon database..."
		return m, m.repairAddons
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateInstall(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		url := m.textInput.Value()
		if url == "" {
			return m, nil
		}
		m.state = viewProgress
		m.progressMsg = "Installing addon..."
		return m, m.installAddon(url)

	case tea.KeyEsc:
		m.state = viewList
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) updateConfirmRemove(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Confirm):
		if m.selectedAddon != nil {
			m.state = viewProgress
			m.progressMsg = "Removing " + m.selectedAddon.Name + "..."
			return m, m.removeAddon(m.selectedAddon.Name)
		}
		m.state = viewList
		return m, nil

	case key.Matches(msg, m.keys.Back), msg.String() == "n":
		m.state = viewList
		m.selectedAddon = nil
		return m, nil
	}

	return m, nil
}

func (m Model) updateInfo(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) || msg.Type == tea.KeyEnter {
		m.state = viewList
		m.selectedAddon = nil
	}
	return m, nil
}

// Commands

func (m Model) installAddon(url string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.manager.Install(url, nil)
		if err != nil {
			return operationCompleteMsg{false, err.Error()}
		}
		return operationCompleteMsg{true, fmt.Sprintf("Addon %s installed successfully", result.Name)}
	}
}

func (m Model) removeAddon(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.manager.Remove(name, true) // Always backup
		if err != nil {
			return operationCompleteMsg{false, err.Error()}
		}
		return operationCompleteMsg{true, "Addon removed (backup created)"}
	}
}

func (m Model) updateAddon(name string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.manager.Update(name, nil)
		if err != nil {
			return operationCompleteMsg{false, err.Error()}
		}
		if result.AlreadyUpToDate {
			return operationCompleteMsg{true, "Addon already up to date"}
		}
		return operationCompleteMsg{true, "Addon updated successfully"}
	}
}

func (m Model) updateAllAddons() tea.Msg {
	result := m.manager.UpdateAll()
	if result.Failed > 0 {
		return operationCompleteMsg{false, fmt.Sprintf("Updated %d, failed %d: %v", result.Updated, result.Failed, result.Errors)}
	}
	return operationCompleteMsg{true, fmt.Sprintf("Updated %d addons", result.Updated)}
}

func (m Model) repairAddons() tea.Msg {
	result, err := m.manager.Repair()
	if err != nil {
		return operationCompleteMsg{false, err.Error()}
	}

	if result.IssuesFound == 0 {
		return operationCompleteMsg{true, "No issues found"}
	}

	msg := fmt.Sprintf("Fixed %d issues: %d orphaned, %d untracked, %d corrupted",
		result.IssuesFound, len(result.OrphanedEntries), len(result.UntrackedAddons), len(result.CorruptedRepos))
	return operationCompleteMsg{true, msg}
}

// View renders the UI
func (m Model) View() string {
	var content string

	switch m.state {
	case viewList:
		content = m.viewList()
	case viewInstall:
		content = m.viewInstall()
	case viewConfirmRemove:
		content = m.viewConfirmRemove()
	case viewProgress:
		content = m.viewProgress()
	case viewInfo:
		content = m.viewInfo()
	}

	return styles.App.Render(content)
}

func (m Model) viewList() string {
	var s strings.Builder

	s.WriteString(m.list.View())

	// Status/error messages
	if m.checkingUpdates {
		s.WriteString("\n" + m.spinner.View() + " " + styles.MutedText.Render("Checking for updates..."))
	} else if m.errorMsg != "" {
		s.WriteString("\n" + styles.FormatError(m.errorMsg))
	} else if m.statusMsg != "" {
		s.WriteString("\n" + styles.FormatSuccess(m.statusMsg))
	}

	// Help
	help := "\n" + styles.Help.Render("i:install  d:remove  u:update  U:update all  r:repair  ?:help  q:quit")
	s.WriteString(help)

	return s.String()
}

func (m Model) viewInstall() string {
	var s strings.Builder

	s.WriteString(styles.Title.Render("Install Addon") + "\n\n")
	s.WriteString("Enter git repository URL:\n\n")
	s.WriteString(m.textInput.View() + "\n\n")
	s.WriteString(styles.Help.Render("enter:install  esc:cancel"))

	return s.String()
}

func (m Model) viewConfirmRemove() string {
	var s strings.Builder

	name := ""
	if m.selectedAddon != nil {
		name = m.selectedAddon.Name
	}

	s.WriteString(styles.Title.Render("Remove Addon") + "\n\n")
	s.WriteString(fmt.Sprintf("Are you sure you want to remove %s?\n", styles.Highlighted.Render(name)))
	s.WriteString("A backup will be created.\n\n")
	s.WriteString(styles.Help.Render("y:confirm  n/esc:cancel"))

	return s.String()
}

func (m Model) viewProgress() string {
	var s strings.Builder

	s.WriteString(m.spinner.View() + " " + m.progressMsg)

	return s.String()
}

func (m Model) viewInfo() string {
	var s strings.Builder

	if m.selectedAddon == nil {
		return "No addon selected"
	}

	a := m.selectedAddon

	s.WriteString(styles.Title.Render("Addon Info") + "\n\n")

	// Name/Title
	s.WriteString(styles.AddonName.Render(a.Name) + "\n")
	if a.Title != "" && a.Title != a.Name {
		s.WriteString(styles.MutedText.Render(a.Title) + "\n")
	}
	s.WriteString("\n")

	// Details
	if a.Version != "" {
		s.WriteString(fmt.Sprintf("Version:   %s\n", a.Version))
	}
	if a.Author != "" {
		s.WriteString(fmt.Sprintf("Author:    %s\n", a.Author))
	}
	if a.Notes != "" {
		s.WriteString(fmt.Sprintf("Notes:     %s\n", a.Notes))
	}
	if a.GitURL != "" {
		s.WriteString(fmt.Sprintf("Git URL:   %s\n", a.GitURL))
	}
	if !a.InstalledAt.IsZero() {
		s.WriteString(fmt.Sprintf("Installed: %s\n", a.InstalledAt.Format("2006-01-02 15:04")))
	}
	if !a.UpdatedAt.IsZero() {
		s.WriteString(fmt.Sprintf("Updated:   %s\n", a.UpdatedAt.Format("2006-01-02 15:04")))
	}
	s.WriteString(fmt.Sprintf("Path:      %s\n", a.Path))

	s.WriteString("\n" + styles.Help.Render("esc/enter:back"))

	return s.String()
}
