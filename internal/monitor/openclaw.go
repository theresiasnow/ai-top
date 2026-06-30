package monitor

import (
	"fmt"
	"net/http"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// OpenClawDetector checks OpenClaw service status
type OpenClawDetector struct {
	port   int
	client *http.Client
}

// NewOpenClawDetector creates a detector
func NewOpenClawDetector(port int) *OpenClawDetector {
	if port == 0 {
		port = 3000 // Default port
	}

	return &OpenClawDetector{
		port: port,
		client: &http.Client{
			Timeout: 1 * time.Second,
		},
	}
}

// isRunningHTTP checks only the OpenClaw HTTP health endpoint.
func (od *OpenClawDetector) isRunningHTTP() bool {
	url := fmt.Sprintf("http://localhost:%d/health", od.port)
	resp, err := od.client.Get(url)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode < 500 {
			return true
		}
	}
	return false
}

// IsRunning checks if OpenClaw is running, via HTTP health check first and a
// process-table sweep as a fallback.
func (od *OpenClawDetector) IsRunning() bool {
	if od.isRunningHTTP() {
		return true
	}

	// Fallback: check for OpenClaw processes in .openclaw directory
	processes, err := GetOpenClawProcesses()
	if err == nil && len(processes) > 0 {
		return true
	}

	return false
}

// GetStatus returns current OpenClaw status, taking its own process snapshot.
// Prefer GetStatusFromSnapshot when a snapshot already exists for this refresh.
func (od *OpenClawDetector) GetStatus() (OpenClawStatus, error) {
	snap, err := snapshotProcesses()
	if err != nil {
		return OpenClawStatus{}, err
	}
	return od.GetStatusFromSnapshot(snap)
}

// GetStatusFromSnapshot returns current OpenClaw status using a process snapshot
// already taken for this refresh, avoiding a redundant process-table sweep.
func (od *OpenClawDetector) GetStatusFromSnapshot(snap *processSnapshot) (OpenClawStatus, error) {
	status := OpenClawStatus{}

	// Try to find OpenClaw processes from the shared snapshot.
	if snap != nil {
		processes := snap.openClaw()
		if len(processes) > 0 {
			status.Running = true
			// Use first process for aggregate metrics
			status.PID = processes[0].PID
			status.Memory = processes[0].Memory
			status.Uptime = time.Since(processes[0].StartTime)
			return status, nil
		}
	}

	// Fallback to HTTP check
	status.Running = false
	if od.isRunningHTTP() {
		status.Running = true
		if pid, err := GetOpenClawPID(); err == nil {
			status.PID = pid

			if p, err := process.NewProcess(int32(pid)); err == nil {
				if createTime, err := p.CreateTime(); err == nil {
					status.Uptime = time.Duration(time.Now().UnixMilli()-createTime) * time.Millisecond
				}
				if memInfo, err := p.MemoryInfo(); err == nil {
					status.Memory = memInfo.RSS
				}
			}
		}
	}

	return status, nil
}
