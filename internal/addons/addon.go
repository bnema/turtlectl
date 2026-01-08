package addons

import "time"

// Addon represents an installed WoW addon
type Addon struct {
	Name        string    `json:"name"`         // Folder name (e.g., "pfQuest")
	Title       string    `json:"title"`        // From .toc: ## Title
	Version     string    `json:"version"`      // From .toc: ## Version
	Author      string    `json:"author"`       // From .toc: ## Author
	Notes       string    `json:"notes"`        // From .toc: ## Notes
	GitURL      string    `json:"git_url"`      // Source repository URL
	Path        string    `json:"path"`         // Full path to addon folder
	InstalledAt time.Time `json:"installed_at"` // When the addon was installed
	UpdatedAt   time.Time `json:"updated_at"`   // When the addon was last updated
}

// AddonMetadata is stored in addons.json for tracking
type AddonMetadata struct {
	GitURL      string    `json:"git_url"`
	InstalledAt time.Time `json:"installed_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Store represents the persistent addon metadata storage
type Store struct {
	Addons map[string]AddonMetadata `json:"addons"`
}

// RepairResult represents the outcome of a repair scan
type RepairResult struct {
	OrphanedEntries []string // In metadata but folder missing
	UntrackedAddons []string // Folder exists but no metadata
	CorruptedRepos  []string // Git repo is corrupted
	NameMismatches  []string // Folder name doesn't match .toc
	TotalScanned    int
	IssuesFound     int
}
