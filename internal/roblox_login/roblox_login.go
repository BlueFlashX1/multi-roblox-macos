package roblox_login

import (
	"fmt"
	"insadem/multi_roblox_macos/internal/open_app"
	"os/exec"
	"time"
)

// LaunchWithAccount launches Roblox and attempts to auto-login
// Note: This is a simplified version. Full automation would require
// AppleScript/UI automation which is complex and fragile.
func LaunchWithAccount(username, password string) error {
	// Launch Roblox
	if err := open_app.Open("/Applications/Roblox.app"); err != nil {
		return fmt.Errorf("failed to launch Roblox: %w", err)
	}

	// Wait for app to launch
	time.Sleep(1 * time.Second)

	// Note: Actual login automation would require:
	// 1. Opening Roblox login page in browser
	// 2. Using AppleScript to fill in credentials
	// 3. Handling 2FA if enabled
	// For now, we just launch the app and the user logs in manually
	// Future enhancement: Use AppleScript or browser automation

	return nil
}

// LaunchWithoutAccount launches Roblox without attempting login
func LaunchWithoutAccount() error {
	return open_app.Open("/Applications/Roblox.app")
}

// OpenRobloxLoginPage opens the Roblox login page in default browser
func OpenRobloxLoginPage() error {
	cmd := exec.Command("open", "https://www.roblox.com/login")
	return cmd.Run()
}
