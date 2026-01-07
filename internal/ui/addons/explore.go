package addons

import (
	"fmt"
	"strings"

	"github.com/bnema/turtlectl/internal/addons"
	"github.com/bnema/turtlectl/internal/ui/styles"
	"github.com/bnema/turtlectl/internal/wiki"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// exploreState represents the current view state
type exploreState int

const (
	exploreViewList exploreState = iota
	exploreViewDetails
	exploreViewInstalling
)

// exploreItem implements list.Item for wiki addons
type exploreItem struct {
	addon wiki.WikiAddon
}

func (i exploreItem) Title() string {
	name := i.addon.Name

	// Build suffix with badges
	var badges []string
	if i.addon.IsNew() {
		badges = append(badges, styles.FormatNewBadge())
	}
	if i.addon.IsInstalled {
		badges = append(badges, styles.FormatInstalledBadge())
	}

	if len(badges) > 0 {
		return name + "  " + strings.Join(badges, " ")
	}
	return name
}

func (i exploreItem) Description() string {
	var parts []string

	if i.addon.Author != "" {
		parts = append(parts, "by "+i.addon.Author)
	}

	if i.addon.Stars > 0 {
		parts = append(parts, styles.FormatStars(i.addon.Stars))
	}

	if i.addon.Description != "" {
		// Truncate description if too long
		desc := i.addon.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		parts = append(parts, desc)
	}

	return strings.Join(parts, " | ")
}

func (i exploreItem) FilterValue() string {
	return i.addon.Name + " " + i.addon.Author + " " + i.addon.Description
}

// ExploreKeyMap defines keyboard shortcuts for explore view
type ExploreKeyMap struct {
	Install key.Binding
	Details key.Binding
	Refresh key.Binding
	Quit    key.Binding
	Back    key.Binding
}

// DefaultExploreKeyMap returns the default key bindings
func DefaultExploreKeyMap() ExploreKeyMap {
	return ExploreKeyMap{
		Install: key.NewBinding(
			key.WithKeys("i", "enter"),
			key.WithHelp("i/enter", "install"),
		),
		Details: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "details"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	}
}

// ExploreModel is the TUI model for browsing wiki addons
type ExploreModel struct {
	addonManager *addons.Manager
	registry     *wiki.Registry
	list         list.Model
	spinner      spinner.Model
	keys         ExploreKeyMap

	state         exploreState
	width, height int

	// Data
	wikiAddons    []wiki.WikiAddon
	selectedAddon *wiki.WikiAddon
	registryInfo  wiki.RegistryInfo

	// Status
	loading     bool
	refreshing  bool
	statusMsg   string
	errorMsg    string
	progressMsg string
}

// NewExploreModel creates a new explore TUI model
func NewExploreModel(manager *addons.Manager, registry *wiki.Registry, refresh bool) ExploreModel {
	// Setup list
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(styles.Primary).
		BorderForeground(styles.Primary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(styles.Muted).
		BorderForeground(styles.Primary)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Explore Addons"
	l.Styles.Title = styles.Title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	// Setup spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	return ExploreModel{
		addonManager: manager,
		registry:     registry,
		list:         l,
		spinner:      s,
		keys:         DefaultExploreKeyMap(),
		state:        exploreViewList,
		loading:      true,
		refreshing:   refresh,
	}
}

// Init initializes the model
func (m ExploreModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadAddonsCmd(),
		m.spinner.Tick,
	)
}

// Messages
type exploreAddonsLoadedMsg struct {
	addons       []wiki.WikiAddon
	registryInfo wiki.RegistryInfo
	err          error
}

type exploreInstallCompleteMsg struct {
	success bool
	name    string
	err     error
}

