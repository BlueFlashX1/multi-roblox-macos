package preset_manager

import (
	"encoding/json"
	"fmt"
	"insadem/multi_roblox_macos/internal/logger"
	"insadem/multi_roblox_macos/internal/roblox_api"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// timeNowMillis returns current time in milliseconds
func timeNowMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// Preset represents a saved Roblox game shortcut
type Preset struct {
	Name                  string `json:"name"`
	URL                   string `json:"url"`
	PlaceID               int64  `json:"place_id,omitempty"`
	ThumbnailURL          string `json:"thumbnail_url,omitempty"`
	LastAccountUsed       string `json:"last_account_used,omitempty"`
	PrivateServerLinkCode string `json:"private_server_link_code,omitempty"`
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

	return os.WriteFile(configPath, data, 0600) // Secure permissions - owner only
}

// AddPreset adds a new preset with auto-fetched game info
func AddPreset(name, url string) error {
	presets, err := LoadPresets()
	if err != nil {
		return err
	}

	preset := Preset{Name: name, URL: url}

	// Try to extract private server link code if present
	if linkCode := ExtractPrivateServerLinkCode(url); linkCode != "" {
		preset.PrivateServerLinkCode = linkCode
		logger.LogInfo("Detected private server link code in URL")
	}

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

// ExtractPrivateServerLinkCode extracts the link code from a private server URL
// Supports multiple formats:
// - https://www.roblox.com/games/123456?privateServerLinkCode=XXXXX
// - https://www.roblox.com/share?code=XXXXX&type=Server
// - https://ro.blox.com/Ebh5?pid=share&is_retargeting=true&af_dp=...&code=XXXXX
// - Direct code paste: XXXXX
func ExtractPrivateServerLinkCode(input string) string {
	input = strings.TrimSpace(input)
	logger.LogDebug("Extracting private server link code from: %s", input)

	// If it looks like just a code (no URL characters), return it directly
	if !strings.Contains(input, "/") && !strings.Contains(input, "?") && !strings.Contains(input, "=") {
		if len(input) > 10 && len(input) < 50 {
			logger.LogDebug("Input looks like a direct code: %s", input)
			return input
		}
	}

	// Look for privateServerLinkCode parameter
	if strings.Contains(input, "privateServerLinkCode=") {
		parts := strings.Split(input, "privateServerLinkCode=")
		if len(parts) > 1 {
			code := parts[1]
			if idx := strings.Index(code, "&"); idx != -1 {
				code = code[:idx]
			}
			code = strings.TrimSpace(code)
			logger.LogDebug("Found privateServerLinkCode: %s", code)
			return code
		}
	}

	// Check for linkCode parameter (shorter form)
	if strings.Contains(input, "linkCode=") {
		parts := strings.Split(input, "linkCode=")
		if len(parts) > 1 {
			code := parts[1]
			if idx := strings.Index(code, "&"); idx != -1 {
				code = code[:idx]
			}
			code = strings.TrimSpace(code)
			logger.LogDebug("Found linkCode: %s", code)
			return code
		}
	}

	// Check for code= parameter (used in share links)
	if strings.Contains(input, "code=") {
		parts := strings.Split(input, "code=")
		if len(parts) > 1 {
			code := parts[1]
			if idx := strings.Index(code, "&"); idx != -1 {
				code = code[:idx]
			}
			code = strings.TrimSpace(code)
			logger.LogDebug("Found code: %s", code)
			return code
		}
	}

	logger.LogDebug("No link code found in input")
	return ""
}

// UpdatePresetPrivateServer updates the private server link code for a preset
func UpdatePresetPrivateServer(index int, linkCode string) error {
	presets, err := LoadPresets()
	if err != nil {
		return err
	}

	if index < 0 || index >= len(presets) {
		return fmt.Errorf("invalid preset index")
	}

	presets[index].PrivateServerLinkCode = linkCode
	logger.LogInfo("Updated preset %d private server link code", index)
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
	_, err := LaunchPresetWithTicket(preset, "")
	return err
}

// isRobloxRunning checks if any Roblox player instances are currently running
func isRobloxRunning() bool {
	cmd := exec.Command("pgrep", "-x", "RobloxPlayer")
	err := cmd.Run()
	return err == nil
}

// getNextRobloxCopyPath returns the next available path for a Roblox copy
func getNextRobloxCopyPath() string {
	for i := 2; i <= 10; i++ {
		path := fmt.Sprintf("/tmp/Roblox%d.app", i)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
	}
	// Cleanup old ones and use Roblox2.app
	for i := 2; i <= 10; i++ {
		path := fmt.Sprintf("/tmp/Roblox%d.app", i)
		os.RemoveAll(path)
	}
	return "/tmp/Roblox2.app"
}

// copyRobloxApp copies Roblox.app to a temporary location for multi-instance
func copyRobloxApp(destPath string) error {
	logger.LogInfo("Copying Roblox.app to %s for multi-instance...", destPath)

	// Use cp -R to copy the entire bundle
	cmd := exec.Command("cp", "-R", "/Applications/Roblox.app", destPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.LogError("Failed to copy Roblox.app: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to copy Roblox.app: %w", err)
	}

	logger.LogInfo("Successfully copied Roblox.app to %s", destPath)
	return nil
}

// LaunchPresetWithTicket launches a preset with an optional authentication ticket
// Returns the PID of the launched process (or 0 if unknown)
func LaunchPresetWithTicket(preset Preset, authTicket string) (int, error) {
	logger.LogInfo("LaunchPresetWithTicket called for: %s", preset.Name)
	logger.LogDebug("Preset URL: %s", preset.URL)
	logger.LogDebug("Preset PlaceID: %d", preset.PlaceID)
	if authTicket != "" {
		logger.LogDebug("Using auth ticket (length: %d)", len(authTicket))
	}

	// Determine the place ID
	var placeID int64
	if preset.PlaceID > 0 {
		placeID = preset.PlaceID
	} else if !strings.HasPrefix(preset.URL, "roblox://") {
		if extractedID, err := roblox_api.ExtractPlaceID(preset.URL); err == nil && extractedID > 0 {
			placeID = extractedID
		}
	}

	// Build the launch URL/protocol string
	var protocolString string
	if placeID > 0 && authTicket != "" {
		// Use the proper roblox-player: format with auth ticket (how browsers do it!)
		launchTime := fmt.Sprintf("%d", timeNowMillis())
		browserTrackerId := fmt.Sprintf("%d", timeNowMillis()%1000000000)

		// Check if launching to private server
		if preset.PrivateServerLinkCode != "" {
			linkCode := preset.PrivateServerLinkCode
			logger.LogInfo("Attempting private server launch with code: %s...", linkCode[:min(10, len(linkCode))])

			// Use the share link resolution approach - PlaceLauncher with share code
			// Try request=RequestPrivateGame with the share code as linkCode
			protocolString = fmt.Sprintf("roblox-player:1+launchmode:play+gameinfo:%s+launchtime:%s+placelauncherurl:https://assetgame.roblox.com/game/PlaceLauncher.ashx?request=RequestPrivateGame&placeId=%d&linkCode=%s&browserTrackerId=%s+browsertrackerid:%s+robloxLocale:en_us+gameLocale:en_us+channel:",
				authTicket, launchTime, placeID, linkCode, browserTrackerId, browserTrackerId)
			logger.LogInfo("Using RequestPrivateGame with linkCode")
		} else {
			// Regular game launch
			protocolString = fmt.Sprintf("roblox-player:1+launchmode:play+gameinfo:%s+launchtime:%s+placelauncherurl:https://assetgame.roblox.com/game/PlaceLauncher.ashx?request=RequestGame&browserTrackerId=%s&placeId=%d&isPlayTogetherGame=false+browsertrackerid:%s+robloxLocale:en_us+gameLocale:en_us+channel:",
				authTicket, launchTime, browserTrackerId, placeID, browserTrackerId)
			logger.LogDebug("Using roblox-player: with auth ticket")
		}
	} else if placeID > 0 {
		protocolString = fmt.Sprintf("roblox://placeId=%d", placeID)
	} else if strings.HasPrefix(preset.URL, "roblox://") {
		protocolString = preset.URL
	} else {
		protocolString = preset.URL
		logger.LogDebug("Using original URL (may not launch game): %s", protocolString)
	}

	if len(protocolString) > 100 {
		logger.LogDebug("Final protocol string: %s...", protocolString[:100])
	} else {
		logger.LogDebug("Final protocol string: %s", protocolString)
	}

	// Check if Roblox is already running - need to use copied app for multi-instance
	robloxApp := "/Applications/Roblox.app/Contents/MacOS/RobloxPlayer"
	if isRobloxRunning() && authTicket != "" {
		// For multi-instance with auth ticket, we need to copy the app
		copyPath := getNextRobloxCopyPath()
		if err := copyRobloxApp(copyPath); err != nil {
			// Fall back to regular launch
			logger.LogError("Failed to copy Roblox for multi-instance, falling back to regular launch: %v", err)
		} else {
			robloxApp = filepath.Join(copyPath, "Contents", "MacOS", "RobloxPlayer")
			logger.LogInfo("Using copied app for multi-instance: %s", robloxApp)
		}
	}

	// Launch directly with -protocolString for auth ticket launches
	if authTicket != "" {
		logger.LogDebug("Launching with direct -protocolString: %s", robloxApp)
		cmd := exec.Command(robloxApp, "-protocolString", protocolString)
		err := cmd.Start() // Use Start() to not wait
		if err != nil {
			logger.LogError("LaunchPreset failed: %v", err)
			return 0, err
		}
		pid := cmd.Process.Pid
		logger.LogInfo("LaunchPreset successful (direct launch), PID: %d", pid)
		return pid, nil
	}

	// For non-auth launches, use open command
	cmdArgs := []string{"-n", protocolString}
	logger.LogDebug("Command: open -n %v", cmdArgs)

	cmd := exec.Command("open", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.LogError("LaunchPreset failed: %v, output: %s", err, string(output))
		return 0, err
	}

	logger.LogInfo("LaunchPreset successful, output: %s", string(output))
	// For open command, we can't easily get the PID
	return 0, nil
}

// LaunchRobloxHomeWithAccount launches Roblox home screen with a specific account
// using multi-instance support (copies app if Roblox is already running)
// Returns the PID of the launched process
func LaunchRobloxHomeWithAccount(cookie string) (int, error) {
	logger.LogInfo("LaunchRobloxHomeWithAccount called")

	// Determine which Roblox app to use
	robloxApp := "/Applications/Roblox.app/Contents/MacOS/RobloxPlayer"

	// Check if Roblox is already running - need to use copied app for multi-instance
	if isRobloxRunning() {
		copyPath := getNextRobloxCopyPath()
		if err := copyRobloxApp(copyPath); err != nil {
			logger.LogError("Failed to copy Roblox for multi-instance: %v", err)
			// Fall through to use main app anyway
		} else {
			robloxApp = filepath.Join(copyPath, "Contents", "MacOS", "RobloxPlayer")
			logger.LogInfo("Using copied app for multi-instance: %s", robloxApp)
		}
	}

	// Launch Roblox directly without a game URL - it will open to home screen
	// The app will use whatever credentials are stored
	logger.LogInfo("Launching Roblox home: %s", robloxApp)
	cmd := exec.Command(robloxApp)
	err := cmd.Start()
	if err != nil {
		logger.LogError("Failed to launch Roblox home: %v", err)
		return 0, err
	}

	pid := cmd.Process.Pid
	logger.LogInfo("Roblox home launched successfully, PID: %d", pid)
	return pid, nil
}
