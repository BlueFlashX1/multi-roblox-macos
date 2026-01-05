package preset_manager

import (
	"encoding/json"
	"fmt"
	"insadem/multi_roblox_macos/internal/open_app"
	"insadem/multi_roblox_macos/internal/roblox_api"
	"os"
	"os/exec"
	"path/filepath"
)

// Preset represents a saved Roblox game shortcut
type Preset struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	PlaceID      int64  `json:"place_id,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	LastAccountUsed string `json:"last_account_used,omitempty"`
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

// AddPreset adds a new preset with auto-fetched game info
func AddPreset(name, url string) error {
	presets, err := LoadPresets()
	if err != nil {
		return err
	}

	preset := Preset{Name: name, URL: url}

	// Try to auto-fetch game info
	if placeID, err := roblox_api.ExtractPlaceID(url); err == nil {
		preset.PlaceID = placeID

		// Fetch game info
		if gameInfo, err := roblox_api.GetGameInfo(placeID); err == nil {
			// Use fetched name if user didn't provide one
			if name == "" {
				preset.Name = gameInfo.Name
			}
			preset.ThumbnailURL = gameInfo.ThumbnailURL
		}
	}

	presets = append(presets, preset)
	return SavePresets(presets)
}

// UpdatePresetLastAccount updates the last used account for a preset
func UpdatePresetLastAccount(index int, accountID string) error {
	presets, err := LoadPresets()
	if err != nil {
		return err
	}

	if index < 0 || index >= len(presets) {
		return fmt.Errorf("invalid preset index")
	}

	presets[index].LastAccountUsed = accountID
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
