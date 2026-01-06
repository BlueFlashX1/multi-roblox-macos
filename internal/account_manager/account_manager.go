package account_manager

import (
	"encoding/json"
	"fmt"
	"insadem/multi_roblox_macos/internal/logger"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Account represents a Roblox account
type Account struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Label    string `json:"label"` // e.g., "Main Account", "Alt 1"
}

const (
	keychainService = "multi-roblox-manager"
	accountsFile    = "accounts.json"
)

// GetAccountsPath returns the path to the accounts file
func GetAccountsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "multi_roblox_macos", accountsFile)
}

// LoadAccounts loads all accounts from disk
func LoadAccounts() ([]Account, error) {
	path := GetAccountsPath()

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty list
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []Account{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var accounts []Account
	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

// SaveAccounts saves accounts to disk
func SaveAccounts(accounts []Account) error {
	path := GetAccountsPath()

	data, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// AddAccount adds a new account with secure password storage
func AddAccount(username, password, label string) error {
	logger.LogInfo("AddAccount called for username: %s, label: %s", username, label)

	accounts, err := LoadAccounts()
	if err != nil {
		logger.LogError("Failed to load accounts: %v", err)
		return err
	}

	// Generate unique ID
	id := fmt.Sprintf("account_%d", len(accounts)+1)
	logger.LogDebug("Generated account ID: %s", id)

	// Store password in macOS Keychain
	if err := storePasswordInKeychain(id, password); err != nil {
		logger.LogError("Failed to store password in keychain for %s: %v", username, err)
		return fmt.Errorf("failed to store password in keychain: %w", err)
	}
	logger.LogDebug("Password stored in keychain for account ID: %s", id)

	// Add account metadata (no password stored here)
	account := Account{
		ID:       id,
		Username: username,
		Label:    label,
	}

	accounts = append(accounts, account)
	if err := SaveAccounts(accounts); err != nil {
		logger.LogError("Failed to save accounts: %v", err)
		return err
	}

	logger.LogInfo("Account added successfully: %s (ID: %s)", username, id)
	return nil
}

// GetPassword retrieves password from macOS Keychain
func GetPassword(accountID string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", keychainService,
		"-a", accountID,
		"-w")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("password not found in keychain")
	}

	return strings.TrimSpace(string(output)), nil
}

// storePasswordInKeychain stores password securely in macOS Keychain with enhanced security
func storePasswordInKeychain(accountID, password string) error {
	// First, try to delete existing entry (ignore errors)
	exec.Command("security", "delete-generic-password",
		"-s", keychainService,
		"-a", accountID).Run()

	// Add new entry with enhanced security flags
	cmd := exec.Command("security", "add-generic-password",
		"-s", keychainService,
		"-a", accountID,
		"-w", password,
		"-T", "", // Trusted applications (empty = only this app)
		"-U") // Update if exists

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to store password: %w (output: %s)", err, string(output))
	}

	// Clear password from memory
	password = ""

	return nil
}

// DeleteAccount removes account and its password from keychain
func DeleteAccount(accountID string) error {
	logger.LogInfo("DeleteAccount called for ID: %s", accountID)

	accounts, err := LoadAccounts()
	if err != nil {
		logger.LogError("Failed to load accounts: %v", err)
		return err
	}

	// Find the account being deleted for logging
	var deletedUsername string
	for _, acc := range accounts {
		if acc.ID == accountID {
			deletedUsername = acc.Username
			break
		}
	}

	// Remove from accounts list
	newAccounts := []Account{}
	for _, acc := range accounts {
		if acc.ID != accountID {
			newAccounts = append(newAccounts, acc)
		}
	}

	// Delete password from keychain
	if err := exec.Command("security", "delete-generic-password",
		"-s", keychainService,
		"-a", accountID).Run(); err != nil {
		logger.LogDebug("Keychain entry deletion returned: %v (may not exist)", err)
	} else {
		logger.LogDebug("Password removed from keychain for ID: %s", accountID)
	}

	if err := SaveAccounts(newAccounts); err != nil {
		logger.LogError("Failed to save accounts after deletion: %v", err)
		return err
	}

	logger.LogInfo("Account deleted successfully: %s (ID: %s)", deletedUsername, accountID)
	return nil
}

// GetAccount finds an account by ID
func GetAccount(accountID string) (*Account, error) {
	accounts, err := LoadAccounts()
	if err != nil {
		return nil, err
	}

	for _, acc := range accounts {
		if acc.ID == accountID {
			return &acc, nil
		}
	}

	return nil, fmt.Errorf("account not found")
}

// UpdateAccountLabel updates the label for an account
func UpdateAccountLabel(accountID, newLabel string) error {
	logger.LogInfo("UpdateAccountLabel called for ID: %s, new label: %s", accountID, newLabel)

	accounts, err := LoadAccounts()
	if err != nil {
		logger.LogError("Failed to load accounts: %v", err)
		return err
	}

	for i := range accounts {
		if accounts[i].ID == accountID {
			oldLabel := accounts[i].Label
			accounts[i].Label = newLabel
			if err := SaveAccounts(accounts); err != nil {
				logger.LogError("Failed to save accounts after label update: %v", err)
				return err
			}
			logger.LogInfo("Account label updated: %s -> %s (ID: %s)", oldLabel, newLabel, accountID)
			return nil
		}
	}

	logger.LogError("Account not found for ID: %s", accountID)
	return fmt.Errorf("account not found")
}
