package preset_manager

import (
	"encoding/json"
	"fmt"
	"insadem/multi_roblox_macos/internal/open_app"
	"os"
	"os/exec"
	"path/filepath"
)

// Preset represents a saved Roblox game shortcut
type Preset struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Config stores all presets
type Config struct {
	Presets []Preset `json:"presets"`
}

// GetConfigPath returns the path to the presets config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(homeDir, "Library", "Application Support", "multi_roblox_macos")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(configDir, "presets.json"), nil
}

// LoadPresets loads presets from config file
func LoadPresets() ([]Preset, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty list
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return []Preset{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config.Presets, nil
}

// SavePresets saves presets to config file
func SavePresets(presets []Preset) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	config := Config{Presets: presets}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// AddPreset adds a new preset
func AddPreset(name, url string) error {
	presets, err := LoadPresets()
	if err != nil {
		return err
	}

	presets = append(presets, Preset{Name: name, URL: url})
	return SavePresets(presets)
}

// DeletePreset removes a preset by index
func DeletePreset(index int) error {
	presets, err := LoadPresets()
	if err != nil {
		return err
	}

	if index < 0 || index >= len(presets) {
		return fmt.Errorf("invalid preset index")
	}

	presets = append(presets[:index], presets[index+1:]...)
	return SavePresets(presets)
}

// LaunchPreset launches Roblox with the URL from a preset
func LaunchPreset(preset Preset) error {
	// First open Roblox
	if err := open_app.Open("/Applications/Roblox.app"); err != nil {
		return err
	}

	// Then open the game URL
	cmd := exec.Command("open", preset.URL)
	return cmd.Run()
}
