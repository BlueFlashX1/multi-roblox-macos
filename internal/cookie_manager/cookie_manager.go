package cookie_manager

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"insadem/multi_roblox_macos/internal/logger"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// RobloxCookie represents a .ROBLOSECURITY cookie
type RobloxCookie struct {
	Value      string
	ExpiresUTC int64
	AccountID  string
}

// GetVivaldiCookiePath returns the path to Vivaldi's cookie database
func GetVivaldiCookiePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Vivaldi stores cookies in the Default profile
	cookiePath := filepath.Join(home, "Library", "Application Support", "Vivaldi", "Default", "Cookies")

	if _, err := os.Stat(cookiePath); os.IsNotExist(err) {
		return "", fmt.Errorf("Vivaldi cookie database not found at: %s", cookiePath)
	}

	return cookiePath, nil
}

// pinnedPackages maps each logical package name to its pinned version specifier.
// Versions pinned 2026-05-16. Update when a security patch is released upstream.
//
// Security rationale: unpinned `pip install` is a supply-chain attack surface —
// a typosquat or compromise of browser_cookie3 / pycryptodomex would silently
// install under the user's UID. Pinning to known-good versions limits the blast
// radius to version-specific vulnerabilities rather than arbitrary malicious code.
var pinnedPackages = map[string]string{
	"browser_cookie3": "browser_cookie3==0.19.1",
	"pycryptodomex":   "pycryptodomex==3.20.0",
	"lz4":             "lz4==4.3.3",
}

// runPythonWithAutoInstall runs a Python script and auto-installs missing packages.
// It tries multiple Python paths to find one that works.
//
// Security: packages are installed pinned to known-good versions via --user only.
// The system-wide fallback (install without --user) has been removed — it polluted
// other tools' Python environments and could silently escalate privileges. If the
// --user install fails, the error is surfaced to the caller rather than retried.
func runPythonWithAutoInstall(script string, requiredPackages []string) ([]byte, error) {
	// Try multiple Python paths in order of preference
	pythonPaths := []string{
		"/usr/local/bin/python3",
		"/usr/bin/python3",
		"python3",
	}

	var lastErr error
	var lastOutput []byte

	for _, pythonPath := range pythonPaths {
		// First, try running the script directly
		cmd := exec.Command(pythonPath, "-c", script)
		output, err := cmd.CombinedOutput()

		if err == nil {
			return output, nil
		}

		outputStr := string(output)
		lastOutput = output
		lastErr = err

		// Check if it's a missing module error
		if strings.Contains(outputStr, "ModuleNotFoundError") || strings.Contains(outputStr, "No module named") {
			logger.LogInfo("Missing Python package detected, attempting auto-install...")

			// Announce what we are about to install so the user can abort (Ctrl-C).
			// This is the consent notice before touching their Python environment.
			// Sort keys for deterministic output — map iteration order is random,
			// producing unstable log output and making log diffs harder to read.
			var pinned []string
			sortedPkgs := make([]string, len(requiredPackages))
			copy(sortedPkgs, requiredPackages)
			sort.Strings(sortedPkgs)
			for _, pkg := range sortedPkgs {
				spec, ok := pinnedPackages[pkg]
				if !ok {
					// Guard: refuse to proceed with an unpinned package. A caller that
					// adds a new required package without updating pinnedPackages would
					// silently re-open the supply-chain attack surface this map closes.
					return nil, fmt.Errorf("refusing to install unpinned package %q — add to pinnedPackages map first", pkg)
				}
				pinned = append(pinned, spec)
			}
			consentMsg := fmt.Sprintf("[multi-roblox] About to pip-install (--user): %s. Press Ctrl-C to abort.",
				strings.Join(pinned, ", "))
			fmt.Fprintln(os.Stderr, consentMsg)
			// Also log to the rotating log file — users who launched via Finder/Spotlight
			// have no terminal, so stderr is silently discarded. The log file is always
			// written regardless of launch method.
			logger.LogInfo("%s", consentMsg)

			// Install all required packages at pinned versions, --user only.
			for _, pkg := range requiredPackages {
				spec, ok := pinnedPackages[pkg]
				if !ok {
					// This branch is unreachable after the guard above, but kept for
					// defensive completeness — it ensures the loop can never fall through
					// to an unpinned install even if the guard logic is refactored.
					return nil, fmt.Errorf("refusing to install unpinned package %q — add to pinnedPackages map first", pkg)
				}

				logger.LogInfo("Auto-installing Python package: %s using %s", spec, pythonPath)

				installCmd := exec.Command(pythonPath, "-m", "pip", "install", "--user", spec)
				installOutput, installErr := installCmd.CombinedOutput()
				// Show pip output so the user sees what's happening (no --quiet).
				if len(installOutput) > 0 {
					fmt.Fprintf(os.Stderr, "[multi-roblox] pip: %s", string(installOutput))
				}

				if installErr != nil {
					logger.LogError("Failed to install %s with --user: %v", spec, installErr)
					// Do NOT retry without --user — that would pollute the system
					// Python environment. Surface the error instead.
					return nil, fmt.Errorf("pip install --user %s failed: %w\noutput: %s", spec, installErr, string(installOutput))
				}

				logger.LogInfo("Successfully installed %s", spec)
			}

			// Retry the script after installing
			cmd = exec.Command(pythonPath, "-c", script)
			output, err = cmd.CombinedOutput()
			if err == nil {
				logger.LogInfo("Script succeeded after auto-installing packages")
				return output, nil
			}
			lastOutput = output
			lastErr = err
		}
	}

	return lastOutput, fmt.Errorf("all Python paths failed: %w", lastErr)
}

