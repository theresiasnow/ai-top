package ui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tess/ai-top/internal/monitor"
)

var (
	// btop-inspired color palette
	colorCyan  = lipgloss.Color("#00C8FF")
	colorGreen = lipgloss.Color("#60FF60")
	colorRed   = lipgloss.Color("#FF4444")
	colorAmber = lipgloss.Color("#FFB300")
	colorBlue  = lipgloss.Color("#7AAFFF")
	colorText  = lipgloss.Color("#CCCCCC")
	colorDim   = lipgloss.Color("#445566")

	styleTitle       = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	styleTabActive   = lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(colorCyan).Padding(0, 1).Bold(true)
	styleTabInactive = lipgloss.NewStyle().Foreground(colorDim).Padding(0, 1)
	styleGood        = lipgloss.NewStyle().Foreground(colorGreen)
	styleBad         = lipgloss.NewStyle().Foreground(colorRed)
	styleWarn        = lipgloss.NewStyle().Foreground(colorAmber)
	styleDim         = lipgloss.NewStyle().Foreground(colorDim)
	styleColHead     = lipgloss.NewStyle().Foreground(colorBlue).Bold(true)
	styleText        = lipgloss.NewStyle().Foreground(colorText)
)

type Model struct {
	mon       *monitor.Monitor
	width     int
	height    int
	activeTab int
	sortBy    string
	paused    bool
}

func NewModel(mon *monitor.Monitor) Model {
	return Model{
		mon:       mon,
		activeTab: 0,
		sortBy:    "memory",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case " ":
			m.paused = !m.paused
		case "1":
			m.activeTab = 0
		case "2":
			m.activeTab = 1
		case "3":
			m.activeTab = 2
		case "4":
			m.activeTab = 3
		case "s":
			m.sortBy = "name"
		case "c":
			m.sortBy = "cpu"
		case "m":
			m.sortBy = "memory"
		case "tab":
			m.activeTab = (m.activeTab + 1) % 4
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + 4) % 4
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	metrics := m.mon.GetMetrics()

	var sb strings.Builder
	sb.WriteString(m.renderHeader(metrics))
	sb.WriteString("\n")
	sb.WriteString(m.renderMainPanel(metrics))
	sb.WriteString("\n")
	sb.WriteString(m.renderFooter())

	return sb.String()
}

// ── bar helpers ──────────────────────────────────────────────────────────────

// miniBar draws a filled/empty bar for a 0-100 percentage value.
func miniBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(math.Round(float64(width) * pct / 100.0))
	empty := width - filled

	c := colorGreen
	switch {
	case pct >= 80:
		c = colorRed
	case pct >= 50:
		c = colorAmber
	}
	return lipgloss.NewStyle().Foreground(c).Render(strings.Repeat("█", filled)) +
		styleDim.Render(strings.Repeat("░", empty))
}

// cpuBar renders an 8-wide bar + "XX.X%"  (15 visible chars).
func cpuBar(pct float64) string {
	c := colorGreen
	switch {
	case pct >= 80:
		c = colorRed
	case pct >= 50:
		c = colorAmber
	}
	return miniBar(pct, 8) + " " +
		lipgloss.NewStyle().Foreground(c).Render(fmt.Sprintf("%5.1f%%", pct))
}

// memBar renders an 8-wide bar + right-padded size string (17 visible chars).
func memBar(bytes uint64, pct float32) string {
	c := colorGreen
	switch {
	case pct >= 10:
		c = colorRed
	case pct >= 5:
		c = colorAmber
	}
	return miniBar(float64(pct), 8) + " " +
		lipgloss.NewStyle().Foreground(c).Render(fmt.Sprintf("%-8s", monitor.FormatMemory(bytes)))
}

// ── box drawing ──────────────────────────────────────────────────────────────

func ds(s string) string { return styleDim.Render(s) }

