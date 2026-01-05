package label_manager

import (
	"encoding/json"
	"os"
	"path/filepath"
)

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

// LoadLabels loads instance labels from config file
func LoadLabels() ([]InstanceLabel, error) {
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

// SaveLabels saves instance labels to config file
func SaveLabels(labels []InstanceLabel) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	config := Config{Labels: labels}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetLabel returns the label for a specific PID
func GetLabel(pid int) (InstanceLabel, bool) {
	labels, err := LoadLabels()
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
	labels, err := LoadLabels()
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

	return SaveLabels(labels)
}

// DeleteLabel removes a label for a PID
func DeleteLabel(pid int) error {
	labels, err := LoadLabels()
	if err != nil {
		return err
	}

	newLabels := []InstanceLabel{}
	for _, label := range labels {
		if label.PID != pid {
			newLabels = append(newLabels, label)
		}
	}

	return SaveLabels(newLabels)
}

// CleanupStaleLabels removes labels for PIDs that no longer exist
func CleanupStaleLabels(activePIDs []int) error {
	labels, err := LoadLabels()
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

	return SaveLabels(newLabels)
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
