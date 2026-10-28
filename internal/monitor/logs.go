package monitor

import (
	"bufio"
	"os"

	"github.com/shirou/gopsutil/v3/disk"
)

const defaultOllamaLogPath = "/opt/homebrew/var/log/ollama.log"

// OllamaLogPath returns the configured log path, falling back to the default.
func OllamaLogPath() string {
	if p := os.Getenv("OLLAMON_LOG_PATH"); p != "" {
		return p
	}
	return defaultOllamaLogPath
}

// ReadOllamaLogs returns the last n lines from the ollama log file.
func ReadOllamaLogs(n int) []string {
	path := OllamaLogPath()
	f, err := os.Open(path)
	if err != nil {
		return []string{"(log unavailable)"}
	}
	defer f.Close()

	ring := make([]string, n)
	idx, count := 0, 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ring[idx%n] = scanner.Text()
		idx++
		count++
	}
	if count == 0 {
		return []string{"(log is empty)"}
	}
	if count <= n {
		return ring[:count]
	}
	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = ring[(idx+i)%n]
	}
	return result
}

// OllamaModelsDiskUsage returns the used-percent and free bytes for the
// partition containing ~/.ollama/models (falls back to the home partition).
func OllamaModelsDiskUsage() (pct float64, free uint64, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, 0, err
	}
	for _, p := range []string{home + "/.ollama/models", home} {
		if u, err := disk.Usage(p); err == nil {
			return u.UsedPercent, u.Free, nil
		}
	}
	return 0, 0, nil
}

// DiskLevel classifies disk pressure. Percent alone misleads on large disks
// (90% of 4TB is still hundreds of GB), so absolute free space also counts.
type DiskLevel int

const (
	DiskOK DiskLevel = iota
	DiskHigh
	DiskCritical
)

const (
	diskCriticalFree = 10 << 30
	diskHighFree     = 25 << 30
)

func ClassifyDisk(pct float64, free uint64) DiskLevel {
	switch {
	case pct >= 90 || free < diskCriticalFree:
		return DiskCritical
	case pct >= 80 || free < diskHighFree:
		return DiskHigh
	}
	return DiskOK
}
