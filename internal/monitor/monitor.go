package monitor

import (
	"fmt"
	"sync"
	"time"
)

// SystemMetrics holds current system state
type SystemMetrics struct {
	OpenClaw  OpenClawStatus
	Ollama    OllamaStatus
	CronJobs  []CronJob
	Processes []ProcessInfo
	UpdatedAt time.Time
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
	Modified  time.Time
	Loaded    bool
}

type CronJob struct {
	Name      string
	Schedule  string
	LastRun   time.Time
	Status    string
	NextRun   time.Time
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
	ollama      *OllamaClient
	ollamaPort  string
	refreshRate time.Duration
}

// NewMonitor creates a new monitor
func NewMonitor() *Monitor {
	return &Monitor{
		metrics:     &SystemMetrics{
			UpdatedAt: time.Now(),
		},
		openClaw:    NewOpenClawDetector(3000),
		ollama:      NewOllamaClient(""),
		ollamaPort:  "11434",
		refreshRate: 2 * time.Second,
	}
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

	return &snapshot
}

// Refresh updates metrics from system
func (m *Monitor) Refresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// OpenClaw status
	status, _ := m.openClaw.GetStatus()
	m.metrics.OpenClaw = status

	// Ollama status
	running := m.ollama.IsRunning()
	models := []ModelInfo{}
	if running {
		if modelList, err := m.ollama.GetModels(); err == nil {
			models = modelList
		}
	}
	m.metrics.Ollama = OllamaStatus{
		Running: running,
		Models:  models,
	}

	// Processes
	if processes, err := m.GetAllProcesses(); err == nil {
		m.metrics.Processes = processes
	}

	// Update timestamp
	m.metrics.UpdatedAt = time.Now()

	return nil
}

// StartAutoRefresh runs a background goroutine that refreshes metrics
func (m *Monitor) StartAutoRefresh() {
	go func() {
		ticker := time.NewTicker(m.refreshRate)
		defer ticker.Stop()

		for range ticker.C {
			if err := m.Refresh(); err != nil {
				fmt.Printf("Monitor refresh error: %v\n", err)
			}
		}
	}()
}

// GetRefreshRate returns the refresh interval
func (m *Monitor) GetRefreshRate() time.Duration {
	return m.refreshRate
}

// SetRefreshRate changes the refresh interval
func (m *Monitor) SetRefreshRate(rate time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshRate = rate
}
