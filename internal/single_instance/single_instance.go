// Package single_instance prevents two copies of the app from running at
// once. Without the guard, concurrent instances raced load→modify→save cycles
// on the shared JSON config files (accounts, presets, labels), silently
// discarding each other's writes.
package single_instance

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// lockFile is held for the process lifetime so the flock persists; the kernel
// releases it automatically when the process exits, so a crash never leaves a
// stale lock behind (unlike a PID file).
var lockFile *os.File

// Acquire takes an exclusive advisory lock on a lockfile in the app's config
// directory. Returns an error if another instance already holds it.
func Acquire() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("single_instance: failed to get home dir: %w", err)
	}
	dir := filepath.Join(home, "Library", "Application Support", "multi_roblox_macos")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("single_instance: failed to create config dir: %w", err)
	}

	f, err := os.OpenFile(filepath.Join(dir, "app.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("single_instance: failed to open lockfile: %w", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close() //nolint:errcheck — lock not acquired, fd is useless
		return fmt.Errorf("another instance of Multi Roblox Manager is already running")
	}

	lockFile = f
	return nil
}
