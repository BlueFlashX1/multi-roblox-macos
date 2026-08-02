// Package keychain wraps the macOS `security` CLI for storing secrets.
//
// Secrets are fed to `security -i` (interactive mode) on stdin instead of as
// a `-w <value>` argv element: command-line arguments are visible to every
// local process via `ps` during the exec window, stdin is not. Every store is
// round-trip verified, because interactive-mode quoting is under-documented —
// a store that silently mangled the secret would strand the account.
package keychain

import (
	"fmt"
	"os/exec"
	"strings"
)

// quoteToken escapes a value for security(1)'s interactive command tokenizer
// (backslash-escaped double quotes, wrapped in double quotes).
func quoteToken(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// StoreGenericPassword stores value under (service, account), replacing any
// existing entry. The secret travels via stdin, never argv.
func StoreGenericPassword(service, account, value string) error {
	if value == "" {
		return fmt.Errorf("keychain: refusing to store empty secret")
	}
	if strings.ContainsAny(value, "\n\r") {
		return fmt.Errorf("keychain: secret must not contain newlines")
	}

	// Delete any existing entry first (ignore "not found"): -U alone can fail
	// when the item exists with different attributes.
	exec.Command("security", "delete-generic-password",
		"-s", service, "-a", account).Run() //nolint:errcheck

	script := fmt.Sprintf("add-generic-password -s %s -a %s -w %s -U\n",
		quoteToken(service), quoteToken(account), quoteToken(value))
	cmd := exec.Command("security", "-i")
	cmd.Stdin = strings.NewReader(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("keychain: add-generic-password failed: %w (output: %s)", err, string(output))
	}

	// Round-trip verification: read the entry back and compare byte-for-byte.
	stored, err := GetGenericPassword(service, account)
	if err != nil {
		return fmt.Errorf("keychain: stored secret could not be read back: %w", err)
	}
	if stored != value {
		// Don't leave a corrupted entry behind.
		exec.Command("security", "delete-generic-password",
			"-s", service, "-a", account).Run() //nolint:errcheck
		return fmt.Errorf("keychain: stored secret failed round-trip verification (len %d stored vs %d expected)", len(stored), len(value))
	}

	return nil
}

// GetGenericPassword retrieves the secret stored under (service, account).
func GetGenericPassword(service, account string) (string, error) {
	output, err := exec.Command("security", "find-generic-password",
		"-s", service, "-a", account, "-w").Output()
	if err != nil {
		return "", fmt.Errorf("keychain: entry not found for service %q", service)
	}
	// find-generic-password -w appends exactly one newline; trim only that so
	// secrets with meaningful leading/trailing whitespace survive.
	return strings.TrimSuffix(string(output), "\n"), nil
}

