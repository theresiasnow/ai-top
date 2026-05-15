package monitor

import (
	"fmt"
	"sync"
	"time"
)

// SystemMetrics holds current system state
type SystemMetrics struct {
	OpenClaw      OpenClawStatus
	Ollama        OllamaStatus
	Omlx          OmlxStatus
	OllamaProcess *ProcessInfo
	OmlxProcess   *ProcessInfo
	CronJobs      []CronJob
	Processes     []ProcessInfo
	UpdatedAt     time.Time
	SysInfo       SystemInfo
	DiskUsagePct  float64
	OllamaLogs    []string
	RunningModels []RunningModel // models currently loaded in memory
}

type OpenClawStatus struct {
	Running bool
	Memory  uint64
	PID     int
	Uptime  time.Duration
}

type OllamaStatus struct {
	Running bool
	Models  []ModelInfo
	Memory  uint64
}

type ModelInfo struct {
	Name      string
	Size      string
	SizeBytes int64
	Modified  time.Time
	Loaded    bool
}

// RunningModel represents a model currently loaded in memory by Ollama.
type RunningModel struct {
	Name      string
	SizeVRAM  uint64    // bytes in GPU/unified memory
	SizeRAM   uint64    // bytes in CPU RAM (size - size_vram)
	ExpiresAt time.Time // when Ollama will unload it if idle
}

type CronJob struct {
	Name     string
	Schedule string
	LastRun  time.Time
	Status   string
	NextRun  time.Time
}

type ProcessInfo struct {
	PID         int
	Name        string
	User        string
	CPU         float64
	Memory      uint64
	MemoryPct   float32
	StartTime   time.Time
	CommandLine string
}

// Monitor tracks system metrics
type Monitor struct {
	mu          sync.RWMutex
	metrics     *SystemMetrics
	openClaw    *OpenClawDetector
	Ollama      *OllamaClient
	omlx        *OmlxClient
	ollamaPort  string
	refreshRate time.Duration
}

// NewMonitor creates a new monitor
func NewMonitor() *Monitor {
	return &Monitor{
		metrics: &SystemMetrics{
			UpdatedAt: time.Now(),
		},
		openClaw:    NewOpenClawDetector(3000),
		Ollama:      NewOllamaClient(""),
		omlx:        newOmlxClientForPlatform(),
		ollamaPort:  "11434",
		refreshRate: 2 * time.Second,
	}
}

func newOmlxClientForPlatform() *OmlxClient {
	if !SupportsOmlx() {
		return nil
	}
	return NewOmlxClient()
}

// GetMetrics returns current metrics (snapshot)
func (m *Monitor) GetMetrics() *SystemMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	snapshot := *m.metrics
	if m.metrics.Processes != nil {
		snapshot.Processes = make([]ProcessInfo, len(m.metrics.Processes))
		copy(snapshot.Processes, m.metrics.Processes)
	}
	if m.metrics.CronJobs != nil {
		snapshot.CronJobs = make([]CronJob, len(m.metrics.CronJobs))
		copy(snapshot.CronJobs, m.metrics.CronJobs)
	}
	if m.metrics.Ollama.Models != nil {
		snapshot.Ollama.Models = make([]ModelInfo, len(m.metrics.Ollama.Models))
		copy(snapshot.Ollama.Models, m.metrics.Ollama.Models)
	}
	if m.metrics.Omlx.Models != nil {
		snapshot.Omlx.Models = make([]OmlxModelInfo, len(m.metrics.Omlx.Models))
		copy(snapshot.Omlx.Models, m.metrics.Omlx.Models)
	}

	if m.metrics.OllamaProcess != nil {
		ollamaProcess := *m.metrics.OllamaProcess
		snapshot.OllamaProcess = &ollamaProcess
	}
	if m.metrics.OmlxProcess != nil {
		omlxProcess := *m.metrics.OmlxProcess
		snapshot.OmlxProcess = &omlxProcess
	}

	if m.metrics.OllamaLogs != nil {
		snapshot.OllamaLogs = make([]string, len(m.metrics.OllamaLogs))
		copy(snapshot.OllamaLogs, m.metrics.OllamaLogs)
	}

	if m.metrics.RunningModels != nil {
		snapshot.RunningModels = make([]RunningModel, len(m.metrics.RunningModels))
		copy(snapshot.RunningModels, m.metrics.RunningModels)
	}

	return &snapshot
}

// Refresh updates metrics from system
func (m *Monitor) Refresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// OpenClaw status
	status, _ := m.openClaw.GetStatus()
	m.metrics.OpenClaw = status

	// omlx status
	if m.omlx != nil {
		if omlxStatus, err := m.omlx.GetStatus(); err == nil {
			m.metrics.Omlx = omlxStatus
		} else {
			m.metrics.Omlx = OmlxStatus{}
		}
	} else {
		m.metrics.Omlx = OmlxStatus{}
		m.metrics.OmlxProcess = nil
	}

	// Ollama status
	running := m.Ollama.IsRunning()
	models := []ModelInfo{}
	var runningModels []RunningModel
	if running {
		if modelList, err := m.Ollama.GetModels(); err == nil {
			models = modelList
		}
		if rm, err := m.Ollama.GetRunningModels(); err == nil {
			runningModels = rm
		}
	}
	m.metrics.Ollama = OllamaStatus{
		Running: running,
		Models:  models,
	}
	m.metrics.RunningModels = runningModels

	// Processes — single enumeration for list, ollama process, and omlx process
	if procs, ollamaProc, omlxProc, err := m.GetAllProcesses(); err == nil {
		m.metrics.Processes = procs
		m.metrics.OllamaProcess = ollamaProc
		m.metrics.OmlxProcess = omlxProc
	}

	// Update timestamp
	m.metrics.UpdatedAt = time.Now()

	// System info + disk
	if sysInfo, err := GetSystemInfo(); err == nil {
		m.metrics.SysInfo = sysInfo
	}
	if pct, err := OllamaModelsDiskUsagePct(); err == nil {
		m.metrics.DiskUsagePct = pct
	}

	// Ollama logs (tail)
	m.metrics.OllamaLogs = ReadOllamaLogs(8)

	return nil
}

// StartAutoRefresh runs a background goroutine that refreshes metrics
func (m *Monitor) StartAutoRefresh() {
	go func() {
		ticker := time.NewTicker(m.refreshRate)
		defer ticker.Stop()

		for range ticker.C {
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Printf("Monitor refresh panic: %v\n", r)
					}
				}()
				if err := m.Refresh(); err != nil {
					fmt.Printf("Monitor refresh error: %v\n", err)
				}
			}()
		}
	}()
}

// GetRefreshRate returns the refresh interval
func (m *Monitor) GetRefreshRate() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.refreshRate
}

// SetRefreshRate changes the refresh interval
func (m *Monitor) SetRefreshRate(rate time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshRate = rate
}