// loadAddonsCmd loads addons from the registry
func (m ExploreModel) loadAddonsCmd() tea.Cmd {
	return func() tea.Msg {
		// Fetch addons from registry
		addons, err := m.registry.GetAddons(m.refreshing)
		if err != nil {
			return exploreAddonsLoadedMsg{err: err}
		}

		// Mark installed addons
		installedURLs := m.getInstalledURLs()
		wiki.MarkInstalled(addons, installedURLs)

		// Sort alphabetically
		wiki.SortAddons(addons)

		return exploreAddonsLoadedMsg{
			addons:       addons,
			registryInfo: m.registry.GetInfo(),
		}
	}
}

// getInstalledURLs returns a map of installed addon URLs
func (m ExploreModel) getInstalledURLs() map[string]bool {
	urls := make(map[string]bool)
	installed, err := m.addonManager.ListInstalled()
	if err != nil {
		return urls
	}
	for _, addon := range installed {
		if addon.GitURL != "" {
			urls[addon.GitURL] = true
			// Also add normalized version
			normalized := strings.TrimSuffix(addon.GitURL, ".git")
			urls[normalized] = true
		}
	}
	return urls
}

// installAddon installs the selected addon
func (m ExploreModel) installAddon(url string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.addonManager.Install(url, nil)
		if err != nil {
			return exploreInstallCompleteMsg{success: false, err: err}
		}
		return exploreInstallCompleteMsg{success: true, name: result.Name}
	}
}

// Update handles messages
func (m ExploreModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		h, v := styles.App.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-4)
		return m, nil

	case tea.KeyMsg:
		// Handle global keys
		if key.Matches(msg, m.keys.Quit) {
			if m.state == exploreViewList {
				return m, tea.Quit
			}
			m.state = exploreViewList
			m.errorMsg = ""
			m.statusMsg = ""
			return m, nil
		}

		if key.Matches(msg, m.keys.Back) {
			if m.state != exploreViewList {
				m.state = exploreViewList
				m.errorMsg = ""
				m.statusMsg = ""
				return m, nil
			}
		}

		// State-specific handling
		if !m.loading {
			switch m.state {
			case exploreViewList:
				return m.updateList(msg)
			case exploreViewDetails:
				return m.updateDetails(msg)
			}
		}

	case exploreAddonsLoadedMsg:
		m.loading = false
		m.refreshing = false
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
			return m, nil
		}
		m.wikiAddons = msg.addons
		m.registryInfo = msg.registryInfo

		// Update list items
		items := make([]list.Item, len(msg.addons))
		for i, addon := range msg.addons {
			items[i] = exploreItem{addon: addon}
		}
		m.list.SetItems(items)

		// Update title with counts
		m.list.Title = fmt.Sprintf("Explore Addons (%d available", len(msg.addons))
		if msg.registryInfo.NewAddons > 0 {
			m.list.Title += fmt.Sprintf(", %d new", msg.registryInfo.NewAddons)
		}
		m.list.Title += ")"

		return m, nil

	case exploreInstallCompleteMsg:
		m.state = exploreViewList
		m.loading = false
		if msg.err != nil {
			m.errorMsg = "Install failed: " + msg.err.Error()
		} else {
			m.statusMsg = fmt.Sprintf("Installed %s successfully", msg.name)
			// Reload to update installed status
			m.loading = true
			return m, m.loadAddonsCmd()
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update list
	if !m.loading {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m ExploreModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Install):
		if item, ok := m.list.SelectedItem().(exploreItem); ok {
			if item.addon.IsInstalled {
				m.statusMsg = "Addon is already installed"
				return m, nil
			}
			m.selectedAddon = &item.addon
			m.state = exploreViewInstalling
			m.loading = true
			m.progressMsg = "Installing " + item.addon.Name + "..."
			m.errorMsg = ""
			m.statusMsg = ""
			return m, tea.Batch(
				m.installAddon(item.addon.URL),
				m.spinner.Tick,
			)
		}
		return m, nil

	case key.Matches(msg, m.keys.Details):
		if item, ok := m.list.SelectedItem().(exploreItem); ok {
			m.selectedAddon = &item.addon
			m.state = exploreViewDetails
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		m.refreshing = true
		m.statusMsg = ""
		m.errorMsg = ""
		return m, tea.Batch(
			m.loadAddonsCmd(),
			m.spinner.Tick,
		)
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ExploreModel) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Details):
		m.state = exploreViewList
		m.selectedAddon = nil
		return m, nil

	case key.Matches(msg, m.keys.Install):
		if m.selectedAddon != nil && !m.selectedAddon.IsInstalled {
			m.state = exploreViewInstalling
			m.loading = true
			m.progressMsg = "Installing " + m.selectedAddon.Name + "..."
			return m, tea.Batch(
				m.installAddon(m.selectedAddon.URL),
				m.spinner.Tick,
			)
		}
		return m, nil
	}

	return m, nil
}

