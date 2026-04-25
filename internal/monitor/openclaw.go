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

// IsRunning checks if OpenClaw HTTP server is responding
func (od *OpenClawDetector) IsRunning() bool {
	url := fmt.Sprintf("http://localhost:%d/health", od.port)
	resp, err := od.client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Any response is good, we just need connectivity
	return resp.StatusCode < 500
}

// GetStatus returns current OpenClaw status
func (od *OpenClawDetector) GetStatus() (OpenClawStatus, error) {
	status := OpenClawStatus{
		Running: od.IsRunning(),
	}

	// Try to find process info if running
	if status.Running {
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
