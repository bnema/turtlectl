package addons

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

var (
	ErrAddonNotFound = errors.New("addon not found")
	ErrAddonExists   = errors.New("addon already exists")
	ErrInvalidURL    = errors.New("invalid git URL")
	ErrAddonsDir     = errors.New("failed to access addons directory")
)

// Manager handles addon operations
type Manager struct {
	gameDir   string
	addonsDir string
	dataDir   string
	store     *StoreManager
	backup    *BackupManager
	log       *log.Logger
}

// NewManager creates a new addon manager
func NewManager(gameDir, dataDir string, logger *log.Logger) *Manager {
	addonsDir := filepath.Join(gameDir, "Interface", "AddOns")

	m := &Manager{
		gameDir:   gameDir,
		addonsDir: addonsDir,
		dataDir:   dataDir,
		store:     NewStoreManager(dataDir),
		backup:    NewBackupManager(dataDir),
		log:       logger,
	}

	return m
}

// EnsureAddonsDir creates the Interface/AddOns directory if it doesn't exist
func (m *Manager) EnsureAddonsDir() error {
	if err := os.MkdirAll(m.addonsDir, 0755); err != nil {
		return fmt.Errorf("%w: %v", ErrAddonsDir, err)
	}
	return nil
}

// Load loads the addon store from disk
func (m *Manager) Load() error {
	return m.store.Load()
}

// Save saves the addon store to disk
func (m *Manager) Save() error {
	return m.store.Save()
}

// InstallResult contains information about a completed install
type InstallResult struct {
	Name  string
	Title string
	Path  string
}

// Install installs an addon from a git URL
// progressWriter can be nil to disable progress output
func (m *Manager) Install(gitURL string, progressWriter io.Writer) (*InstallResult, error) {
	// Validate URL
	if err := ValidateGitURL(gitURL); err != nil {
		return nil, ErrInvalidURL
	}

	gitURL = NormalizeGitURL(gitURL)

	// Extract addon name from URL
	addonName := ExtractRepoName(gitURL)

	// Check if addon already exists
	addonPath := filepath.Join(m.addonsDir, addonName)
	if _, err := os.Stat(addonPath); err == nil {
		return nil, fmt.Errorf("%w: %s", ErrAddonExists, addonName)
	}

	// Ensure addons directory exists
	if err := m.EnsureAddonsDir(); err != nil {
		return nil, err
	}

	// Clone the repository
	if err := CloneRepo(gitURL, addonPath, progressWriter); err != nil {
		_ = CleanupFailedClone(addonPath)
		return nil, err
	}

	// Check for .toc file and get correct addon name
	tocPath, tocName, err := FindTOCFile(addonPath)
	if err != nil {
		// No .toc file found - might be a multi-addon repo or invalid
		m.log.Warn("No .toc file found in repository", "path", addonPath)
	}

	// If .toc name differs from folder name, rename
	if tocName != "" && tocName != addonName {
		newPath := filepath.Join(m.addonsDir, tocName)
		if _, err := os.Stat(newPath); err == nil {
			// Target already exists, keep original name
			m.log.Warn("Target addon name already exists, keeping original",
				"original", addonName, "target", tocName)
		} else {
			if err := os.Rename(addonPath, newPath); err != nil {
				m.log.Warn("Failed to rename addon folder", "error", err)
			} else {
				addonPath = newPath
				addonName = tocName
				m.log.Debug("Renamed addon folder", "name", addonName)
			}
		}
	}

	// Parse .toc for metadata
	var tocInfo *TOCInfo
	if tocPath != "" {
		// Update tocPath if we renamed the folder
		tocPath = filepath.Join(addonPath, filepath.Base(tocPath))
		tocInfo, _ = ParseTOC(tocPath)
	}

	// Store metadata
	now := time.Now()
	meta := AddonMetadata{
		GitURL:      gitURL,
		InstalledAt: now,
		UpdatedAt:   now,
	}
	m.store.Set(addonName, meta)

	if err := m.store.Save(); err != nil {
		m.log.Warn("Failed to save addon metadata", "error", err)
	}

	result := &InstallResult{
		Name: addonName,
		Path: addonPath,
	}
	if tocInfo != nil && tocInfo.Title != "" {
		result.Title = tocInfo.Title
	} else {
		result.Title = addonName
	}

	m.log.Info("Addon installed", "name", addonName, "url", gitURL)
	return result, nil
}

// Remove removes an installed addon
func (m *Manager) Remove(name string, createBackup bool) error {
	addonPath := filepath.Join(m.addonsDir, name)

	// Check addon exists
	if _, err := os.Stat(addonPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrAddonNotFound, name)
	}

	// Create backup if requested
	if createBackup {
		backupPath, err := m.backup.CreateBackup(addonPath, name)
		if err != nil {
			m.log.Warn("Failed to create backup", "error", err)
		} else {
			m.log.Info("Backup created", "path", backupPath)
		}
	}

	// Remove the addon directory
	if err := os.RemoveAll(addonPath); err != nil {
		return fmt.Errorf("failed to remove addon: %w", err)
	}

	// Remove from store
	m.store.Delete(name)
	if err := m.store.Save(); err != nil {
		m.log.Warn("Failed to save store after removal", "error", err)
	}

	m.log.Info("Addon removed", "name", name)
	return nil
}