// boxTop builds the top border:  ╭─ [left] ──────────────── [right] ─╮
func boxTop(left, right string, totalWidth int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	if right == "" {
		fill := totalWidth - 3 - lw
		if fill < 0 {
			fill = 0
		}
		return ds("╭─") + left + ds(strings.Repeat("─", fill)) + ds("╮")
	}
	dashes := totalWidth - 4 - lw - rw
	if dashes < 0 {
		dashes = 0
	}
	return ds("╭─") + left + ds(strings.Repeat("─", dashes)) + right + ds("─╮")
}

// boxLine wraps content in │ … │, padding to totalWidth.
func boxLine(content string, totalWidth int) string {
	innerW := totalWidth - 4
	pad := innerW - lipgloss.Width(content)
	if pad < 0 {
		pad = 0
	}
	return ds("│") + " " + content + strings.Repeat(" ", pad) + " " + ds("│")
}

// boxDivider renders a full-width rule inside the box.
func boxDivider(totalWidth int) string {
	return ds("│") + " " + ds(strings.Repeat("─", totalWidth-4)) + " " + ds("│")
}

// boxBottom renders: ╰──────────────────────╯
func boxBottom(totalWidth int) string {
	return ds("╰" + strings.Repeat("─", totalWidth-2) + "╯")
}

// ── sections ─────────────────────────────────────────────────────────────────

func (m Model) renderHeader(metrics *monitor.SystemMetrics) string {
	w := m.width

	// line 1: title + clock
	title := styleTitle.Render("⟨ ai-top ⟩")
	subtitle := styleDim.Render("  AI Development Monitor")
	timeStr := styleDim.Render(time.Now().Format("Mon 15:04:05"))
	left1 := title + subtitle
	fill1 := w - lipgloss.Width(left1) - lipgloss.Width(timeStr) - 2
	if fill1 < 1 {
		fill1 = 1
	}
	line1 := "  " + left1 + strings.Repeat(" ", fill1) + timeStr

	// line 2: service status
	var ocPart string
	if metrics.OpenClaw.Running {
		ocPart = styleGood.Render("● OpenClaw") +
			styleDim.Render(fmt.Sprintf("  pid %d · %s · up %s",
				metrics.OpenClaw.PID,
				monitor.FormatMemory(metrics.OpenClaw.Memory),
				formatUptime(metrics.OpenClaw.Uptime)))
	} else {
		ocPart = styleBad.Render("● OpenClaw offline")
	}

	var olPart string
	if metrics.Ollama.Running {
		olPart = styleGood.Render("● Ollama") +
			styleDim.Render(fmt.Sprintf("  %d models", len(metrics.Ollama.Models)))
	} else {
		olPart = styleBad.Render("● Ollama offline")
	}

	pauseStr := ""
	if m.paused {
		pauseStr = "    " + styleWarn.Render("⏸ paused")
	}
	line2 := "  " + ocPart + "    " + olPart + pauseStr

	return line1 + "\n" + line2
}

func formatUptime(d time.Duration) string {
	h := int(d.Hours())
	mn := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, mn)
	}
	return fmt.Sprintf("%dm", mn)
}

func (m Model) renderMainPanel(metrics *monitor.SystemMetrics) string {
	w := m.width
	if w < 40 {
		w = 40
	}

	// Tab bar embedded in the top border
	tabs := []string{"Node.js", "Ollama", "OpenClaw", "Cron"}
	nums := []string{"1", "2", "3", "4"}
	var tabBar strings.Builder
	for i, name := range tabs {
		if i > 0 {
			tabBar.WriteString(" ")
		}
		label := nums[i] + ":" + name
		if i == m.activeTab {
			tabBar.WriteString(styleTabActive.Render(label))
		} else {
			tabBar.WriteString(styleTabInactive.Render(label))
		}
	}

	// Sort indicator in the top-right corner for the process tab
	sortIndicator := ""
	if m.activeTab == 0 {
		switch m.sortBy {
		case "cpu":
			sortIndicator = styleDim.Render(" sort:cpu ")
		case "name":
			sortIndicator = styleDim.Render(" sort:name ")
		default:
			sortIndicator = styleDim.Render(" sort:mem ")
		}
	}

	// Content height accounting for header(2) + newline(1) + borders(2) + divider(1) + footer(1) + gap(1)
	contentHeight := m.height - 8
	if contentHeight < 3 {
		contentHeight = 3
	}

	var lines []string
	switch m.activeTab {
	case 0:
		lines = m.processLines(metrics.Processes, contentHeight)
	case 1:
		lines = m.ollamaLines(metrics.Ollama, contentHeight)
	case 2:
		lines = m.openClawLines(metrics.OpenClaw)
	case 3:
		lines = m.cronLines(contentHeight)
	}

	var sb strings.Builder
	sb.WriteString(boxTop(tabBar.String(), sortIndicator, w))
	sb.WriteString("\n")
	sb.WriteString(boxDivider(w))
	sb.WriteString("\n")

	for i, line := range lines {
		if i >= contentHeight {
			break
		}
		sb.WriteString(boxLine(line, w))
		sb.WriteString("\n")
	}
	for i := len(lines); i < contentHeight; i++ {
		sb.WriteString(boxLine("", w))
		sb.WriteString("\n")
	}

	sb.WriteString(boxBottom(w))
	return sb.String()
}

