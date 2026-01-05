package roblox_session

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetCurrentUsername attempts to detect the currently logged-in Roblox username
func GetCurrentUsername() (string, error) {
	// Roblox on macOS stores data in ~/Library/Application Support/Roblox
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	robloxDir := filepath.Join(home, "Library", "Application Support", "Roblox")

	// Try to find username from Roblox local storage or settings
	// Method 1: Check LocalStorage (if accessible)
	if username, err := getUsernameFromLocalStorage(robloxDir); err == nil && username != "" {
		return username, nil
	}

	// Method 2: Check for .ROBLOSECURITY cookie (macOS Keychain)
	if username, err := getUsernameFromCookie(); err == nil && username != "" {
		return username, nil
	}

	return "", fmt.Errorf("no active Roblox session found")
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
