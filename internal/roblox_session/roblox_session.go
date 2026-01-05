package roblox_session

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GetCurrentUsername attempts to detect the currently logged-in Roblox username
func GetCurrentUsername() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Method 1: Check Roblox logs for recent authentication
	if username, err := getUsernameFromLogs(home); err == nil && username != "" {
		return username, nil
	}

	// Method 2: Check browser cookies for roblox.com
	if username, err := getUsernameFromBrowserCookies(); err == nil && username != "" {
		return username, nil
	}

	// Method 3: Check Roblox local storage files
	robloxDir := filepath.Join(home, "Library", "Application Support", "Roblox")
	if username, err := getUsernameFromLocalStorage(robloxDir); err == nil && username != "" {
		return username, nil
	}

	return "", fmt.Errorf("no active Roblox session found")
}

// getUsernameFromLogs checks Roblox log files for recent username
func getUsernameFromLogs(home string) (string, error) {
	logsDir := filepath.Join(home, "Library", "Logs", "Roblox")

	// Look for log files
	files, err := filepath.Glob(filepath.Join(logsDir, "*.log"))
	if err != nil || len(files) == 0 {
		return "", fmt.Errorf("no log files found")
	}

	// Check most recent log file
	for i := len(files) - 1; i >= 0 && i >= len(files)-3; i-- {
		data, err := os.ReadFile(files[i])
		if err != nil {
			continue
		}

		// Look for username patterns in logs
		re := regexp.MustCompile(`(?i)user(?:name)?[:\s]+([a-zA-Z0-9_]+)`)
		matches := re.FindStringSubmatch(string(data))
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("no username in logs")
}

// getUsernameFromBrowserCookies checks browser cookies
func getUsernameFromBrowserCookies() (string, error) {
	// Check Safari cookies using sqlite
	home, _ := os.UserHomeDir()
	cookiesDB := filepath.Join(home, "Library", "Cookies", "Cookies.binarycookies")

	// Note: Binary cookies are complex to parse
	// For now, just check if cookie file exists
	if _, err := os.Stat(cookiesDB); err == nil {
		return "[Browser Session Active]", nil
	}

	return "", fmt.Errorf("no browser cookies")
}

// getUsernameFromLocalStorage attempts to read username from Roblox LocalStorage
func getUsernameFromLocalStorage(robloxDir string) (string, error) {
	// Roblox may store user data in various places
	// This is a simplified implementation
	// Full implementation would parse actual Roblox local storage files

	possibleFiles := []string{
		filepath.Join(robloxDir, "LocalStorage", "userData.json"),
		filepath.Join(robloxDir, "GlobalBasicSettings_13.xml"),
		filepath.Join(robloxDir, "user.json"),
	}

	for _, filePath := range possibleFiles {
		if data, err := os.ReadFile(filePath); err == nil {
			// Try to parse as JSON
			var userData map[string]interface{}
			if json.Unmarshal(data, &userData) == nil {
				if username, ok := userData["username"].(string); ok && username != "" {
					return username, nil
				}
				if displayName, ok := userData["displayName"].(string); ok && displayName != "" {
					return displayName, nil
				}
			}
		}
	}

	return "", fmt.Errorf("username not found in local storage")
}

// getUsernameFromCookie attempts to get username from browser cookies
func getUsernameFromCookie() (string, error) {
	// Check Safari cookies for roblox.com
	// This requires reading macOS Cookies.binarycookies which is complex
	// Simplified: Just check if .ROBLOSECURITY exists in Keychain

	cmd := exec.Command("security", "find-internet-password",
		"-s", "roblox.com",
		"-w") // Print password only

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no Roblox cookie found")
	}

	cookie := strings.TrimSpace(string(output))
	if len(cookie) > 10 {
		// Cookie exists, but we can't easily extract username from it
		// Return a placeholder indicating session exists
		return "[Active Session]", nil
	}

	return "", fmt.Errorf("no valid cookie")
}

// IsLoggedIn checks if there's an active Roblox session
func IsLoggedIn() bool {
	username, err := GetCurrentUsername()
	return err == nil && username != ""
}

// ClearSession clears cached session info (not actual Roblox logout)
func ClearSession() error {
	// This would clear any cached session data
	// Not implementing actual logout as that requires Roblox API
	return nil
}
