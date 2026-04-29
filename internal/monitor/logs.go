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

// OllamaModelsDiskUsagePct returns the used-percent for the partition
// containing ~/.ollama/models (falls back to the home partition).
func OllamaModelsDiskUsagePct() (float64, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, err
	}
	for _, p := range []string{home + "/.ollama/models", home} {
		if u, err := disk.Usage(p); err == nil {
			return u.UsedPercent, nil
		}
	}
	return 0, nil
}
