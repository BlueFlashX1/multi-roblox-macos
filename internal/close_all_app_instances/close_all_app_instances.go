package close_all_app_instances

import (
	"insadem/multi_roblox_macos/internal/logger"
	"insadem/multi_roblox_macos/internal/ps_darwin"
)

// Close terminates every process whose executable name matches. It uses the
// same graceful SIGTERM→wait→SIGKILL sequence as single-instance close
// (ps_darwin.ForceKillProcess) instead of an immediate unconditional SIGKILL,
// and logs failures instead of silently discarding them.
func Close(name string) {
	processes, err := ps_darwin.Processes()
	if err != nil {
		logger.LogError("Close all: failed to list processes: %v", err)
		return
	}

	for _, process := range processes {
		if process.Executable() == name {
			if err := ps_darwin.ForceKillProcess(process.Pid()); err != nil {
				logger.LogError("Close all: failed to kill PID %d: %v", process.Pid(), err)
			}
		}
	}
}