// GetCurrentRobloxCookie reads the current .ROBLOSECURITY cookie from Vivaldi using browser_cookie3
func GetCurrentRobloxCookie() (*RobloxCookie, error) {
	logger.LogInfo("Reading .ROBLOSECURITY cookie from Vivaldi using browser_cookie3")

	// Use Python's browser_cookie3 library which handles decryption automatically
	pythonScript := `
import browser_cookie3
import json

try:
    cj = browser_cookie3.vivaldi(domain_name='.roblox.com')
    for cookie in cj:
        if cookie.name == '.ROBLOSECURITY':
            result = {
                "value": cookie.value,
                "expires": cookie.expires if cookie.expires else 0
            }
            print(json.dumps(result))
            exit(0)
    print(json.dumps({"error": "No .ROBLOSECURITY cookie found - are you logged into Roblox in Vivaldi?"}))
except Exception as e:
    print(json.dumps({"error": str(e)}))
`

	// Required packages for cookie reading
	requiredPackages := []string{"browser_cookie3", "pycryptodomex", "lz4"}

	output, err := runPythonWithAutoInstall(pythonScript, requiredPackages)
	if err != nil {
		return nil, fmt.Errorf("failed to read cookie: %w, output: %s", err, string(output))
	}

	// Parse JSON result
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cookie result: %w, output: %s", err, string(output))
	}

	if errMsg, ok := result["error"].(string); ok {
		return nil, fmt.Errorf("%s", errMsg)
	}

	value, ok := result["value"].(string)
	if !ok || value == "" {
		return nil, fmt.Errorf("empty cookie value")
	}

	var expiresUTC int64
	if exp, ok := result["expires"].(float64); ok {
		expiresUTC = int64(exp)
	}

	logger.LogInfo("Successfully read .ROBLOSECURITY cookie (length: %d)", len(value))

	return &RobloxCookie{
		Value:      value,
		ExpiresUTC: expiresUTC,
	}, nil
}

// SaveCookieForAccount saves a .ROBLOSECURITY cookie to Keychain for an account
func SaveCookieForAccount(accountID string, cookie *RobloxCookie) error {
	logger.LogInfo("Saving .ROBLOSECURITY cookie for account: %s", accountID)

	// Store in Keychain
	keychainService := "multi-roblox-cookie"

	// Delete existing entry
	exec.Command("security", "delete-generic-password",
		"-s", keychainService,
		"-a", accountID).Run()

	// Add new entry
	cmd := exec.Command("security", "add-generic-password",
		"-s", keychainService,
		"-a", accountID,
		"-w", cookie.Value,
		"-U")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to save cookie to keychain: %w", err)
	}

	logger.LogInfo("Cookie saved to Keychain for account: %s", accountID)
	return nil
}

// GetCookieForAccount retrieves a saved .ROBLOSECURITY cookie from Keychain
func GetCookieForAccount(accountID string) (*RobloxCookie, error) {
	logger.LogDebug("Getting saved cookie for account: %s", accountID)

	keychainService := "multi-roblox-cookie"

	cmd := exec.Command("security", "find-generic-password",
		"-s", keychainService,
		"-a", accountID,
		"-w")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("no saved cookie for account: %s", accountID)
	}

	cookieValue := strings.TrimSpace(string(output))
	if cookieValue == "" {
		return nil, fmt.Errorf("empty cookie for account: %s", accountID)
	}

	return &RobloxCookie{
		Value:     cookieValue,
		AccountID: accountID,
	}, nil
}