// UpdateResult contains information about an update operation
type UpdateResult struct {
	Updated         bool
	AlreadyUpToDate bool
	ReCloned        bool
}

// Update updates an addon using git fast-forward
// progressWriter can be nil to disable progress output
func (m *Manager) Update(name string, progressWriter io.Writer) (*UpdateResult, error) {
	addonPath := filepath.Join(m.addonsDir, name)
	result := &UpdateResult{}

	// Check addon exists
	if _, err := os.Stat(addonPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrAddonNotFound, name)
	}

	// Check it's a git repo
	if !IsGitRepo(addonPath) {
		// Try to get URL from store and re-clone
		meta, ok := m.store.Get(name)
		if !ok || meta.GitURL == "" {
			return nil, fmt.Errorf("addon is not a git repository and has no stored URL")
		}

		// Backup and re-clone
		if _, err := m.backup.CreateBackup(addonPath, name); err != nil {
			m.log.Warn("Failed to create backup before re-clone", "error", err)
		}

		if err := os.RemoveAll(addonPath); err != nil {
			return nil, fmt.Errorf("failed to remove for re-clone: %w", err)
		}

		if err := CloneRepo(meta.GitURL, addonPath, progressWriter); err != nil {
			return nil, err
		}

		meta.UpdatedAt = time.Now()
		m.store.Set(name, meta)
		_ = m.store.Save()

		result.Updated = true
		result.ReCloned = true
		return result, nil
	}

	// Perform git update
	err := UpdateRepo(addonPath, progressWriter)
	if errors.Is(err, ErrAlreadyUpToDate) {
		m.log.Debug("Addon already up to date", "name", name)
		result.AlreadyUpToDate = true
		return result, nil
	}
	if errors.Is(err, ErrFFNotPossible) {
		return nil, fmt.Errorf("cannot update %s: local modifications exist (backup and re-install to force)", name)
	}
	if err != nil {
		return nil, err
	}

	// Update metadata
	if meta, ok := m.store.Get(name); ok {
		meta.UpdatedAt = time.Now()
		m.store.Set(name, meta)
		_ = m.store.Save()
	}

	result.Updated = true
	m.log.Info("Addon updated", "name", name)
	return result, nil
}

// UpdateAllResult contains results from updating all addons
type UpdateAllResult struct {
	Updated int
	Failed  int
	Skipped int
	Errors  []string
}

// UpdateAll updates all tracked addons
func (m *Manager) UpdateAll() *UpdateAllResult {
	result := &UpdateAllResult{}
	addons := m.store.List()

	for _, name := range addons {
		updateResult, err := m.Update(name, nil)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}

		if updateResult.AlreadyUpToDate {
			result.Skipped++
		} else if updateResult.Updated {
			result.Updated++
		}
	}

	return result
}

// GetTrackedAddons returns the list of tracked addon names
func (m *Manager) GetTrackedAddons() []string {
	return m.store.List()
}

// CheckUpdatesResult contains information about available updates
type CheckUpdatesResult struct {
	Name      string
	HasUpdate bool
	Error     error
}

// CheckAllUpdates checks all tracked addons for available updates
func (m *Manager) CheckAllUpdates() []CheckUpdatesResult {
	var results []CheckUpdatesResult
	tracked := m.store.List()

	for _, name := range tracked {
		addonPath := filepath.Join(m.addonsDir, name)

		// Skip if not a git repo
		if !IsGitRepo(addonPath) {
			continue
		}

		hasUpdate, err := CheckForUpdates(addonPath)
		results = append(results, CheckUpdatesResult{
			Name:      name,
			HasUpdate: hasUpdate,
			Error:     err,
		})
	}

	return results
}

// GetInfo returns detailed information about an addon
func (m *Manager) GetInfo(name string) (*Addon, error) {
	addonPath := filepath.Join(m.addonsDir, name)

	// Check addon exists
	if _, err := os.Stat(addonPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrAddonNotFound, name)
	}

	addon := &Addon{
		Name: name,
		Path: addonPath,
	}

	// Get .toc info
	tocPath, _, err := FindTOCFile(addonPath)
	if err == nil {
		if tocInfo, err := ParseTOC(tocPath); err == nil {
			addon.Title = tocInfo.Title
			addon.Version = tocInfo.Version
			addon.Author = tocInfo.Author
			addon.Notes = tocInfo.Notes
		}
	}

	// Get stored metadata
	if meta, ok := m.store.Get(name); ok {
		addon.GitURL = meta.GitURL
		addon.InstalledAt = meta.InstalledAt
		addon.UpdatedAt = meta.UpdatedAt
	} else {
		// Try to get URL from git remote
		if url, err := GetRepoRemoteURL(addonPath); err == nil {
			addon.GitURL = url
		}
	}

	return addon, nil
}