// ── tab content ───────────────────────────────────────────────────────────────

func (m Model) processLines(processes []monitor.ProcessInfo, maxLines int) []string {
	// Column widths: cpuW=15 (bar8 + space + %5.1f%% 6), memW=17 (bar8 + space + %-8s)
	const (
		pidW    = 7
		userW   = 10
		cpuW    = 15
		memW    = 17
		spacers = 8 // 4 × "  "
	)
	innerW := m.width - 4
	nameW := innerW - pidW - userW - cpuW - memW - spacers
	if nameW < 12 {
		nameW = 12
	}

	hdr := styleColHead.Render(fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s",
		pidW, "PID", nameW, "NAME", userW, "USER", cpuW, "CPU", memW, "MEMORY"))
	divider := styleDim.Render(strings.Repeat("─", lipgloss.Width(hdr)))
	lines := []string{hdr, divider}

	if len(processes) == 0 {
		return append(lines, styleDim.Render("  no processes found"))
	}

	sorted := make([]monitor.ProcessInfo, len(processes))
	copy(sorted, processes)
	switch m.sortBy {
	case "cpu":
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].CPU > sorted[j].CPU })
	case "name":
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	default:
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Memory > sorted[j].Memory })
	}

	available := maxLines - 2
	if available < 1 {
		available = 1
	}
	if len(sorted) > available {
		sorted = sorted[:available]
	}

	for _, p := range sorted {
		name := truncate(p.Name, nameW)
		user := truncate(p.User, userW)

		pidStr := lipgloss.NewStyle().Width(pidW).Render(fmt.Sprintf("%d", p.PID))
		nameStr := lipgloss.NewStyle().Width(nameW).Foreground(colorText).Render(name)
		userStr := lipgloss.NewStyle().Width(userW).Foreground(colorDim).Render(user)

		lines = append(lines, pidStr+"  "+nameStr+"  "+userStr+"  "+cpuBar(p.CPU)+"  "+memBar(p.Memory, p.MemoryPct))
	}

	return lines
}

func (m Model) ollamaLines(status monitor.OllamaStatus, maxLines int) []string {
	if !status.Running {
		return []string{"", "  " + styleBad.Render("● Ollama is offline")}
	}

	const (
		sizeW     = 10
		priorityW = 14
		statusW   = 12
		spacers   = 6
	)
	innerW := m.width - 4
	nameW := innerW - sizeW - priorityW - statusW - spacers
	if nameW < 15 {
		nameW = 15
	}

	hdr := styleColHead.Render(fmt.Sprintf("%-*s  %-*s  %-*s  %-*s",
		nameW, "MODEL", sizeW, "SIZE", priorityW, "PRIORITY", statusW, "STATUS"))
	divider := styleDim.Render(strings.Repeat("─", lipgloss.Width(hdr)))
	lines := []string{hdr, divider}

	if len(status.Models) == 0 {
		return append(lines, styleDim.Render("  no models available"))
	}

	available := maxLines - 2
	if available < 1 {
		available = 1
	}

	for i, model := range status.Models {
		if i >= available {
			break
		}

		var prioLabel string
		switch {
		case isHotModel(model.Name):
			prioLabel = styleGood.Render("🔥 hot")
		case isWarmModel(model.Name):
			prioLabel = styleWarn.Render("🌡 warm")
		default:
			prioLabel = styleDim.Render("❄ cold")
		}

		nameStr := lipgloss.NewStyle().Width(nameW).Foreground(colorText).Render(truncate(model.Name, nameW))
		sizeStr := lipgloss.NewStyle().Width(sizeW).Foreground(colorDim).Render(model.Size)
		prioStr := lipgloss.NewStyle().Width(priorityW).Render(prioLabel)
		statStr := styleGood.Render("● loaded")

		lines = append(lines, nameStr+"  "+sizeStr+"  "+prioStr+"  "+statStr)
	}

	return lines
}

