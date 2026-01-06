package roblox_login

import (
	"fmt"
	"insadem/multi_roblox_macos/internal/logger"
	"insadem/multi_roblox_macos/internal/open_app"
	"os/exec"
	"time"
)

// AccountSwitchNeeded indicates if we need to switch accounts before launching
var AccountSwitchNeeded = false
var PendingUsername = ""

// LaunchWithAccount launches Roblox and attempts to auto-login
// Note: This is a simplified version. Full automation would require
// AppleScript/UI automation which is complex and fragile.
func LaunchWithAccount(username, password string) error {
	logger.LogInfo("LaunchWithAccount called for user: %s", username)

	// Launch Roblox
	if err := open_app.Open("/Applications/Roblox.app"); err != nil {
		logger.LogError("Failed to launch Roblox for user %s: %v", username, err)
		return fmt.Errorf("failed to launch Roblox: %w", err)
	}

	logger.LogDebug("Roblox app launched, waiting for startup...")

	// Wait for app to launch
	time.Sleep(1 * time.Second)

	// Note: Actual login automation would require:
	// 1. Opening Roblox login page in browser
	// 2. Using AppleScript to fill in credentials
	// 3. Handling 2FA if enabled
	// For now, we just launch the app and the user logs in manually
	// Future enhancement: Use AppleScript or browser automation

	logger.LogInfo("Roblox instance launched successfully for user: %s (manual login required)", username)
	return nil
}

// LaunchWithoutAccount launches Roblox without attempting login
func LaunchWithoutAccount() error {
	logger.LogInfo("LaunchWithoutAccount called")

	if err := open_app.Open("/Applications/Roblox.app"); err != nil {
		logger.LogError("Failed to launch Roblox: %v", err)
		return err
	}

	logger.LogInfo("Roblox instance launched successfully (no account)")
	return nil
}

// OpenRobloxLoginPage opens the Roblox login page in default browser
func OpenRobloxLoginPage() error {
	logger.LogDebug("Opening Roblox login page in browser")
	cmd := exec.Command("open", "https://www.roblox.com/login")
	if err := cmd.Run(); err != nil {
		logger.LogError("Failed to open Roblox login page: %v", err)
		return err
	}
	logger.LogInfo("Roblox login page opened in browser")
	return nil
}

// OpenRobloxLogoutPage opens the Roblox logout page first to clear session
func OpenRobloxLogoutPage() error {
	logger.LogDebug("Opening Roblox logout page")
	cmd := exec.Command("open", "https://www.roblox.com/logout")
	if err := cmd.Run(); err != nil {
		logger.LogError("Failed to open Roblox logout page: %v", err)
		return err
	}
	logger.LogInfo("Roblox logout page opened")
	return nil
}
