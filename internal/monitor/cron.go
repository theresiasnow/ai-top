package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CronJob represents a cron job from .openclaw/cron/jobs.json
type CronJobDetail struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	LastRun  string `json:"lastRun"`
	Status   string `json:"status"`
}

// PM2Process represents a PM2 managed process
type PM2Process struct {
	PID         int
	Name        string
	CPU         float64
	Memory      uint64
	Status      string
	Uptime      time.Duration
	Restarts    int
	LastRestart time.Time
}

// GetCronJobs reads OpenClaw cron jobs
func GetCronJobs() ([]CronJobDetail, error) {
	cronPath := filepath.Join(os.Getenv("HOME"), ".openclaw", "cron", "jobs.json")

	data, err := os.ReadFile(cronPath)
	if err != nil {
		return nil, fmt.Errorf("cron jobs not found: %w", err)
	}

	var jobs []CronJobDetail
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse cron jobs: %w", err)
	}

	return jobs, nil
}

// GetPM2Processes gets processes from PM2
func GetPM2Processes() ([]PM2Process, error) {
	processes, err := getProcessesByName([]string{"pm2", "node"})
	if err != nil {
		return nil, err
	}

	var pm2Procs []PM2Process
	for _, p := range processes {
		// Filter for PM2 processes (look for pm2 in command line)
		if len(p.CommandLine) > 0 {
			pm2Procs = append(pm2Procs, PM2Process{
				PID:    p.PID,
				Name:   p.Name,
				CPU:    p.CPU,
				Memory: p.Memory,
				Status: "running",
				Uptime: time.Since(p.StartTime),
			})
		}
	}

	return pm2Procs, nil
}

// GetOpenClawProcesses gets processes in .openclaw directory
func GetOpenClawProcesses() ([]ProcessInfo, error) {
	// Look for Node processes that belong to OpenClaw
	allProcesses, err := getProcessesByName([]string{"node"})
	if err != nil {
		return nil, err
	}

	// Filter for processes in .openclaw directories
	var openclawProcs []ProcessInfo
	openclawPath := filepath.Join(os.Getenv("HOME"), ".openclaw")

	for _, p := range allProcesses {
		// Check if command line contains .openclaw path
		if len(p.CommandLine) > 0 && (containsPath(p.CommandLine, openclawPath) ||
			containsPath(p.CommandLine, "openclaw")) {
			openclawProcs = append(openclawProcs, p)
		}
	}

	return openclawProcs, nil
}

// GetOllamaProcesses finds Ollama system processes
func (m *Monitor) GetOllamaSystemProcesses() ([]ProcessInfo, error) {
	processes, err := getProcessesByName([]string{"ollama"})
	if err != nil {
		return nil, err
	}
	return processes, nil
}

func containsPath(str, path string) bool {
	return len(str) > 0 && (contains(str, path) || contains(str, "openclaw"))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
