package monitor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// watchedProcessNames returns the set of process names ai-top cares about.
// Only these processes pay the cost of the expensive per-process detail lookup
// (Cmdline/MemoryInfo/Username) — everything else on the system is skipped after
// a single cheap Name() read. "node"/"nodejs" are included because OpenClaw runs
// as a node process and is matched by command line.
func watchedProcessNames() map[string]bool {
	watched := map[string]bool{
		"node": true, "nodejs": true, "ollama": true,
	}
	if SupportsOmlx() {
		watched["omlx"] = true
	}
	return watched
}

// processSnapshot is the result of enumerating the system process table exactly
// once. It holds detailed info only for the watched processes (see
// watchedProcessNames), so callers within a single refresh can share one
// enumeration instead of each sweeping the full table independently.
type processSnapshot struct {
	watched []ProcessInfo
}

// snapshotProcesses enumerates the process table a single time. It reads the
// cheap Name() for every PID but only computes full ProcessInfo (which calls the
// expensive Cmdline() syscall) for processes whose name is watched. This is the
// hot path that runs every refresh; keep it lean.
func snapshotProcesses() (*processSnapshot, error) {
	watched := watchedProcessNames()

	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	snap := &processSnapshot{}
	for _, p := range procs {
		name, nameErr := p.Name()
		if nameErr != nil {
			continue
		}
		base := strings.TrimSuffix(name, ".exe")
		if !watched[name] && !watched[base] {
			continue
		}
		// Only node processes can be OpenClaw, and the command line is used
		// solely for that match — skip the costly Cmdline() syscall elsewhere.
		needCmdline := name == "node" || base == "node" || name == "nodejs" || base == "nodejs"
		info, infoErr := getProcessInfoWithOpts(p, needCmdline)
		if infoErr != nil {
			continue
		}
		snap.watched = append(snap.watched, info)
	}
	return snap, nil
}

// monitored splits the watched processes into the full list plus the first
// ollama and omlx processes, matching the previous GetAllProcesses contract.
func (s *processSnapshot) monitored() (all []ProcessInfo, ollamaProc *ProcessInfo, omlxProc *ProcessInfo) {
	for _, info := range s.watched {
		base := strings.TrimSuffix(info.Name, ".exe")
		all = append(all, info)
		if ollamaProc == nil && (info.Name == "ollama" || base == "ollama") {
			cp := info
			ollamaProc = &cp
		}
		if SupportsOmlx() && omlxProc == nil && (info.Name == "omlx" || base == "omlx") {
			cp := info
			omlxProc = &cp
		}
	}
	return all, ollamaProc, omlxProc
}

// openClaw returns the watched processes whose command line references OpenClaw.
func (s *processSnapshot) openClaw() []ProcessInfo {
	home, _ := os.UserHomeDir()
	openclawPath := filepath.Join(home, ".openclaw")

	var results []ProcessInfo
	for _, info := range s.watched {
		if strings.Contains(info.CommandLine, openclawPath) || strings.Contains(info.CommandLine, "openclaw") {
			results = append(results, info)
		}
	}
	return results
}

// GetAllProcesses returns all monitored processes, the first ollama process,
// and the first omlx process, enumerating system processes exactly once.
func (m *Monitor) GetAllProcesses() (all []ProcessInfo, ollamaProc *ProcessInfo, omlxProc *ProcessInfo, err error) {
	snap, err := snapshotProcesses()
	if err != nil {
		return nil, nil, nil, err
	}
	all, ollamaProc, omlxProc = snap.monitored()
	return all, ollamaProc, omlxProc, nil
}

// getProcessInfo extracts detailed info from a process, including the command
// line. Cmdline() is the most expensive per-process syscall on macOS; prefer
// getProcessInfoWithOpts(p, false) when the command line is not needed.
func getProcessInfo(p *process.Process) (ProcessInfo, error) {
	return getProcessInfoWithOpts(p, true)
}

// getProcessInfoWithOpts extracts detailed info from a process. When
// withCmdline is false the command line is left empty, skipping the costly
// Cmdline() syscall — only node processes need it (to identify OpenClaw); the
// command line is never displayed.
func getProcessInfoWithOpts(p *process.Process, withCmdline bool) (ProcessInfo, error) {
	name, _ := p.Name()
	createTime, _ := p.CreateTime()
	memInfo, _ := p.MemoryInfo()
	cpuPercent, _ := p.CPUPercent()
	memPercent, _ := p.MemoryPercent()

	var cmdline string
	if withCmdline {
		cmdline, _ = p.Cmdline()
	}

	// Get username
	username := "?"
	if u, err := p.Username(); err == nil {
		username = u
	}

	var memRSS uint64
	if memInfo != nil {
		memRSS = memInfo.RSS
	}

	return ProcessInfo{
		PID:         int(p.Pid),
		Name:        name,
		User:        username,
		CPU:         cpuPercent,
		Memory:      memRSS,
		MemoryPct:   memPercent,
		StartTime:   time.UnixMilli(createTime),
		CommandLine: cmdline,
	}, nil
}

// ProcessLabel derives a human-friendly name for a process. For node processes
// every row's Name() is just "node"; the useful identity lives in the command
// line (the script or bin being run), which we already fetch for node. Returns
// the script/bin basename (".js"/".cjs"/".mjs" stripped) when it can be found,
// otherwise falls back to the process name.
func ProcessLabel(p ProcessInfo) string {
	base := strings.TrimSuffix(p.Name, ".exe")
	if base != "node" && base != "nodejs" {
		return p.Name
	}
	if label := scriptFromCmdline(p.CommandLine); label != "" {
		return label
	}
	return p.Name
}

// scriptFromCmdline picks the script/bin a node invocation is running: the first
// argument after the node executable that isn't a flag. Returns its basename with
// a JS extension stripped, or "" if none is found.
func scriptFromCmdline(cmdline string) string {
	fields := strings.Fields(cmdline)
	if len(fields) == 0 {
		return ""
	}
	// fields[0] is the node executable itself; scan the rest for the first
	// non-flag token (skips "--liftoff-only", "--type=utility", etc.).
	for _, arg := range fields[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		name := filepath.Base(arg)
		for _, ext := range []string{".js", ".cjs", ".mjs"} {
			name = strings.TrimSuffix(name, ext)
		}
		if name != "" {
			return name
		}
	}
	return ""
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

// GetOpenClawProcesses finds OpenClaw-related processes. It enumerates the
// process table once and only inspects the command line of watched processes
// (OpenClaw runs as node), avoiding a Cmdline() syscall on every PID on the
// system every refresh — the original cause of ai-top's runaway CPU use.
func GetOpenClawProcesses() ([]ProcessInfo, error) {
	snap, err := snapshotProcesses()
	if err != nil {
		return nil, err
	}
	return snap.openClaw(), nil
}
