package label_manager

import (
	"encoding/json"
	"insadem/multi_roblox_macos/internal/atomicfile"
	"os"
	"path/filepath"
	"sync"
)

// mu serializes every load→modify→save cycle on labels.json. Without it the
// 2s updateInstances ticker (CleanupStaleLabels) races UI callbacks (SetLabel)
// and the last writer silently discards the other's change. Same pattern as
// instance_account_tracker.
var mu sync.Mutex

// InstanceLabel represents a label for a Roblox instance
type InstanceLabel struct {
	PID   int    `json:"pid"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// Config stores all instance labels
type Config struct {
	Labels []InstanceLabel `json:"labels"`
}

// GetConfigPath returns the path to the labels config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(homeDir, "Library", "Application Support", "multi_roblox_macos")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(configDir, "labels.json"), nil
}

// loadLabelsLocked reads labels from disk. Caller MUST hold mu.
func loadLabelsLocked() ([]InstanceLabel, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty list
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return []InstanceLabel{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config.Labels, nil
}

// saveLabelsLocked persists labels to disk. Caller MUST hold mu.
func saveLabelsLocked(labels []InstanceLabel) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	config := Config{Labels: labels}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write — crash-safe, see internal/atomicfile.
	return atomicfile.WriteFile(configPath, data, 0600) // Secure permissions - owner only
}

// LoadLabels loads instance labels from config file (public API, acquires lock).
func LoadLabels() ([]InstanceLabel, error) {
	mu.Lock()
	defer mu.Unlock()
	return loadLabelsLocked()
}

// SaveLabels saves instance labels to config file (public API, acquires lock).
func SaveLabels(labels []InstanceLabel) error {
	mu.Lock()
	defer mu.Unlock()
	return saveLabelsLocked(labels)
}

// GetLabel returns the label for a specific PID
func GetLabel(pid int) (InstanceLabel, bool) {
	mu.Lock()
	defer mu.Unlock()

	labels, err := loadLabelsLocked()
	if err != nil {
		return InstanceLabel{}, false
	}

	for _, label := range labels {
		if label.PID == pid {
			return label, true
		}
	}

	return InstanceLabel{}, false
}

// SetLabel sets or updates a label for a PID
func SetLabel(pid int, labelText, color string) error {
	mu.Lock()
	defer mu.Unlock()

	labels, err := loadLabelsLocked()
	if err != nil {
		return err
	}

	// Update existing or add new
	found := false
	for i := range labels {
		if labels[i].PID == pid {
			labels[i].Label = labelText
			labels[i].Color = color
			found = true
			break
		}
	}

	if !found {
		labels = append(labels, InstanceLabel{
			PID:   pid,
			Label: labelText,
			Color: color,
		})
	}

	return saveLabelsLocked(labels)
}

// DeleteLabel removes a label for a PID
func DeleteLabel(pid int) error {
	mu.Lock()
	defer mu.Unlock()

	labels, err := loadLabelsLocked()
	if err != nil {
		return err
	}

	newLabels := []InstanceLabel{}
	for _, label := range labels {
		if label.PID != pid {
			newLabels = append(newLabels, label)
		}
	}

	return saveLabelsLocked(newLabels)
}

// CleanupStaleLabels removes labels for PIDs that no longer exist
func CleanupStaleLabels(activePIDs []int) error {
	mu.Lock()
	defer mu.Unlock()

	labels, err := loadLabelsLocked()
	if err != nil {
		return err
	}

	pidMap := make(map[int]bool)
	for _, pid := range activePIDs {
		pidMap[pid] = true
	}

	newLabels := []InstanceLabel{}
	for _, label := range labels {
		if pidMap[label.PID] {
			newLabels = append(newLabels, label)
		}
	}

	return saveLabelsLocked(newLabels)
}

// DefaultColors returns a list of default label colors
func DefaultColors() []string {
	return []string{
		"#FF6B6B", // Red
		"#4ECDC4", // Cyan
		"#45B7D1", // Blue
		"#FFA07A", // Orange
		"#98D8C8", // Mint
		"#F7DC6F", // Yellow
		"#BB8FCE", // Purple
		"#85C1E2", // Light Blue
	}
}

// DefaultLabels returns common label suggestions
func DefaultLabels() []string {
	return []string{
		"Main Account",
		"Alt 1",
		"Alt 2",
		"Trading",
		"AFK Farming",
		"Testing",
	}
}
