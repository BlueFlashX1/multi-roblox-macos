package instance_manager

import (
	"insadem/multi_roblox_macos/internal/label_manager"
	"insadem/multi_roblox_macos/internal/ps_darwin"
	"time"
)

// Instance represents a running Roblox instance
type Instance struct {
	PID       int
	StartTime time.Time
	Name      string
	Label     string
	Color     string
}

// GetRunningInstances returns a list of all running Roblox instances with labels
func GetRunningInstances() ([]Instance, error) {
	processes, err := ps_darwin.Processes()
	if err != nil {
		return nil, err
	}

	var instances []Instance
	var pids []int

	for _, proc := range processes {
		if proc.Executable() == "RobloxPlayer" {
			pid := proc.Pid()
			pids = append(pids, pid)

			// Get label if exists
			label, hasLabel := label_manager.GetLabel(pid)
			labelText := ""
			color := ""
			if hasLabel {
				labelText = label.Label
				color = label.Color
			}

			instances = append(instances, Instance{
				PID:       pid,
				StartTime: time.Now(), // TODO: Get actual start time from process
				Name:      proc.Executable(),
				Label:     labelText,
				Color:     color,
			})
		}
	}

	// Cleanup stale labels
	if len(pids) > 0 {
		label_manager.CleanupStaleLabels(pids)
	}

	return instances, nil
}

// GetInstanceCount returns the number of running Roblox instances
func GetInstanceCount() (int, error) {
	instances, err := GetRunningInstances()
	if err != nil {
		return 0, err
	}
	return len(instances), nil
}

// CloseInstance closes a specific Roblox instance by PID
func CloseInstance(pid int) error {
	// Remove label when closing
	label_manager.DeleteLabel(pid)
	return ps_darwin.KillProcess(pid)
}
