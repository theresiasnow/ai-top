package monitor

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// GetNodeProcesses finds all Node.js processes
func (m *Monitor) GetNodeProcesses() ([]ProcessInfo, error) {
	return getProcessesByName([]string{"node", "nodejs"})
}

// GetOllamaProcesses finds Ollama processes
func (m *Monitor) GetOllamaProcesses() ([]ProcessInfo, error) {
	return getProcessesByName([]string{"ollama"})
}

// GetAllProcesses returns all monitored processes
func (m *Monitor) GetAllProcesses() ([]ProcessInfo, error) {
	var all []ProcessInfo

	node, err := m.GetNodeProcesses()
	if err == nil {
		all = append(all, node...)
	}

	ollama, err := m.GetOllamaProcesses()
	if err == nil {
		all = append(all, ollama...)
	}

	return all, nil
}

// getProcessesByName finds processes by executable name
func getProcessesByName(names []string) ([]ProcessInfo, error) {
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	var results []ProcessInfo

	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}

		// Check exact match and basename match
		if nameMap[name] || nameMap[strings.TrimSuffix(name, ".exe")] {
			info, err := getProcessInfo(p)
			if err == nil {
				results = append(results, info)
			}
		}
	}

	return results, nil
}

// getProcessInfo extracts detailed info from a process
func getProcessInfo(p *process.Process) (ProcessInfo, error) {
	name, _ := p.Name()
	cmdline, _ := p.Cmdline()
	createTime, _ := p.CreateTime()
	memInfo, _ := p.MemoryInfo()
	cpuPercent, _ := p.CPUPercent()

	// Get username
	username := "?"
	if u, err := p.Username(); err == nil {
		username = u
	}

	return ProcessInfo{
		PID:         int(p.Pid),
		Name:        name,
		User:        username,
		CPU:         cpuPercent,
		Memory:      memInfo.RSS,
		MemoryPct:   0, // Will calculate later if needed
		StartTime:   time.UnixMilli(createTime),
		CommandLine: cmdline,
	}, nil
}

// FormatMemory converts bytes to human-readable format
func FormatMemory(bytes uint64) string {
	units := []string{"B", "KB", "MB", "GB"}
	value := float64(bytes)

	for _, unit := range units {
		if value < 1024 {
			return fmt.Sprintf("%.1f%s", value, unit)
		}
		value /= 1024
	}

	return fmt.Sprintf("%.1f%s", value, "TB")
}

// GetProcessUptime returns uptime as a formatted string
func GetProcessUptime(startTime time.Time) string {
	uptime := time.Since(startTime)
	
	hours := int(uptime.Hours())
	mins := int(uptime.Minutes()) % 60
	secs := int(uptime.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
	if mins > 0 {
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

// GetOpenClawPID attempts to find OpenClaw process
func GetOpenClawPID() (int, error) {
	// Try to find by process name
	processes, err := process.Processes()
	if err != nil {
		return 0, err
	}

	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}

		if strings.Contains(name, "openclaw") || strings.Contains(name, "node") {
			cmdline, err := p.Cmdline()
			if err == nil && strings.Contains(cmdline, "openclaw") {
				return int(p.Pid), nil
			}
		}
	}

	return 0, fmt.Errorf("openclaw process not found")
}

// CheckOpenClawPort checks if OpenClaw is listening on expected port
func CheckOpenClawPort(port int) bool {
	cmd := exec.Command("lsof", "-Pi", fmt.Sprintf(":%d", port), "-sTCP:LISTEN", "-t")
	err := cmd.Run()
	return err == nil
}
