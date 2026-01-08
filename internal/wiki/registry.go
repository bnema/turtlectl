package wiki

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// Registry fetches and caches the addon registry from GitHub
type Registry struct {
	cacheDir  string
	cachePath string
	etagPath  string
	logger    *log.Logger
	client    *http.Client
}

// NewRegistry creates a new registry manager
func NewRegistry(cacheDir string, logger *log.Logger) *Registry {
	return &Registry{
		cacheDir:  cacheDir,
		cachePath: filepath.Join(cacheDir, "addons-registry.json"),
		etagPath:  filepath.Join(cacheDir, "addons-registry.etag"),
		logger:    logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAddons returns the addon list, fetching from GitHub if needed
// forceRefresh bypasses the cache TTL check
func (r *Registry) GetAddons(forceRefresh bool) ([]WikiAddon, error) {
	// Try to load from cache first
	cached, cacheTime, err := r.loadCache()
	if err == nil && cached != nil {
		cacheAge := time.Since(cacheTime)

		// If cache is fresh and not forcing refresh, use it
		if !forceRefresh && cacheAge < RegistryCacheTTL {
			r.logger.Debug("Using cached registry", "age", cacheAge.Round(time.Minute))
			return cached.Addons, nil
		}

		r.logger.Debug("Cache is stale", "age", cacheAge.Round(time.Hour))
	}

	// Try to fetch from GitHub
	fresh, err := r.fetchFromGitHub()
	if err != nil {
		// Network failed - use stale cache if available
		if cached != nil {
			r.logger.Warn("Failed to fetch registry, using stale cache",
				"error", err,
				"cache_age", time.Since(cacheTime).Round(time.Hour))
			return cached.Addons, nil
		}
		return nil, fmt.Errorf("failed to fetch registry and no cache available: %w", err)
	}

	// If fetch returned nil, it means 304 Not Modified - cache is still valid
	if fresh == nil {
		if cached != nil {
			// Update cache timestamp
			_ = r.touchCache()
			return cached.Addons, nil
		}
		return nil, fmt.Errorf("registry returned not-modified but no cache exists")
	}

	// Save fresh data to cache
	if err := r.saveCache(fresh); err != nil {
		r.logger.Warn("Failed to save cache", "error", err)
	}

	return fresh.Addons, nil
}

// fetchFromGitHub fetches the registry from GitHub raw URL
// Returns nil if 304 Not Modified (cache is still valid)
func (r *Registry) fetchFromGitHub() (*RegistryData, error) {
	req, err := http.NewRequest("GET", RegistryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "turtlectl/1.0 (Turtle WoW addon manager)")
	req.Header.Set("Accept", "application/json")

	// Add ETag for conditional request
	if etag, err := r.loadETag(); err == nil && etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	r.logger.Debug("Fetching registry from GitHub", "url", RegistryURL)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		r.logger.Debug("Registry not modified (304)")
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON
	var registry RegistryData
	if err := json.Unmarshal(body, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	// Validate version
	if registry.Version != RegistryVersion {
		r.logger.Warn("Registry version mismatch",
			"expected", RegistryVersion, "got", registry.Version)
	}

	// Save ETag for future requests
	if etag := resp.Header.Get("ETag"); etag != "" {
		_ = r.saveETag(etag)
	}

	r.logger.Info("Fetched registry from GitHub",
		"addons", len(registry.Addons),
		"generated_at", registry.GeneratedAt.Format("2006-01-02"))

	return &registry, nil
}

// loadCache loads the cached registry from disk
func (r *Registry) loadCache() (*RegistryData, time.Time, error) {
	info, err := os.Stat(r.cachePath)
	if err != nil {
		return nil, time.Time{}, err
	}

	data, err := os.ReadFile(r.cachePath)
	if err != nil {
		return nil, time.Time{}, err
	}

	var registry RegistryData
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, time.Time{}, err
	}

	return &registry, info.ModTime(), nil
}

// saveCache saves the registry to disk
func (r *Registry) saveCache(registry *RegistryData) error {
	// Ensure directory exists
	if err := os.MkdirAll(r.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(r.cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	return nil
}

// touchCache updates the cache file's modification time
func (r *Registry) touchCache() error {
	now := time.Now()
	return os.Chtimes(r.cachePath, now, now)
}

// loadETag loads the cached ETag
func (r *Registry) loadETag() (string, error) {
	data, err := os.ReadFile(r.etagPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// saveETag saves the ETag to disk
func (r *Registry) saveETag(etag string) error {
	if err := os.MkdirAll(r.cacheDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(r.etagPath, []byte(etag), 0644)
}

// RegistryInfo contains information about the registry cache state
type RegistryInfo struct {
	HasCache    bool
	IsStale     bool
	LastUpdated time.Time // When local cache was last updated
	GeneratedAt time.Time // When registry was generated (from registry metadata)
	Age         time.Duration
	TotalAddons int
	NewAddons   int
}

// GetInfo returns information about the registry cache state
func (r *Registry) GetInfo() RegistryInfo {
	cached, cacheTime, err := r.loadCache()
	if err != nil || cached == nil {
		return RegistryInfo{
			HasCache: false,
		}
	}

	cacheAge := time.Since(cacheTime)
	newCount := 0
	for _, addon := range cached.Addons {
		if addon.IsNew() {
			newCount++
		}
	}

	return RegistryInfo{
		HasCache:    true,
		IsStale:     cacheAge > RegistryCacheTTL,
		LastUpdated: cacheTime,
		Age:         cacheAge,
		TotalAddons: len(cached.Addons),
		NewAddons:   newCount,
		GeneratedAt: cached.GeneratedAt,
	}
}

// MarkInstalled marks addons that are already installed
func MarkInstalled(addons []WikiAddon, installedURLs map[string]bool) {
	for i := range addons {
		url := addons[i].URL
		// Check both with and without .git suffix
		addons[i].IsInstalled = installedURLs[url] ||
			installedURLs[url+".git"] ||
			installedURLs[trimGitSuffix(url)]
	}
}

// trimGitSuffix removes .git suffix if present
func trimGitSuffix(url string) string {
	if len(url) > 4 && url[len(url)-4:] == ".git" {
		return url[:len(url)-4]
	}
	return url
}

// SortAddons sorts addons alphabetically by name
func SortAddons(addons []WikiAddon) {
	sort.Slice(addons, func(i, j int) bool {
		return addons[i].Name < addons[j].Name
	})
}
