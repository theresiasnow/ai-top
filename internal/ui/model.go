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

	styleTitle   = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	styleGood    = lipgloss.NewStyle().Foreground(colorGreen)
	styleBad     = lipgloss.NewStyle().Foreground(colorRed)
	styleWarn    = lipgloss.NewStyle().Foreground(colorAmber)
	styleDim     = lipgloss.NewStyle().Foreground(colorDim)
	styleColHead = lipgloss.NewStyle().Foreground(colorBlue).Bold(true)
	styleText    = lipgloss.NewStyle().Foreground(colorText)
)

type Model struct {
	mon    *monitor.Monitor
	width  int
	height int
	list   SelectableList
	sortBy string
	paused bool
	errMsg string
}

func NewModel(mon *monitor.Monitor) Model {
	return Model{
		mon:    mon,
		sortBy: "memory",
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
		m.syncList()
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case " ":
			m.paused = !m.paused
		case "up", "down":
			m.list.HandleKey(msg.String())
		case "k", "r":
			action := m.list.HandleKey(msg.String())
			m.dispatchAction(action)
		case "s":
			m.sortBy = "name"
		case "c":
			m.sortBy = "cpu"
		case "m":
			m.sortBy = "memory"
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncList()

	case tickMsg:
		m.syncList()
		return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}

	return m, nil
}

func (m *Model) dispatchAction(action Action) {
	if action == ActionNone {
		return
	}

	item := m.list.SelectedItem()
	var err error
	switch action {
	case ActionKill:
		err = monitor.KillProcess(item.PID)
	case ActionRestart:
		err = monitor.RestartProcess(item.PID)
	case ActionUnload:
		err = monitor.UnloadOllamaModel(m.mon.Ollama, item.ModelName)
	}
	if err != nil {
		m.errMsg = err.Error()
		return
	}
	m.errMsg = ""
	if err := m.mon.Refresh(); err != nil {
		m.errMsg = err.Error()
	}
	m.syncList()
}

func (m *Model) syncList() {
	if m.mon == nil {
		return
	}
	m.list.SetItems(m.buildListItems(m.mon.GetMetrics()))
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	if m.mon == nil {
		return "No monitor"
	}

	metrics := m.mon.GetMetrics()

	// Layout constants (terminal lines):
	//   status bar:    4   separator: 1
	//   insights:      insightBoxRows+2   separator: 1  (optional)
	//   process panel: contentHeight+2   separator: 1
	//   footer:        1
	const fixedOverhead = 4 + 1 + 2 + 1 + 1 // 9  (no insights)
	const insightOverhead = insightBoxRows + 2 + 1 // 9
	const minProcRows = 4

	showInsights := m.height >= fixedOverhead+insightOverhead+minProcRows
	overhead := fixedOverhead
	if showInsights {
		overhead += insightOverhead
	}
	contentHeight := m.height - overhead
	if contentHeight < minProcRows {
		contentHeight = minProcRows
	}

	var sb strings.Builder
	sb.WriteString(m.renderStatusBar(metrics))
	sb.WriteString("\n")
	if showInsights {
		sb.WriteString(m.renderInsightsAndLogs(metrics))
		sb.WriteString("\n")
	}
	sb.WriteString(m.renderProcessPanel(metrics, contentHeight))
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
	content = truncate(content, innerW)
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

func (m Model) renderStatusBar(metrics *monitor.SystemMetrics) string {
	w := max(41, m.width)
	gap := 1
	leftW := (w - gap) / 2
	rightW := w - gap - leftW

	openClawLines := m.openClawStatusLines(metrics.OpenClaw)
	ollamaLines := m.ollamaStatusLines(metrics.Ollama)

	leftBox := renderSmallBox(styleTitle.Render(" OpenClaw "), leftW, openClawLines)
	rightBox := renderSmallBox(styleTitle.Render(" Ollama "), rightW, ollamaLines)

	var lines []string
	for i := range leftBox {
		lines = append(lines, leftBox[i]+" "+rightBox[i])
	}
	return strings.Join(lines, "\n")
}

func (m Model) openClawStatusLines(s monitor.OpenClawStatus) []string {
	if !s.Running {
		return []string{
			styleBad.Render("● offline"),
			styleDim.Render("no OpenClaw service detected"),
		}
	}
	return []string{
		styleGood.Render("● online") + styleDim.Render(fmt.Sprintf("  pid %d · %s · up %s",
			s.PID, monitor.FormatMemory(s.Memory), formatUptime(s.Uptime))),
		styleDim.Render("OpenClaw service detected"),
	}
}

func (m Model) ollamaStatusLines(s monitor.OllamaStatus) []string {
	if !s.Running {
		return []string{
			styleBad.Render("● offline"),
			styleDim.Render("no Ollama API detected"),
		}
	}

	var names []string
	for i, model := range s.Models {
		if i >= 3 {
			break
		}
		names = append(names, model.Name+" "+modelHeat(model.Name))
	}
	modelLine := styleDim.Render("no loaded models")
	if len(names) > 0 {
		modelLine = strings.Join(names, styleDim.Render("  "))
	}
	return []string{
		styleGood.Render("● online") + styleDim.Render(fmt.Sprintf("  %d models", len(s.Models))),
		modelLine,
	}
}

func renderSmallBox(title string, width int, lines []string) []string {
	if width < 20 {
		width = 20
	}
	line0, line1 := "", ""
	if len(lines) > 0 {
		line0 = lines[0]
	}
	if len(lines) > 1 {
		line1 = lines[1]
	}
	return []string{
		boxTop(title, "", width),
		boxLine(line0, width),
		boxLine(line1, width),
		boxBottom(width),
	}
}

func formatUptime(d time.Duration) string {
	h := int(d.Hours())
	mn := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, mn)
	}
	return fmt.Sprintf("%dm", mn)
}

