package addons

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// StoreManager handles persistence of addon metadata
type StoreManager struct {
	path  string
	store *Store
	mu    sync.RWMutex
}

// NewStoreManager creates a new store manager
func NewStoreManager(dataDir string) *StoreManager {
	return &StoreManager{
		path: filepath.Join(dataDir, "addons.json"),
		store: &Store{
			Addons: make(map[string]AddonMetadata),
		},
	}
}

// Load reads the store from disk
func (sm *StoreManager) Load() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	data, err := os.ReadFile(sm.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize empty store
			sm.store = &Store{
				Addons: make(map[string]AddonMetadata),
			}
			return nil
		}
		return err
	}

	var store Store
	if err := json.Unmarshal(data, &store); err != nil {
		return err
	}

	if store.Addons == nil {
		store.Addons = make(map[string]AddonMetadata)
	}

	sm.store = &store
	return nil
}

// Save writes the store to disk
func (sm *StoreManager) Save() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(sm.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(sm.store, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sm.path, data, 0644)
}

// Get retrieves metadata for an addon
func (sm *StoreManager) Get(name string) (AddonMetadata, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	meta, ok := sm.store.Addons[name]
	return meta, ok
}

// Set stores metadata for an addon
func (sm *StoreManager) Set(name string, meta AddonMetadata) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.store.Addons[name] = meta
}

// Delete removes metadata for an addon
func (sm *StoreManager) Delete(name string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.store.Addons, name)
}

// List returns all addon names in the store
func (sm *StoreManager) List() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	names := make([]string, 0, len(sm.store.Addons))
	for name := range sm.store.Addons {
		names = append(names, name)
	}
	return names
}

// All returns a copy of all addon metadata
func (sm *StoreManager) All() map[string]AddonMetadata {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]AddonMetadata, len(sm.store.Addons))
	for k, v := range sm.store.Addons {
		result[k] = v
	}
	return result
}
