package resource_monitor

import (
	"fmt"
	"insadem/multi_roblox_macos/internal/ps_darwin"
	"os/exec"
	"strconv"
	"strings"
)

// ProcessStats holds CPU and memory statistics for a process
type ProcessStats struct {
	PID        int
	CPUPercent float64
	MemoryMB   float64
}

// GetProcessStats gets CPU and memory usage for a specific PID
func GetProcessStats(pid int) (ProcessStats, error) {
	stats := ProcessStats{PID: pid}

	// Use ps command to get CPU and memory info
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "%cpu,%mem")
	output, err := cmd.Output()
	if err != nil {
		return stats, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return stats, fmt.Errorf("invalid ps output")
	}

	// Parse the second line (first is header)
	fields := strings.Fields(lines[1])
	if len(fields) >= 2 {
		if cpu, err := strconv.ParseFloat(fields[0], 64); err == nil {
			stats.CPUPercent = cpu
		}
		if mem, err := strconv.ParseFloat(fields[1], 64); err == nil {
			// Convert memory percentage to MB (approximate)
			stats.MemoryMB = mem * getSystemMemoryMB() / 100.0
		}
	}

	return stats, nil
}

// GetAllRobloxStats gets stats for all running Roblox instances
func GetAllRobloxStats() ([]ProcessStats, error) {
	processes, err := ps_darwin.Processes()
	if err != nil {
		return nil, err
	}

	var stats []ProcessStats
	for _, proc := range processes {
		if proc.Executable() == "RobloxPlayer" {
			if procStats, err := GetProcessStats(proc.Pid()); err == nil {
				stats = append(stats, procStats)
			}
		}
	}

	return stats, nil
}

// GetSystemStats returns overall system CPU and memory usage
func GetSystemStats() (cpuPercent float64, memoryUsedMB float64, memoryTotalMB float64, err error) {
	// Get total memory
	cmd := exec.Command("sysctl", "-n", "hw.memsize")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, err
	}
	memBytes, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	memoryTotalMB = float64(memBytes) / 1024 / 1024

	// Get used memory using vm_stat
	cmd = exec.Command("vm_stat")
	output, err = cmd.Output()
	if err != nil {
		return 0, 0, 0, err
	}

	pageSize := 4096.0 // macOS page size
	var pagesActive, pagesWired, pagesCompressed float64

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Pages active:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				val, _ := strconv.ParseFloat(strings.TrimSuffix(fields[2], "."), 64)
				pagesActive = val
			}
		} else if strings.Contains(line, "Pages wired down:") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				val, _ := strconv.ParseFloat(strings.TrimSuffix(fields[3], "."), 64)
				pagesWired = val
			}
		} else if strings.Contains(line, "Pages occupied by compressor:") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				val, _ := strconv.ParseFloat(strings.TrimSuffix(fields[4], "."), 64)
				pagesCompressed = val
			}
		}
	}

	memoryUsedMB = (pagesActive + pagesWired + pagesCompressed) * pageSize / 1024 / 1024

	// Get CPU usage using top
	cmd = exec.Command("top", "-l", "1", "-n", "0")
	output, err = cmd.Output()
	if err != nil {
		return 0, memoryUsedMB, memoryTotalMB, err
	}

	lines = strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "CPU usage:") {
			// Parse: "CPU usage: 5.71% user, 4.28% sys, 90.00% idle"
			parts := strings.Split(line, ",")
			if len(parts) >= 3 {
				idlePart := strings.TrimSpace(parts[2])
				idleStr := strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(idlePart, "")), "% idle")
				if idle, err := strconv.ParseFloat(idleStr, 64); err == nil {
					cpuPercent = 100.0 - idle
				}
			}
			break
		}
	}

	return cpuPercent, memoryUsedMB, memoryTotalMB, nil
}

// getSystemMemoryMB returns total system memory in MB
func getSystemMemoryMB() float64 {
	cmd := exec.Command("sysctl", "-n", "hw.memsize")
	output, err := cmd.Output()
	if err != nil {
		return 16384 // Default to 16GB if we can't determine
	}
	memBytes, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 16384
	}
	return float64(memBytes) / 1024 / 1024
}

// FormatMemory formats memory in MB to a human-readable string
func FormatMemory(mb float64) string {
	if mb >= 1024 {
		return fmt.Sprintf("%.1f GB", mb/1024)
	}
	return fmt.Sprintf("%.0f MB", mb)
}