func isHotModel(modelName string) bool {
	return map[string]bool{
		"qwen2.5-coder:14b": true,
		"gemma4:latest":     true,
	}[modelName]
}

func isWarmModel(modelName string) bool {
	return map[string]bool{"qwen3:8b": true}[modelName]
}

func (m Model) openClawLines(s monitor.OpenClawStatus) []string {
	if !s.Running {
		return []string{"", "  " + styleBad.Render("● OpenClaw is offline")}
	}

	kv := func(k, v string) string {
		return "  " + styleColHead.Render(fmt.Sprintf("%-10s", k)) + styleText.Render(v)
	}

	return []string{
		"",
		"  " + styleGood.Render("● OpenClaw online"),
		"",
		kv("Status", "running"),
		kv("PID", fmt.Sprintf("%d", s.PID)),
		kv("Memory", monitor.FormatMemory(s.Memory)),
		kv("Uptime", formatUptime(s.Uptime)),
	}
}

func (m Model) cronLines(maxLines int) []string {
	cronJobs, err := monitor.GetCronJobs()
	if err != nil || len(cronJobs) == 0 {
		return []string{styleDim.Render("  no cron jobs defined")}
	}

	const (
		scheduleW = 22
		statusW   = 14
		spacers   = 4
	)
	innerW := m.width - 4
	nameW := innerW - scheduleW - statusW - spacers
	if nameW < 15 {
		nameW = 15
	}

	hdr := styleColHead.Render(fmt.Sprintf("%-*s  %-*s  %-*s",
		nameW, "JOB NAME", scheduleW, "SCHEDULE", statusW, "STATUS"))
	divider := styleDim.Render(strings.Repeat("─", lipgloss.Width(hdr)))
	lines := []string{hdr, divider}

	available := maxLines - 2
	if available < 1 {
		available = 1
	}
	if len(cronJobs) > available {
		cronJobs = cronJobs[:available]
	}

	for _, job := range cronJobs {
		var statusStr string
		switch job.Status {
		case "success", "enabled":
			statusStr = styleGood.Render("✓ " + job.Status)
		case "running":
			statusStr = styleWarn.Render("▶ " + job.Status)
		default:
			statusStr = styleBad.Render("✗ " + job.Status)
		}

		nameStr := lipgloss.NewStyle().Width(nameW).Foreground(colorText).Render(truncate(job.Name, nameW))
		schedStr := lipgloss.NewStyle().Width(scheduleW).Foreground(colorDim).Render(truncate(job.Schedule, scheduleW))
		lines = append(lines, nameStr+"  "+schedStr+"  "+statusStr)
	}

	return lines
}

func (m Model) renderFooter() string {
	type binding struct{ key, desc string }
	bindings := []binding{
		{"Tab/1-4", "switch"},
		{"m", "sort mem"},
		{"c", "sort cpu"},
		{"s", "sort name"},
		{"Space", "pause"},
		{"q", "quit"},
	}

	sep := styleDim.Render("  │  ")
	var parts []string
	for _, b := range bindings {
		parts = append(parts, styleTitle.Render(b.key)+styleDim.Render(":"+b.desc))
	}
	return "  " + strings.Join(parts, sep)
}

// truncate clips s to at most n visible characters, appending "…" if clipped.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return s[:n-1] + "…"
}
