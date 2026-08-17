package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/tess/ai-top/internal/monitor"
)

// formatContext renders a context window as 32K / 2048, or "—" when the backend
// does not report one (omlx does not).
func formatContext(n int) string {
	if n <= 0 {
		return emptyCell
	}
	if n >= 1024 && n%1024 == 0 {
		return fmt.Sprintf("%dK", n/1024)
	}
	return fmt.Sprintf("%d", n)
}

// formatTTL renders time until an idle model is unloaded.
func formatTTL(model monitor.LoadedModel, now time.Time) string {
	d, ok := model.TTL(now)
	if !ok {
		return emptyCell
	}
	if d == 0 {
		return "expiring"
	}
	if d >= time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	if d >= time.Minute {
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

// formatOwner renders the process holding a model as "user/pid".
func formatOwner(model monitor.LoadedModel) string {
	if model.OwnerPID == 0 {
		return emptyCell
	}
	if model.OwnerUser == "" {
		return fmt.Sprintf("%d", model.OwnerPID)
	}
	return fmt.Sprintf("%s/%d", model.OwnerUser, model.OwnerPID)
}

// orDash renders an optional string, falling back to the empty-cell marker.
func orDash(s string) string {
	if s == "" {
		return emptyCell
	}
	return s
}

func (m Model) renderLoadedModelsPanel(metrics *monitor.SystemMetrics, contentHeight int) string {
	w := max(40, m.width)
	innerW := w - 4

	models := monitor.CollectLoadedModels(metrics)

	var totalVRAM uint64
	for _, lm := range models {
		totalVRAM += lm.SizeVRAM
	}

	indicator := styleDim.Render(fmt.Sprintf(" %d loaded · %s ",
		len(models), monitor.FormatMemory(totalVRAM)))

	var sb strings.Builder
	sb.WriteString(boxTop(styleTitle.Render(" Loaded Models "), indicator, w))
	sb.WriteString("\n")

	for _, line := range m.loadedModelLines(models, innerW) {
		if contentHeight <= 0 {
			break
		}
		sb.WriteString(boxLine(line, w))
		sb.WriteString("\n")
		contentHeight--
	}

	sb.WriteString(boxBottom(w))
	return sb.String()
}

// loadedModelLines builds the merged model table body.
func (m Model) loadedModelLines(models []monitor.LoadedModel, innerW int) []string {
	if len(models) == 0 {
		return []string{styleDim.Render("no models currently loaded")}
	}

	now := time.Now()
	lines := make([]string, 0, len(models)+1)

	header := fmt.Sprintf("%-22s %-8s %8s %8s %6s %8s %-12s %8s",
		"MODEL", "BACKEND", "VRAM", "RAM", "CTX", "QUANT", "OWNER", "TTL")
	lines = append(lines, styleDim.Render(truncate(header, innerW)))

	for _, lm := range models {
		ram := emptyCell
		if lm.SizeRAM > 0 {
			ram = monitor.FormatMemory(lm.SizeRAM)
		}

		line := fmt.Sprintf("%-22s %-8s %8s %8s %6s %8s %-12s %8s",
			truncate(lm.Name, 22),
			truncate(lm.Backend, 8),
			monitor.FormatMemory(lm.SizeVRAM),
			ram,
			formatContext(lm.ContextLength),
			orDash(lm.Quantization),
			truncate(formatOwner(lm), 12),
			formatTTL(lm, now),
		)
		lines = append(lines, truncate(line, innerW))
	}

	return lines
}
