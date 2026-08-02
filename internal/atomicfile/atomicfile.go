// Package atomicfile provides crash-safe file persistence: data is written to
// a temporary file in the target directory and renamed over the destination,
// so readers never observe a truncated or partially-written file. Added after
// the 2026-08 audit found every config file (accounts.json, presets.json,
// labels.json, friends.json, instance_accounts.json) was written with a plain
// os.WriteFile — a crash mid-write corrupted the JSON and wiped the list on
// the next load.
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile atomically replaces the file at path with data. The temporary
// file is created in the same directory so the final os.Rename never crosses
// a filesystem boundary (rename is only atomic within a single volume).
func WriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("atomicfile: create temp: %w", err)
	}
	tmpPath := tmp.Name()

	// CreateTemp uses 0600; Chmod covers callers asking for something else.
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()        //nolint:errcheck — best-effort cleanup on error path
		os.Remove(tmpPath) //nolint:errcheck
		return fmt.Errorf("atomicfile: chmod temp: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()        //nolint:errcheck
		os.Remove(tmpPath) //nolint:errcheck
		return fmt.Errorf("atomicfile: write temp: %w", err)
	}
	// Sync before rename: without it a power loss can leave the rename durable
	// but the data blocks empty — the exact corruption this package prevents.
	if err := tmp.Sync(); err != nil {
		tmp.Close()        //nolint:errcheck
		os.Remove(tmpPath) //nolint:errcheck
		return fmt.Errorf("atomicfile: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath) //nolint:errcheck
		return fmt.Errorf("atomicfile: close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) //nolint:errcheck
		return fmt.Errorf("atomicfile: rename over %s: %w", filepath.Base(path), err)
	}
	return nil
}