// SetRobloxCookie sets the .ROBLOSECURITY cookie in Vivaldi
func SetRobloxCookie(cookie *RobloxCookie) error {
	logger.LogInfo("Setting .ROBLOSECURITY cookie in Vivaldi")

	// Check if Vivaldi is running
	if isVivaldiRunning() {
		return fmt.Errorf("please close Vivaldi before switching accounts")
	}

	cookiePath, err := GetVivaldiCookiePath()
	if err != nil {
		return err
	}

	// Open the SQLite database
	db, err := sql.Open("sqlite3", cookiePath)
	if err != nil {
		return fmt.Errorf("failed to open cookie database: %w", err)
	}
	defer db.Close()

	// Calculate timestamps using Chrome epoch (microseconds since 1601-01-01)
	windowsEpoch := time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC)
	nowMicros := time.Now().Sub(windowsEpoch).Microseconds()
	expiresUTC := time.Now().Add(365 * 24 * time.Hour).Sub(windowsEpoch).Microseconds()

	// Delete existing cookie
	_, err = db.Exec(`DELETE FROM cookies WHERE host_key = '.roblox.com' AND name = '.ROBLOSECURITY'`)
	if err != nil {
		logger.LogError("Failed to delete existing cookie: %v", err)
	}

	// Insert new cookie using Vivaldi's actual schema
	// Note: We use the plain 'value' column - Vivaldi will work with unencrypted cookies
	_, err = db.Exec(`INSERT INTO cookies (
		creation_utc, host_key, top_frame_site_key, name, value, encrypted_value,
		path, expires_utc, is_secure, is_httponly, last_access_utc, has_expires,
		is_persistent, priority, samesite, source_scheme, source_port,
		last_update_utc, source_type, has_cross_site_ancestor
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nowMicros,        // creation_utc
		".roblox.com",    // host_key
		"",               // top_frame_site_key
		".ROBLOSECURITY", // name
		cookie.Value,     // value (unencrypted)
		[]byte{},         // encrypted_value (empty)
		"/",              // path
		expiresUTC,       // expires_utc
		1,                // is_secure
		1,                // is_httponly
		nowMicros,        // last_access_utc
		1,                // has_expires
		1,                // is_persistent
		1,                // priority
		0,                // samesite (None)
		2,                // source_scheme (HTTPS)
		443,              // source_port
		nowMicros,        // last_update_utc
		0,                // source_type
		0,                // has_cross_site_ancestor
	)

	if err != nil {
		return fmt.Errorf("failed to insert cookie: %w", err)
	}

	logger.LogInfo("Successfully set .ROBLOSECURITY cookie in Vivaldi")
	return nil
}

// HasSavedCookie checks if an account has a saved cookie
func HasSavedCookie(accountID string) bool {
	_, err := GetCookieForAccount(accountID)
	return err == nil
}

// CookieStatus represents the validation status of a saved cookie
type CookieStatus int

const (
	CookieStatusNone    CookieStatus = iota // No cookie saved
	CookieStatusValid                       // Cookie is valid
	CookieStatusExpired                     // Cookie is expired/invalid
	CookieStatusError                       // Error checking cookie
)

// CookieValidationResult contains the result of cookie validation
type CookieValidationResult struct {
	Status          CookieStatus
	Username        string // The username the cookie belongs to (if valid)
	ErrorMessage    string // Error message if any
	DaysUntilExpiry int    // Days until cookie expires (-1 if unknown, 0 if expired)
	ExpiresWarning  bool   // True if cookie expires within 7 days
}

// ValidateCookieForAccount validates the saved cookie for an account
func ValidateCookieForAccount(accountID string) CookieValidationResult {
	cookie, err := GetCookieForAccount(accountID)
	if err != nil {
		return CookieValidationResult{Status: CookieStatusNone, DaysUntilExpiry: -1}
	}

	username, err := VerifyCookieUsername(cookie.Value)
	if err != nil {
		// Check if it's an auth error (expired) vs network error
		errStr := err.Error()
		if strings.Contains(errStr, "401") || strings.Contains(errStr, "expired") || strings.Contains(errStr, "invalid") {
			return CookieValidationResult{
				Status:          CookieStatusExpired,
				ErrorMessage:    "Cookie expired - recapture needed",
				DaysUntilExpiry: 0,
			}
		}
		return CookieValidationResult{
			Status:          CookieStatusError,
			ErrorMessage:    errStr,
			DaysUntilExpiry: -1,
		}
	}

	// Calculate days until expiry (cookies typically last 30 days)
	daysLeft := -1
	expiresWarning := false
	if cookie.ExpiresUTC > 0 {
		expiresAt := time.Unix(cookie.ExpiresUTC, 0)
		daysLeft = int(time.Until(expiresAt).Hours() / 24)
		if daysLeft < 0 {
			daysLeft = 0
		}
		expiresWarning = daysLeft <= 7
	}

	return CookieValidationResult{
		Status:          CookieStatusValid,
		Username:        username,
		DaysUntilExpiry: daysLeft,
		ExpiresWarning:  expiresWarning,
	}
}

// TryRefreshCookieFromBrowser attempts to refresh a cookie from Vivaldi if logged in as that account
// Returns true if refresh was successful
func TryRefreshCookieFromBrowser(accountID string, expectedUsername string) (bool, error) {
	logger.LogInfo("Attempting to refresh cookie for %s from browser...", expectedUsername)

	// Get current browser cookie
	browserCookie, err := GetCurrentRobloxCookie()
	if err != nil {
		return false, fmt.Errorf("failed to get browser cookie: %w", err)
	}

	// Verify it matches the expected account
	browserUsername, err := VerifyCookieUsername(browserCookie.Value)
	if err != nil {
		return false, fmt.Errorf("failed to verify browser cookie: %w", err)
	}

	if !strings.EqualFold(browserUsername, expectedUsername) {
		return false, fmt.Errorf("browser logged in as %s, not %s", browserUsername, expectedUsername)
	}

	// Save the refreshed cookie
	if err := SaveCookieForAccount(accountID, browserCookie); err != nil {
		return false, fmt.Errorf("failed to save refreshed cookie: %w", err)
	}

	logger.LogInfo("Successfully refreshed cookie for %s!", expectedUsername)
	return true, nil
}

// PreLaunchCookieCheck validates cookie before launch and tries to refresh if needed
// Returns the cookie value if valid, or error if can't be used
func PreLaunchCookieCheck(accountID string, expectedUsername string) (string, error) {
	logger.LogInfo("Pre-launch cookie check for %s...", expectedUsername)

	result := ValidateCookieForAccount(accountID)

	// If valid and not expiring soon, use it
	if result.Status == CookieStatusValid && !result.ExpiresWarning {
		cookie, _ := GetCookieForAccount(accountID)
		return cookie.Value, nil
	}

	// If expired or expiring soon, try to refresh from browser
	if result.Status == CookieStatusExpired || result.ExpiresWarning {
		logger.LogInfo("Cookie for %s needs refresh (status: %v, expiring: %v)", expectedUsername, result.Status, result.ExpiresWarning)

		refreshed, err := TryRefreshCookieFromBrowser(accountID, expectedUsername)
		if refreshed {
			cookie, _ := GetCookieForAccount(accountID)
			return cookie.Value, nil
		}

		if result.Status == CookieStatusExpired {
			return "", fmt.Errorf("cookie expired and couldn't refresh: %v", err)
		}

		// Expiring soon but couldn't refresh - still usable
		logger.LogInfo("Cookie expiring soon but couldn't refresh, using anyway")
		cookie, _ := GetCookieForAccount(accountID)
		return cookie.Value, nil
	}

	if result.Status == CookieStatusNone {
		return "", fmt.Errorf("no cookie saved for this account")
	}

	return "", fmt.Errorf("cookie error: %s", result.ErrorMessage)
}

// CleanupTempRobloxCopies removes temporary Roblox app copies that aren't in use
func CleanupTempRobloxCopies() {
	logger.LogInfo("Cleaning up temporary Roblox app copies...")

	// Get list of currently running Roblox processes and their paths
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		logger.LogError("Failed to get process list: %v", err)
		return
	}
	psOutput := string(output)

	cleaned := 0
	for i := 2; i <= 10; i++ {
		path := fmt.Sprintf("/tmp/Roblox%d.app", i)
		if _, err := os.Stat(path); err == nil {
			// Check if this copy is currently in use
			if strings.Contains(psOutput, path) {
				logger.LogDebug("Skipping %s - still in use", path)
				continue
			}

			// Not in use, safe to remove
			if err := os.RemoveAll(path); err != nil {
				logger.LogError("Failed to remove %s: %v", path, err)
			} else {
				logger.LogDebug("Removed: %s", path)
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		logger.LogInfo("Cleaned up %d temporary Roblox copies", cleaned)
	}
}

// AutoRefreshCookieResult contains info about a cookie refresh attempt
type AutoRefreshCookieResult struct {
	AccountID  string
	Username   string
	WasExpired bool
	Refreshed  bool
	Error      string
}

// AutoRefreshExpiredCookies checks all saved cookies and refreshes from Vivaldi if expired
// Returns a list of results for each account
func AutoRefreshExpiredCookies(accountIDs []string, accountUsernames map[string]string) []AutoRefreshCookieResult {
	logger.LogInfo("Checking cookies for auto-refresh...")
	var results []AutoRefreshCookieResult

	// Get current browser cookie
	browserCookie, browserErr := GetCurrentRobloxCookie()
	var browserUsername string
	if browserErr == nil {
		browserUsername, _ = VerifyCookieUsername(browserCookie.Value)
	}

	for _, accountID := range accountIDs {
		expectedUsername := accountUsernames[accountID]
		result := AutoRefreshCookieResult{
			AccountID: accountID,
			Username:  expectedUsername,
		}

		// Check if cookie exists and is valid
		validationResult := ValidateCookieForAccount(accountID)

		if validationResult.Status == CookieStatusValid {
			// Cookie is valid, no refresh needed
			logger.LogDebug("Cookie for %s is valid", expectedUsername)
			results = append(results, result)
			continue
		}

		if validationResult.Status == CookieStatusExpired {
			result.WasExpired = true
			logger.LogInfo("Cookie for %s is EXPIRED, checking if browser has valid session...", expectedUsername)

			// Check if browser cookie matches this account
			if browserErr == nil && browserUsername == expectedUsername {
				logger.LogInfo("Browser has valid session for %s, auto-refreshing cookie!", expectedUsername)

				// Save the browser cookie for this account
				if err := SaveCookieForAccount(accountID, browserCookie); err != nil {
					result.Error = fmt.Sprintf("Failed to save: %v", err)
					logger.LogError("Failed to auto-refresh cookie for %s: %v", expectedUsername, err)
				} else {
					result.Refreshed = true
					logger.LogInfo("Successfully auto-refreshed cookie for %s!", expectedUsername)
				}
			} else {
				result.Error = "Browser not logged in as this account"
				logger.LogDebug("Browser is logged in as %s, not %s - can't auto-refresh", browserUsername, expectedUsername)
			}
		}

		results = append(results, result)
	}

	return results
}

// isVivaldiRunning checks if Vivaldi browser is running
func isVivaldiRunning() bool {
	cmd := exec.Command("pgrep", "-x", "Vivaldi")
	err := cmd.Run()
	return err == nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0600) // Secure permissions - owner only
}

// decryptChromeValue attempts to decrypt a Chrome/Vivaldi encrypted cookie value
// On macOS, Chrome uses the Keychain to store the encryption key
func decryptChromeValue(encrypted []byte) (string, error) {
	if len(encrypted) == 0 {
		return "", fmt.Errorf("empty encrypted value")
	}

	// Check for v10 prefix (Chrome 80+ encryption)
	if len(encrypted) > 3 && string(encrypted[:3]) == "v10" {
		// v10 encryption uses AES-256-GCM with key from Keychain
		// The key is stored in Keychain under "Chrome Safe Storage" or "Vivaldi Safe Storage"
		return decryptV10(encrypted)
	}

	// Older format or unencrypted
	return string(encrypted), nil
}

// decryptV10 decrypts v10 encrypted Chrome cookies on macOS
func decryptV10(encrypted []byte) (string, error) {
	// Get the encryption key from Keychain
	// Vivaldi uses "Vivaldi Safe Storage" as the service name
	cmd := exec.Command("security", "find-generic-password",
		"-s", "Vivaldi Safe Storage",
		"-w")

	keyOutput, err := cmd.Output()
	if err != nil {
		// Try Chrome key as fallback
		cmd = exec.Command("security", "find-generic-password",
			"-s", "Chrome Safe Storage",
			"-w")
		keyOutput, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get encryption key from Keychain: %w", err)
		}
	}

	key := strings.TrimSpace(string(keyOutput))
	if key == "" {
		return "", fmt.Errorf("empty encryption key")
	}

	// The encrypted value format is: "v10" + 12-byte nonce + ciphertext + 16-byte tag
	if len(encrypted) < 3+12+16 {
		return "", fmt.Errorf("encrypted value too short")
	}

	// For now, return a placeholder - full AES-GCM decryption would require crypto libraries
	// In practice, we'll use the command-line approach instead
	logger.LogDebug("Cookie is v10 encrypted, attempting external decryption...")

	// Try using Python for decryption (more reliable on macOS)
	return decryptWithPython(encrypted, key)
}

// decryptWithPython uses Python's cryptography library to decrypt
// macOS Chromium uses AES-128-CBC with PKCS7 padding
func decryptWithPython(encrypted []byte, key string) (string, error) {
	// Encode the encrypted value as base64 for passing to Python via env var.
	// The Vivaldi Safe Storage key is passed via env var — never interpolated
	// into the script source to avoid injection if the key contains quotes or
	// newlines.
	encB64 := base64.StdEncoding.EncodeToString(encrypted)

	const pythonScript = `
import base64
import hashlib
import os
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend

encrypted = base64.b64decode(os.environ['MRM_ENC_B64'])
password = os.environ['MRM_CRYPTO_KEY']

# Derive key using PBKDF2 (macOS Chrome uses 1003 iterations, 16 byte key)
key = hashlib.pbkdf2_hmac('sha1', password.encode('utf-8'), b'saltysalt', 1003, dklen=16)

# v10 format on macOS: "v10" + 16-byte IV + ciphertext (AES-128-CBC)
if encrypted[:3] == b'v10':
    iv = b' ' * 16  # macOS uses space-filled IV
    ciphertext = encrypted[3:]

    cipher = Cipher(algorithms.AES(key), modes.CBC(iv), backend=default_backend())
    decryptor = cipher.decryptor()

    try:
        decrypted_padded = decryptor.update(ciphertext) + decryptor.finalize()
        # Remove PKCS7 padding
        pad_len = decrypted_padded[-1]
        if isinstance(pad_len, int) and pad_len <= 16:
            decrypted = decrypted_padded[:-pad_len]
        else:
            decrypted = decrypted_padded
        print(decrypted.decode('utf-8'))
    except Exception as e:
        print("DECRYPTION_FAILED:" + str(e))
else:
    # Not v10 encrypted, try as plain text
    print(encrypted.decode('utf-8', errors='ignore'))
`

	cmd := exec.Command("python3", "-c", pythonScript)
	cmd.Env = append(os.Environ(),
		"MRM_ENC_B64="+encB64,
		"MRM_CRYPTO_KEY="+key,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Python decryption failed: %w, output: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	if strings.HasPrefix(result, "DECRYPTION_FAILED:") {
		return "", fmt.Errorf("decryption failed: %s", result)
	}

	return result, nil
}

// ClearSavedCookie removes a saved cookie for an account
func ClearSavedCookie(accountID string) error {
	keychainService := "multi-roblox-cookie"
	cmd := exec.Command("security", "delete-generic-password",
		"-s", keychainService,
		"-a", accountID)
	return cmd.Run()
}

// GetAuthTicket gets a Roblox authentication ticket from a cookie
// This ticket can be passed in the launch URL for proper authentication
func GetAuthTicket(cookieValue string) (string, error) {
	logger.LogInfo("Getting Roblox auth ticket from cookie...")

	// Cookie is passed via environment variable — never interpolated into the
	// script source to prevent injection if the cookie value contains quotes or
	// newlines (possible in a poisoned local DB row).
	const pythonScript = `
import urllib.request
import json
import os

cookie = os.environ['MRM_ROBLOX_COOKIE']

# First request to get CSRF token
try:
    req = urllib.request.Request('https://auth.roblox.com/v1/authentication-ticket', method='POST')
    req.add_header('Cookie', '.ROBLOSECURITY=' + cookie)
    req.add_header('Content-Type', 'application/json')
    req.add_header('Referer', 'https://www.roblox.com/')

    with urllib.request.urlopen(req, data=b'', timeout=10) as response:
        ticket = response.headers.get('rbx-authentication-ticket')
        if ticket:
            print(json.dumps({"ticket": ticket}))
        else:
            print(json.dumps({"error": "No ticket in response"}))
except urllib.error.HTTPError as e:
    if e.code == 403:
        csrf = e.headers.get('x-csrf-token')
        if csrf:
            # Retry with CSRF token
            req2 = urllib.request.Request('https://auth.roblox.com/v1/authentication-ticket', method='POST')
            req2.add_header('Cookie', '.ROBLOSECURITY=' + cookie)
            req2.add_header('Content-Type', 'application/json')
            req2.add_header('X-CSRF-TOKEN', csrf)
            req2.add_header('Referer', 'https://www.roblox.com/')

            try:
                with urllib.request.urlopen(req2, data=b'', timeout=10) as response2:
                    ticket = response2.headers.get('rbx-authentication-ticket')
                    if ticket:
                        print(json.dumps({"ticket": ticket}))
                    else:
                        print(json.dumps({"error": "No ticket after CSRF retry"}))
            except Exception as e2:
                print(json.dumps({"error": str(e2)}))
        else:
            print(json.dumps({"error": "403 but no CSRF token"}))
    else:
        print(json.dumps({"error": f"HTTP {e.code}: {e.reason}"}))
except Exception as e:
    print(json.dumps({"error": str(e)}))
`

	cmd := exec.Command("python3", "-c", pythonScript)
	cmd.Env = append(os.Environ(), "MRM_ROBLOX_COOKIE="+cookieValue)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get auth ticket: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse auth ticket response: %s", string(output))
	}

	if errMsg, ok := result["error"].(string); ok {
		return "", fmt.Errorf("%s", errMsg)
	}

	ticket, ok := result["ticket"].(string)
	if !ok || ticket == "" {
		return "", fmt.Errorf("no auth ticket in response")
	}

	logger.LogInfo("Successfully got auth ticket (length: %d)", len(ticket))
	return ticket, nil
}

// VerifyCookieUsername uses Roblox API to get the username associated with a cookie
func VerifyCookieUsername(cookieValue string) (string, error) {
	logger.LogInfo("Verifying cookie against Roblox API...")

	// Cookie passed via environment variable — never interpolated into script source.
	const pythonScript = `
import urllib.request
import json
import os

cookie = os.environ['MRM_ROBLOX_COOKIE']

req = urllib.request.Request('https://users.roblox.com/v1/users/authenticated')
req.add_header('Cookie', '.ROBLOSECURITY=' + cookie)

try:
    with urllib.request.urlopen(req, timeout=10) as response:
        data = json.loads(response.read().decode('utf-8'))
        print(json.dumps({"username": data.get("name", ""), "id": data.get("id", 0)}))
except urllib.error.HTTPError as e:
    print(json.dumps({"error": f"HTTP {e.code}: Cookie may be invalid or expired"}))
except Exception as e:
    print(json.dumps({"error": str(e)}))
`

	cmd := exec.Command("python3", "-c", pythonScript)
	cmd.Env = append(os.Environ(), "MRM_ROBLOX_COOKIE="+cookieValue)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %s", string(output))
	}

	if errMsg, ok := result["error"].(string); ok {
		return "", fmt.Errorf("%s", errMsg)
	}

	username, _ := result["username"].(string)
	return username, nil
}

// GetCurrentBrowserCookieUsername gets the username from the currently active browser cookie
func GetCurrentBrowserCookieUsername() (string, error) {
	cookie, err := GetCurrentRobloxCookie()
	if err != nil {
		return "", err
	}
	return VerifyCookieUsername(cookie.Value)
}

// ClearRobloxAppCookies deletes ALL Roblox app cached data including WebKit
// This forces Roblox to start fresh and use our cookie
func ClearRobloxAppCookies() error {
	logger.LogInfo("Clearing ALL Roblox app cached data (cookies + WebKit)...")

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// ALL paths where Roblox stores auth/session data
	pathsToRemove := []string{
		// HTTPStorages cookies
		filepath.Join(home, "Library", "HTTPStorages", "com.roblox.RobloxPlayer.binarycookies"),
		filepath.Join(home, "Library", "HTTPStorages", "com.roblox.RobloxPlayer"),
		// WebKit data (THIS IS THE KEY - contains LocalStorage with cached user!)
		filepath.Join(home, "Library", "WebKit", "com.roblox.RobloxPlayer"),
		// Caches
		filepath.Join(home, "Library", "Caches", "com.roblox.RobloxPlayer"),
	}

	for _, path := range pathsToRemove {
		if _, err := os.Stat(path); err == nil {
			info, _ := os.Stat(path)
			if info.IsDir() {
				if err := os.RemoveAll(path); err != nil {
					logger.LogError("Failed to remove %s: %v", path, err)
				} else {
					logger.LogInfo("Removed directory: %s", path)
				}
			} else {
				if err := os.Remove(path); err != nil {
					logger.LogError("Failed to remove %s: %v", path, err)
				} else {
					logger.LogInfo("Removed file: %s", path)
				}
			}
		}
	}

	// Clear preferences
	exec.Command("defaults", "delete", "com.roblox.RobloxPlayer").Run()

	logger.LogInfo("Roblox app data cleared completely")
	return nil
}

// SaveCurrentBrowserCookieToAccount saves the current Vivaldi cookie to the matching account
// This should be called before clearing cookies to preserve the session
func SaveCurrentBrowserCookieToAccount(accounts []struct{ ID, Username string }) error {
	browserUsername, err := GetCurrentBrowserCookieUsername()
	if err != nil || browserUsername == "" {
		logger.LogDebug("No browser session to save")
		return nil
	}

	// Find matching account
	for _, acc := range accounts {
		if strings.EqualFold(acc.Username, browserUsername) {
			// Get the current cookie
			cookie, err := GetCurrentRobloxCookie()
			if err != nil {
				logger.LogError("Failed to get current cookie for saving: %v", err)
				return err
			}

			// Save it to this account
			cookie.AccountID = acc.ID
			if err := SaveCookieForAccount(acc.ID, cookie); err != nil {
				logger.LogError("Failed to save cookie for %s: %v", acc.Username, err)
				return err
			}

			logger.LogInfo("Saved current browser cookie to account: %s", acc.Username)
			return nil
		}
	}

	logger.LogDebug("No matching account found for browser user: %s", browserUsername)
	return nil
}

// ClearVivaldiRobloxCookies clears Roblox cookies from Vivaldi browser
// This forces the user to log in again when visiting Roblox in the browser
func ClearVivaldiRobloxCookies() error {
	logger.LogInfo("Clearing Roblox cookies from Vivaldi...")

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Path to Vivaldi's cookies database
	cookiesPath := filepath.Join(home, "Library", "Application Support", "Vivaldi", "Default", "Cookies")

	// We need to use sqlite3 to delete the cookies
	// First, close Vivaldi if running
	exec.Command("pkill", "-x", "Vivaldi").Run()
	time.Sleep(500 * time.Millisecond)

	// Delete Roblox cookies from the database
	deleteSQL := `DELETE FROM cookies WHERE host_key LIKE '%roblox.com';`
	cmd := exec.Command("sqlite3", cookiesPath, deleteSQL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.LogError("Failed to clear Vivaldi cookies: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to clear cookies: %v", err)
	}

	logger.LogInfo("Vivaldi Roblox cookies cleared successfully")
	return nil
}

// SetRobloxAppCookie writes a cookie directly to Roblox app's binarycookies file
// This is the most reliable way to switch accounts as it bypasses browser dependency
func SetRobloxAppCookie(cookieValue string) error {
	logger.LogInfo("Writing cookie directly to Roblox app's binarycookies...")

	// Cookie passed via environment variable — never interpolated into script source
	// to prevent injection if the cookie value contains quotes, newlines, or other
	// metacharacters (possible in a poisoned local DB row).
	const pythonScript = `
import struct
import os
from datetime import datetime

class BinaryCookieWriter:
    def __init__(self):
        self.pages = []

    def add_cookie(self, name, value, domain, path="/", expires=None, secure=True, http_only=True):
        if expires is None:
            expires = datetime(2030, 1, 1)

        mac_epoch = datetime(2001, 1, 1)
        expiry_time = (expires - mac_epoch).total_seconds()
        creation_time = (datetime.now() - mac_epoch).total_seconds()

        self.pages.append({
            'name': name,
            'value': value,
            'domain': domain,
            'path': path,
            'expiry': expiry_time,
            'creation': creation_time,
            'flags': (1 if secure else 0) | (4 if http_only else 0)
        })

    def write(self, filepath):
        magic = b'cook'
        num_pages = 1

        cookies_data = b''
        cookie_offsets = []

        for cookie in self.pages:
            cookie_start = len(cookies_data)
            cookie_offsets.append(cookie_start)

            name_bytes = cookie['name'].encode('utf-8') + b'\x00'
            value_bytes = cookie['value'].encode('utf-8') + b'\x00'
            domain_bytes = cookie['domain'].encode('utf-8') + b'\x00'
            path_bytes = cookie['path'].encode('utf-8') + b'\x00'

            cookie_size = 56 + len(name_bytes) + len(value_bytes) + len(domain_bytes) + len(path_bytes)

            url_offset = 56
            name_offset = url_offset + len(domain_bytes)
            path_offset = name_offset + len(name_bytes)
            value_offset = path_offset + len(path_bytes)

            cookie_header = struct.pack('<I', cookie_size)
            cookie_header += struct.pack('<I', 0)
            cookie_header += struct.pack('<I', cookie['flags'])
            cookie_header += struct.pack('<I', 0)
            cookie_header += struct.pack('<I', url_offset)
            cookie_header += struct.pack('<I', name_offset)
            cookie_header += struct.pack('<I', path_offset)
            cookie_header += struct.pack('<I', value_offset)
            cookie_header += struct.pack('<I', 0)
            cookie_header += struct.pack('<d', cookie['expiry'])
            cookie_header += struct.pack('<d', cookie['creation'])

            cookies_data += cookie_header + domain_bytes + name_bytes + path_bytes + value_bytes

        page_header = struct.pack('>I', 0x00000100)
        page_header += struct.pack('<I', len(self.pages))

        for offset in cookie_offsets:
            page_header += struct.pack('<I', offset + len(page_header) + len(cookie_offsets) * 4)

        page_data = page_header + cookies_data
        page_size = len(page_data)

        file_data = magic
        file_data += struct.pack('>I', num_pages)
        file_data += struct.pack('>I', page_size)
        file_data += page_data
        file_data += struct.pack('>I', 0)

        with open(filepath, 'wb') as f:
            f.write(file_data)

        return True

cookie_value = os.environ['MRM_ROBLOX_COOKIE']

writer = BinaryCookieWriter()
writer.add_cookie(
    name='.ROBLOSECURITY',
    value=cookie_value,
    domain='.roblox.com',
    path='/',
    secure=True,
    http_only=True
)

output_path = os.path.expanduser('~/Library/HTTPStorages/com.roblox.RobloxPlayer.binarycookies')
writer.write(output_path)
print("SUCCESS")
`

	cmd := exec.Command("python3", "-c", pythonScript)
	cmd.Env = append(os.Environ(), "MRM_ROBLOX_COOKIE="+cookieValue)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to write binarycookies: %w, output: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	if result != "SUCCESS" {
		return fmt.Errorf("failed to write binarycookies: %s", result)
	}

	logger.LogInfo("Successfully wrote cookie to Roblox app's binarycookies")
	return nil
}