// ListInstalled returns all installed addons
func (m *Manager) ListInstalled() ([]*Addon, error) {
	entries, err := os.ReadDir(m.addonsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Addon{}, nil
		}
		return nil, err
	}

	var addons []*Addon
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		addon, err := m.GetInfo(entry.Name())
		if err != nil {
			// Include addon even if we can't get full info
			addon = &Addon{
				Name: entry.Name(),
				Path: filepath.Join(m.addonsDir, entry.Name()),
			}
		}
		addons = append(addons, addon)
	}

	// Sort by status (default first, then tracked, then untracked), then by name
	sort.Slice(addons, func(i, j int) bool {
		// Get status priority: default=0, tracked=1, untracked=2
		getPriority := func(a *Addon) int {
			if IsDefaultAddon(a.Name) {
				return 0
			}
			if a.GitURL != "" {
				return 1
			}
			return 2
		}

		pi, pj := getPriority(addons[i]), getPriority(addons[j])
		if pi != pj {
			return pi < pj
		}
		return strings.ToLower(addons[i].Name) < strings.ToLower(addons[j].Name)
	})

	return addons, nil
}

// Repair scans and repairs addon metadata
// defaultAddons are addons that ship with Turtle WoW by default
// These should not be flagged as untracked or have issues reported
var defaultAddons = map[string]bool{
	"Blizzard_AuctionUI":          true,
	"Blizzard_BattlefieldMinimap": true,
	"Blizzard_BindingUI":          true,
	"Blizzard_CombatText":         true,
	"Blizzard_CraftUI":            true,
	"Blizzard_GMChatUI":           true,
	"Blizzard_GMSurveyUI":         true,
	"Blizzard_InspectUI":          true,
	"Blizzard_MacroUI":            true,
	"Blizzard_RaidUI":             true,
	"Blizzard_TalentUI":           true,
	"Blizzard_TradeSkillUI":       true,
	"Blizzard_TrainerUI":          true,
}

// IsDefaultAddon returns true if the addon is a default Turtle WoW addon
func IsDefaultAddon(name string) bool {
	return defaultAddons[name]
}

func (m *Manager) Repair() (*RepairResult, error) {
	result := &RepairResult{}

	// Get all folders in addons directory
	entries, err := os.ReadDir(m.addonsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, err
	}

	installedFolders := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			installedFolders[entry.Name()] = true
			result.TotalScanned++
		}
	}

	// Check for orphaned entries (in store but no folder)
	for _, name := range m.store.List() {
		if !installedFolders[name] {
			result.OrphanedEntries = append(result.OrphanedEntries, name)
			result.IssuesFound++
		}
	}

	// Check each installed folder
	storedAddons := m.store.All()
	for name := range installedFolders {
		addonPath := filepath.Join(m.addonsDir, name)

		// Skip default Turtle WoW addons - they don't need tracking
		if IsDefaultAddon(name) {
			continue
		}

		// Check if tracked
		if _, ok := storedAddons[name]; !ok {
			result.UntrackedAddons = append(result.UntrackedAddons, name)
			result.IssuesFound++

			// Try to auto-track if it's a git repo
			if url, err := GetRepoRemoteURL(addonPath); err == nil {
				m.store.Set(name, AddonMetadata{
					GitURL:      url,
					InstalledAt: time.Now(),
					UpdatedAt:   time.Now(),
				})
				m.log.Info("Auto-tracked addon from git remote", "name", name, "url", url)
			}
		}

		// Check git integrity if it's a git repo
		if IsGitRepo(addonPath) {
			if err := VerifyRepoIntegrity(addonPath); err != nil {
				result.CorruptedRepos = append(result.CorruptedRepos, name)
				result.IssuesFound++
			}
		}

		// Check .toc name matches folder name
		_, tocName, err := FindTOCFile(addonPath)
		if err == nil && tocName != name {
			result.NameMismatches = append(result.NameMismatches,
				fmt.Sprintf("%s (should be %s)", name, tocName))
			result.IssuesFound++
		}
	}

	// Remove orphaned entries
	for _, name := range result.OrphanedEntries {
		m.store.Delete(name)
		m.log.Info("Removed orphaned metadata entry", "name", name)
	}

	// Save store
	if err := m.store.Save(); err != nil {
		m.log.Warn("Failed to save store after repair", "error", err)
	}

	return result, nil
}

// GetAddonsDir returns the addons directory path
func (m *Manager) GetAddonsDir() string {
	return m.addonsDir
}

// GetBackupManager returns the backup manager
func (m *Manager) GetBackupManager() *BackupManager {
	return m.backup
}
