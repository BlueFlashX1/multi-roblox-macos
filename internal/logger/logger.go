// Package logger provides structured logging for multi-roblox-macos.
//
// Security convention: NEVER format raw secret values (cookies, auth tickets,
// passwords, Keychain credentials) into log messages with %s or %v.
// Use LogSecret instead:
//
//	logger.LogSecret("cookie", cookieValue)
//	// emits: cookie(len=120, sha8=4a8c2b1f)
//
// This prevents live auth credentials from leaking into log files and
// crash reports. Violations of this convention allowed the protocolString
// auth-ticket leak that was fixed in the security audit (2026-05-16).
package logger

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var logFile *os.File

// InitLogger initializes the log file
func InitLogger() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	logDir := filepath.Join(home, "Library", "Logs", "multi_roblox_macos")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logPath := filepath.Join(logDir, "preset_launch.log")

	// Open log file in append mode
	logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600) // Secure permissions
	if err != nil {
		return err
	}

	Log("Logger initialized")
	return nil
}

// Log writes a message to the log file
func Log(format string, args ...interface{}) {
	if logFile == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] %s\n", timestamp, message)

	logFile.WriteString(logLine)
	logFile.Sync() // Ensure it's written immediately
}

// LogError writes an error to the log file
func LogError(format string, args ...interface{}) {
	Log("ERROR: "+format, args...)
}

// LogInfo writes an info message to the log file
func LogInfo(format string, args ...interface{}) {
	Log("INFO: "+format, args...)
}

// LogDebug writes a debug message to the log file
func LogDebug(format string, args ...interface{}) {
	Log("DEBUG: "+format, args...)
}

// LogSecret logs a secret value safely: name, byte-length, and first 4 bytes
// of SHA-256 only. Never log the raw value — it may appear in crash reports
// or be captured by log-scraping tools.
//
// Example output: "INFO: cookie(len=120, sha8=4a8c2b1f)"
func LogSecret(name string, value string) {
	sum := sha256.Sum256([]byte(value))
	LogInfo("%s(len=%d, sha8=%x)", name, len(value), sum[:4])
}

// Close closes the log file
func Close() {
	if logFile != nil {
		Log("Logger closing")
		logFile.Close()
	}
}

// GetLogPath returns the path to the log file
func GetLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Logs", "multi_roblox_macos", "preset_launch.log")
}
