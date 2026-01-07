package wiki

import "time"

// WikiAddon represents an addon discovered from the Turtle WoW wiki
type WikiAddon struct {
	Name        string `json:"name"`                  // Repository/addon name
	URL         string `json:"url"`                   // GitHub/GitLab URL
	Description string `json:"description,omitempty"` // From GitHub API or repo
	Author      string `json:"author,omitempty"`      // From GitHub API or repo
	Version     string `json:"version,omitempty"`     // From latest release/tag
	Stars       int    `json:"stars,omitempty"`       // GitHub stars count
	Category    string `json:"category,omitempty"`    // Letter section (A-Z) from wiki

	// LastCommit is when the repository was last updated (pushed_at from GitHub)
	// Used to determine if addon is still maintained
	LastCommit time.Time `json:"last_commit,omitempty"`

	// AddedAt is when this addon was first discovered in the registry
	// Used for "new" detection (addons added within NewAddonThreshold are marked new)
	AddedAt time.Time `json:"added_at,omitempty"`

	// Runtime state (not persisted in registry)
	IsInstalled bool `json:"-"`
}

// IsNew returns true if the addon was added to the registry recently
func (a *WikiAddon) IsNew() bool {
	if a.AddedAt.IsZero() {
		return false
	}
	return time.Since(a.AddedAt) < NewAddonThreshold
}

// RegistryData is the structure of the addon registry (data/addons.json)
type RegistryData struct {
	Version     int         `json:"version"`      // Schema version (bump when format changes)
	Revision    int         `json:"revision"`     // Update counter (increments each regeneration)
	GeneratedAt time.Time   `json:"generated_at"` // When this revision was generated
	SourceURL   string      `json:"source_url"`
	AddonCount  int         `json:"addon_count"`
	Addons      []WikiAddon `json:"addons"`
}

// Constants
const (
	// RegistryVersion is incremented when registry format changes
	RegistryVersion = 1

	// RegistryCacheTTL is how long before local cache is considered stale
	RegistryCacheTTL = 24 * time.Hour

	// NewAddonThreshold is how long an addon is considered "new"
	NewAddonThreshold = 7 * 24 * time.Hour

	// RegistryURL is the URL to fetch the addon registry from GitHub
	RegistryURL = "https://raw.githubusercontent.com/bnema/turtlectl/main/data/addons.json"

	// WikiURL is the Turtle WoW addon wiki page (for reference)
	WikiURL = "https://turtle-wow.fandom.com/wiki/Addons"
)

// NewRegistryData creates a new empty registry
func NewRegistryData() *RegistryData {
	return &RegistryData{
		Version:     RegistryVersion,
		GeneratedAt: time.Now(),
		SourceURL:   WikiURL,
		Addons:      []WikiAddon{},
	}
}
