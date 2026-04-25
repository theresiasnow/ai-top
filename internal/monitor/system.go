package monitor

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemInfo provides system-wide metrics
type SystemInfo struct {
	CPUUsage    float64 // percentage 0-100
	CPUCores    int
	MemTotal    uint64
	MemUsed     uint64
	MemPercent  float32
	SwapTotal   uint64
	SwapUsed    uint64
	Uptime      uint64 // seconds
}

// GetSystemInfo returns current system metrics
func GetSystemInfo() (SystemInfo, error) {
	info := SystemInfo{}

	// CPU
	cpus, err := cpu.Counts(false)
	if err == nil {
		info.CPUCores = cpus
	}

	percents, err := cpu.Percent(0, false)
	if err == nil && len(percents) > 0 {
		info.CPUUsage = percents[0]
	}

	// Memory
	vmem, err := mem.VirtualMemory()
	if err == nil {
		info.MemTotal = vmem.Total
		info.MemUsed = vmem.Used
		info.MemPercent = float32(vmem.UsedPercent)
	}

	swap, err := mem.SwapMemory()
	if err == nil {
		info.SwapTotal = swap.Total
		info.SwapUsed = swap.Used
	}

	return info, nil
}

// GetProgressBar returns a visual bar for percentage (0-100)
func GetProgressBar(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(percent * float64(width) / 100)
	bar := "["

	for i := 0; i < width; i++ {
		if i < filled {
			if percent >= 80 {
				bar += "[red]█[-]"
			} else if percent >= 50 {
				bar += "[yellow]█[-]"
			} else {
				bar += "[green]█[-]"
			}
		} else {
			bar += "░"
		}
	}
	bar += "]"

	return bar
}
