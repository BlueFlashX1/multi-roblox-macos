package instance_account_tracker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// InstanceAccountMap tracks which account is used by each instance
type InstanceAccountMap struct {
	PID        int       `json:"pid"`
	AccountID  string    `json:"account_id"`
	LaunchedAt time.Time `json:"launched_at"`
}

var (
	mu      sync.Mutex
	mapping []InstanceAccountMap
)

// GetMappingPath returns the path to the instance-account mapping file
func GetMappingPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "multi_roblox_macos", "instance_accounts.json")
}

// LoadMappings loads instance-account mappings from disk
func LoadMappings() ([]InstanceAccountMap, error) {
	mu.Lock()
	defer mu.Unlock()

	path := GetMappingPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []InstanceAccountMap{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var maps []InstanceAccountMap
	if err := json.Unmarshal(data, &maps); err != nil {
		return nil, err
	}

	mapping = maps
	return maps, nil
}

// SaveMappings saves mappings to disk
func SaveMappings(maps []InstanceAccountMap) error {
	mu.Lock()
	defer mu.Unlock()

	path := GetMappingPath()
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(maps, "", "  ")
	if err != nil {
		return err
	}

	mapping = maps
	return os.WriteFile(path, data, 0600)
}

// TrackInstance records which account was used to launch an instance
func TrackInstance(pid int, accountID string) error {
	maps, err := LoadMappings()
	if err != nil {
		return err
	}

	// Remove old mapping for this PID if exists
	filtered := []InstanceAccountMap{}
	for _, m := range maps {
		if m.PID != pid {
			filtered = append(filtered, m)
		}
	}

	// Add new mapping
	filtered = append(filtered, InstanceAccountMap{
		PID:        pid,
		AccountID:  accountID,
		LaunchedAt: time.Now(),
	})

	return SaveMappings(filtered)
}

// GetAccountForInstance returns the account ID for a given PID
func GetAccountForInstance(pid int) (string, bool) {
	maps, err := LoadMappings()
	if err != nil {
		return "", false
	}

	for _, m := range maps {
		if m.PID == pid {
			return m.AccountID, true
		}
	}

	return "", false
}

// CleanupStaleInstances removes mappings for PIDs that no longer exist
func CleanupStaleInstances(activePIDs []int) error {
	maps, err := LoadMappings()
	if err != nil {
		return err
	}

	// Create map for fast lookup
	pidMap := make(map[int]bool)
	for _, pid := range activePIDs {
		pidMap[pid] = true
	}

	// Keep only active instances
	filtered := []InstanceAccountMap{}
	for _, m := range maps {
		if pidMap[m.PID] {
			filtered = append(filtered, m)
		}
	}

	return SaveMappings(filtered)
}

// UntrackInstance removes tracking for a specific PID
func UntrackInstance(pid int) error {
	maps, err := LoadMappings()
	if err != nil {
		return err
	}

	filtered := []InstanceAccountMap{}
	for _, m := range maps {
		if m.PID != pid {
			filtered = append(filtered, m)
		}
	}

	return SaveMappings(filtered)
}