func (m Model) renderInsightsAndLogs(metrics *monitor.SystemMetrics) string {
	w := max(41, m.width)
	gap := 1
	leftW := (w - gap) / 2
	rightW := w - gap - leftW

	insightLines := m.generateInsights(metrics)
	logLines := m.ollamaLogLines(metrics)

	leftBox := m.renderTextBox(styleTitle.Render(" Operational Insights "), leftW, insightLines)
	rightBox := m.renderTextBox(styleTitle.Render(" Ollama Logs "), rightW, logLines)

	var lines []string
	for i := range leftBox {
		lines = append(lines, leftBox[i]+" "+rightBox[i])
	}
	return strings.Join(lines, "\n")
}

const insightBoxRows = 6 // content rows inside the box

func (m Model) renderTextBox(title string, width int, lines []string) []string {
	if width < 20 {
		width = 20
	}
	out := []string{boxTop(title, "", width)}
	for i := 0; i < insightBoxRows; i++ {
		content := ""
		if i < len(lines) {
			content = lines[i]
		}
		out = append(out, boxLine(content, width))
	}
	out = append(out, boxBottom(width))
	return out
}

func (m Model) generateInsights(metrics *monitor.SystemMetrics) []string {
	var lines []string

	ramPct := float64(metrics.SysInfo.MemPercent)
	switch {
	case ramPct >= 85:
		lines = append(lines, styleWarn.Render("• RAM critically high. Unload models or restart Ollama."))
	case ramPct >= 70:
		lines = append(lines, styleText.Render("• RAM usage is moderately high. Keep-alive")+
			styleText.Render(" settings may need review."))
	}

	switch {
	case metrics.DiskUsagePct >= 90:
		lines = append(lines, styleBad.Render("• Disk usage critical. Free space immediately."))
	case metrics.DiskUsagePct >= 80:
		lines = append(lines, styleWarn.Render("• Disk usage is approaching a critical")+
			styleWarn.Render(" threshold. Old models may need cleanup."))
	}

	var largestName string
	var largestBytes int64
	for _, model := range metrics.Ollama.Models {
		if model.SizeBytes > largestBytes {
			largestBytes = model.SizeBytes
			largestName = model.Name
		}
	}
	if largestName != "" {
		lines = append(lines, styleText.Render(fmt.Sprintf("• Largest installed model: %s (%s)",
			largestName, monitor.FormatMemory(uint64(largestBytes)))))
	}

	if !metrics.Ollama.Running {
		lines = append(lines, styleDim.Render("• Ollama is not running."))
	} else if len(metrics.Ollama.Models) == 0 {
		lines = append(lines, styleDim.Render("• No active in-memory model is visible right now."))
	}

	if len(lines) == 0 {
		lines = append(lines, styleGood.Render("• System looks healthy."))
	}
	return lines
}

func (m Model) ollamaLogLines(metrics *monitor.SystemMetrics) []string {
	path := monitor.OllamaLogPath()
	lines := []string{styleDim.Render(path)}
	for _, l := range metrics.OllamaLogs {
		lines = append(lines, styleText.Render(l))
	}
	return lines
}


