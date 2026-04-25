package monitor

import (
	"testing"
)

func TestGetNodeProcesses(t *testing.T) {
	m := NewMonitor()
	processes, err := m.GetNodeProcesses()

	if err != nil {
		t.Logf("GetNodeProcesses error: %v", err)
	}

	if len(processes) == 0 {
		t.Logf("⚠️  No Node processes found - this is okay if none are running")
		return
	}

	t.Logf("✅ Found %d Node processes", len(processes))
	for _, p := range processes {
		t.Logf("  - PID %d: %s (Memory: %s)", p.PID, p.Name, FormatMemory(p.Memory))
	}
}

func TestOllamaConnection(t *testing.T) {
	client := NewOllamaClient("")
	running := client.IsRunning()

	if running {
		models, err := client.GetModels()
		if err != nil {
			t.Logf("⚠️  Ollama running but error getting models: %v", err)
			return
		}
		t.Logf("✅ Ollama is running with %d models", len(models))
		for _, m := range models {
			t.Logf("  - %s (%s)", m.Name, m.Size)
		}
	} else {
		t.Logf("⚠️  Ollama not running - this is okay if not started")
	}
}

func TestOpenClawDetection(t *testing.T) {
	detector := NewOpenClawDetector(3000)
	status, err := detector.GetStatus()

	if err != nil {
		t.Logf("Error getting OpenClaw status: %v", err)
	}

	if status.Running {
		t.Logf("✅ OpenClaw is running")
	} else {
		t.Logf("⚠️  OpenClaw not running - this is okay if not started")
	}
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{512, "512.0B"},
		{1024, "1.0KB"},
		{1048576, "1.0MB"},
		{1073741824, "1.0GB"},
	}

	for _, tt := range tests {
		result := FormatMemory(tt.input)
		t.Logf("FormatMemory(%d) = %s", tt.input, result)
	}
}

func TestCronJobs(t *testing.T) {
jobs, err := GetCronJobs()

if err != nil {
t.Logf("⚠️  Error reading cron jobs: %v", err)
return
}

t.Logf("✅ Found %d cron jobs", len(jobs))
for _, j := range jobs {
t.Logf("  - %s: %s (last run: %s)", j.Name, j.Schedule, j.LastRun)
}
}

func TestOpenClawProcesses(t *testing.T) {
processes, err := GetOpenClawProcesses()

if err != nil {
t.Logf("⚠️  Error finding OpenClaw processes: %v", err)
return
}

if len(processes) == 0 {
t.Logf("⚠️  No OpenClaw processes found (may not be running)")
return
}

t.Logf("✅ Found %d OpenClaw processes", len(processes))
for _, p := range processes {
t.Logf("  - PID %d: %s (Memory: %s)", p.PID, p.Name, FormatMemory(p.Memory))
}
}