// View renders the UI
func (m ExploreModel) View() string {
	var content string

	switch m.state {
	case exploreViewList:
		content = m.viewList()
	case exploreViewDetails:
		content = m.viewDetails()
	case exploreViewInstalling:
		content = m.viewInstalling()
	}

	return styles.App.Render(content)
}

func (m ExploreModel) viewList() string {
	var s strings.Builder

	if m.loading {
		msg := "Loading addons..."
		if m.refreshing {
			msg = "Refreshing addons..."
		}
		s.WriteString(m.spinner.View() + " " + msg)
		return s.String()
	}

	s.WriteString(m.list.View())

	// Registry status
	if m.registryInfo.IsStale && m.registryInfo.HasCache {
		days := int(m.registryInfo.Age.Hours() / 24)
		if days > 0 {
			s.WriteString("\n" + styles.FormatWarning(fmt.Sprintf("Cache is %d day(s) old. Press 'r' to refresh.", days)))
		}
	}

	// Status/error messages
	if m.errorMsg != "" {
		s.WriteString("\n" + styles.FormatError(m.errorMsg))
	} else if m.statusMsg != "" {
		s.WriteString("\n" + styles.FormatSuccess(m.statusMsg))
	}

	// Help
	help := "\n" + styles.Help.Render("/:filter  i/enter:install  d:details  r:refresh  q:quit")
	s.WriteString(help)

	return s.String()
}

func (m ExploreModel) viewDetails() string {
	var s strings.Builder

	if m.selectedAddon == nil {
		return "No addon selected"
	}

	a := m.selectedAddon

	s.WriteString(styles.Title.Render("Addon Details") + "\n\n")

	// Name with badges
	nameLine := styles.AddonName.Render(a.Name)
	if a.IsNew() {
		nameLine += "  " + styles.FormatNewBadge()
	}
	if a.IsInstalled {
		nameLine += "  " + styles.FormatInstalledBadge()
	}
	s.WriteString(nameLine + "\n\n")

	// Details
	if a.Author != "" {
		s.WriteString(fmt.Sprintf("Author:      %s\n", a.Author))
	}
	if a.Version != "" {
		s.WriteString(fmt.Sprintf("Version:     %s\n", a.Version))
	}
	if a.Stars > 0 {
		s.WriteString(fmt.Sprintf("Stars:       %s\n", styles.FormatStars(a.Stars)))
	}
	if a.Category != "" {
		s.WriteString(fmt.Sprintf("Category:    %s\n", a.Category))
	}
	s.WriteString(fmt.Sprintf("URL:         %s\n", a.URL))

	if a.Description != "" {
		s.WriteString(fmt.Sprintf("\nDescription:\n%s\n", a.Description))
	}

	if !a.AddedAt.IsZero() {
		s.WriteString(fmt.Sprintf("\nAdded:       %s\n", a.AddedAt.Format("2006-01-02")))
	}

	// Help
	s.WriteString("\n")
	if a.IsInstalled {
		s.WriteString(styles.Help.Render("esc/d:back  q:quit"))
	} else {
		s.WriteString(styles.Help.Render("i:install  esc/d:back  q:quit"))
	}

	return s.String()
}

func (m ExploreModel) viewInstalling() string {
	return m.spinner.View() + " " + m.progressMsg
}