func (m Model) renderProcessPanel(metrics *monitor.SystemMetrics, contentHeight int) string {
	w := max(40, m.width)
	innerW := w - 4

	items := m.list.items
	list := m.list
	if len(items) == 0 {
		items = m.buildListItems(metrics)
		list.SetItems(items)
	}

	lines := list.Render(innerW, contentHeight)

	above, below := list.ScrollInfo()
	scrollHint := ""
	if above > 0 {
		scrollHint += styleWarn.Render(fmt.Sprintf(" ↑%d", above))
	}
	if below > 0 {
		scrollHint += styleWarn.Render(fmt.Sprintf(" ↓%d", below))
	}

	count := selectableCount(items)
	sortIndicator := styleDim.Render(fmt.Sprintf(" %d procs · sorted:%s ", count, sortLabel(m.sortBy))) + scrollHint

	var sb strings.Builder
	sb.WriteString(boxTop(styleTitle.Render(" Processes "), sortIndicator, w))
	sb.WriteString("\n")
	for i, line := range lines {
		if i >= contentHeight {
			break
		}
		sb.WriteString(boxLine(line, w))
		sb.WriteString("\n")
	}
	sb.WriteString(boxBottom(w))
	return sb.String()
}

func (m Model) buildListItems(metrics *monitor.SystemMetrics) []ListItem {
	var items []ListItem

	if len(metrics.Ollama.Models) > 0 {
		items = append(items, ListItem{Kind: KindSectionHead, Label: "Ollama models"})
		for _, model := range metrics.Ollama.Models {
			items = append(items, ListItem{
				Kind:      KindOllamaModel,
				Label:     model.Name,
				ModelName: model.Name,
				Extra:     model.Size,
			})
		}
	}

	if metrics.OllamaProcess != nil {
		items = append(items,
			ListItem{Kind: KindSectionHead, Label: "ollama process"},
			processItem(*metrics.OllamaProcess),
		)
	}

	processes := openClawProcesses(metrics.Processes)
	sortProcesses(processes, m.sortBy)
	if len(processes) > 0 {
		items = append(items, ListItem{Kind: KindSectionHead, Label: "OpenClaw processes"})
		for _, p := range processes {
			items = append(items, processItem(p))
		}
	}

	if len(items) == 0 {
		items = append(items, ListItem{Kind: KindSectionHead, Label: "no monitored processes found"})
	}
	return items
}

func processItem(p monitor.ProcessInfo) ListItem {
	return ListItem{
		Kind:      KindProcess,
		Label:     p.Name,
		PID:       p.PID,
		CPU:       p.CPU,
		Memory:    p.Memory,
		MemoryPct: p.MemoryPct,
		Extra:     p.User,
	}
}

func openClawProcesses(processes []monitor.ProcessInfo) []monitor.ProcessInfo {
	var result []monitor.ProcessInfo
	for _, p := range processes {
		if p.Name == "ollama" {
			continue
		}
		result = append(result, p)
	}
	return result
}

func sortProcesses(processes []monitor.ProcessInfo, sortBy string) {
	switch sortBy {
	case "cpu":
		sortByCPU(processes)
	case "name":
		sortByName(processes)
	default:
		sortByMemory(processes)
	}
}

func sortByCPU(processes []monitor.ProcessInfo) {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].CPU > processes[j].CPU
	})
}

func sortByMemory(processes []monitor.ProcessInfo) {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].Memory > processes[j].Memory
	})
}

func sortByName(processes []monitor.ProcessInfo) {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].Name < processes[j].Name
	})
}

func selectableCount(items []ListItem) int {
	count := 0
	for _, item := range items {
		if item.selectable() {
			count++
		}
	}
	return count
}

func sortLabel(sortBy string) string {
	switch sortBy {
	case "cpu":
		return "cpu"
	case "name":
		return "name"
	default:
		return "mem"
	}
}

func (m Model) renderFooter() string {
	type binding struct{ key, desc string }
	bindings := []binding{
		{"↑↓", "select row"},
		{"k", "kill/unload"},
		{"r", "restart/unload"},
		{"m/c/s", "sort"},
		{"Space", "pause"},
		{"q", "quit"},
	}

	sep := styleDim.Render("  │  ")
	var parts []string
	for _, b := range bindings {
		parts = append(parts, styleTitle.Render(b.key)+styleDim.Render(":"+b.desc))
	}
	if m.paused {
		parts = append(parts, styleWarn.Render("paused"))
	}
	if m.errMsg != "" {
		parts = append(parts, styleBad.Render(truncate(m.errMsg, 36)))
	}
	return "  " + strings.Join(parts, sep)
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

// truncate clips s to at most n visible characters, appending "…" if clipped.
func truncate(s string, n int) string {
	if lipgloss.Width(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}

	var b strings.Builder
	for _, r := range s {
		if lipgloss.Width(b.String()+string(r)+"…") > n {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + "…"
}
