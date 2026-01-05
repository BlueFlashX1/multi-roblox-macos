package instance_manager

import (
	"insadem/multi_roblox_macos/internal/ps_darwin"
	"time"
)

// Instance represents a running Roblox instance
type Instance struct {
	PID       int
	StartTime time.Time
	Name      string
}

// GetRunningInstances returns a list of all running Roblox instances
func GetRunningInstances() ([]Instance, error) {
	processes, err := ps_darwin.Processes()
	if err != nil {
		return nil, err
	}

	var instances []Instance
	for _, proc := range processes {
		if proc.Executable() == "RobloxPlayer" {
			instances = append(instances, Instance{
				PID:       proc.Pid(),
				StartTime: time.Now(), // TODO: Get actual start time from process
				Name:      proc.Executable(),
			})
		}
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
	return ps_darwin.KillProcess(pid)
}
