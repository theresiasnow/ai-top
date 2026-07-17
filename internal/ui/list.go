package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tess/ai-top/internal/monitor"
)

type ItemKind int

const (
	KindSectionHead ItemKind = iota
	KindOllamaModel
	KindProcess
)

type ListItem struct {
	Kind      ItemKind
	Provider  string
	Label     string
	PID       int
	ModelName string
	CPU       float64
	Memory    uint64
	MemoryPct float32
	Extra     string
	Loaded    bool // for KindOllamaModel: true = in VRAM, false = on disk
}

type Action int

const (
	ActionNone Action = iota
	ActionKill
	ActionRestart
	ActionUnload
)

type SelectableList struct {
	items       []ListItem
	selected    int
	offset      int
	lastVisible int // set by Render, used by ScrollInfo
}

func (l *SelectableList) SetItems(items []ListItem) {
	l.items = items
	if len(l.selectableIndexes()) == 0 {
		l.selected = 0
		l.offset = 0
		return
	}
	if l.selected < 0 || l.selected >= len(l.items) || !l.items[l.selected].selectable() {
		l.selected = l.firstSelectable()
	}
}

func (l *SelectableList) HandleKey(key string) Action {
	switch key {
	case "up":
		l.move(-1)
	case "down":
		l.move(1)
	case "k":
		item := l.SelectedItem()
		if item.Kind == KindOllamaModel {
			if item.Provider == "omlx" {
				return ActionNone
			}
			return ActionUnload
		}
		if item.Kind == KindProcess {
			return ActionKill
		}
	case "r":
		item := l.SelectedItem()
		if item.Kind == KindOllamaModel {
			if item.Provider == "omlx" {
				return ActionNone
			}
			return ActionUnload
		}
		if item.Kind == KindProcess {
			return ActionRestart
		}
	}
	return ActionNone
}

func (l *SelectableList) SelectedItem() ListItem {
	if l.selected < 0 || l.selected >= len(l.items) {
		return ListItem{}
	}
	return l.items[l.selected]
}

func (l *SelectableList) Render(innerWidth, height int) []string {
	if height < 3 {
		height = 3
	}

	lines := []string{l.header(innerWidth), styleDim.Render(strings.Repeat("─", innerWidth))}
	visibleRows := height - 3
	if visibleRows < 1 {
		visibleRows = 1
	}
	l.clampOffset(visibleRows)
	l.lastVisible = visibleRows

	end := l.offset + visibleRows
	if end > len(l.items) {
		end = len(l.items)
	}

	for i := l.offset; i < end; i++ {
		lines = append(lines, l.renderItem(i, l.items[i], innerWidth))
	}
	for len(lines) < height-1 {
		lines = append(lines, "")
	}

	lines = append(lines, l.actionBar(innerWidth))
	return lines
}

// column widths shared by the header and every data row so they stay aligned.
const (
	colSizeW = 14 // "SIZE / USER" — user for processes, size for models
	colCPUW  = 15 // CPU% or model-heat
	colMemW  = 17 // memory value or VRAM bar
)

// colNameW returns the NAME column width for a given inner box width.
// The header and data rows MUST use this same value or columns drift apart.
func colNameW(innerWidth int) int {
	// "  " cursor gutter + 3× "  " inter-column spacers = 8 fixed chars.
	nameW := innerWidth - colSizeW - colCPUW - colMemW - 8
	if nameW < 12 {
		nameW = 12
	}
	return nameW
}

func (l *SelectableList) header(innerWidth int) string {
	return styleColHead.Render(fmt.Sprintf("  %-*s  %-*s  %*s  %*s",
		colNameW(innerWidth), "NAME", colSizeW, "SIZE / USER", colCPUW, "CPU", colMemW, "MEMORY"))
}

func (l *SelectableList) renderItem(index int, item ListItem, innerWidth int) string {
	if item.Kind == KindSectionHead {
		prefix := "── " + truncate(item.Label, max(1, innerWidth-6)) + " "
		fill := innerWidth - lipgloss.Width(prefix)
		if fill < 0 {
			fill = 0
		}
		return styleDim.Render(prefix + strings.Repeat("─", fill))
	}

	nameW := colNameW(innerWidth)

	cursor := " "
	if l.selected == index {
		cursor = styleTitle.Render("▶")
	}

	name := item.Label
	if item.PID != 0 {
		// Pad the PID to 6 columns so labels start at the same column
		// regardless of PID width (macOS PIDs vary from 3 to 5+ digits).
		name = fmt.Sprintf("%-6d %s", item.PID, item.Label)
	}
	nameStr := lipgloss.NewStyle().Width(nameW).Foreground(colorText).Render(truncate(name, nameW))
	extraStr := lipgloss.NewStyle().Width(colSizeW).Foreground(colorDim).Render(truncate(item.Extra, colSizeW))

	if item.Kind == KindOllamaModel {
		cpuStr := lipgloss.NewStyle().Width(colCPUW).Render(modelHeat(item.ModelName))
		var memStr string
		if item.Loaded {
			cpuStr = lipgloss.NewStyle().Width(colCPUW).Render(greenCPUBar(item.CPU))
			memStr = lipgloss.NewStyle().Width(colMemW).Render(
				styleGood.Render("▶ ") + greenMemBar(item.Memory, item.MemoryPct))
		} else {
			sizeInfo := item.Extra
			if sizeInfo == "" {
				sizeInfo = "on disk"
			}
			memStr = lipgloss.NewStyle().Width(colMemW).Render(
				styleDim.Render("○ " + sizeInfo))
		}
		return cursor + " " + nameStr + "  " + extraStr + "  " + cpuStr + "  " + memStr
	}

	// Process rows: plain right-aligned numbers (no bars). The bar backgrounds
	// carry no signal for a process list where most values sit near zero.
	cpuStr := lipgloss.NewStyle().Width(colCPUW).Foreground(cpuColor(item.CPU)).
		Render(fmt.Sprintf("%*s", colCPUW, fmt.Sprintf("%.1f%%", item.CPU)))
	memStr := lipgloss.NewStyle().Width(colMemW).Foreground(memColor(item.MemoryPct)).
		Render(fmt.Sprintf("%*s", colMemW, monitor.FormatMemory(item.Memory)))
	return cursor + " " + nameStr + "  " + extraStr + "  " + cpuStr + "  " + memStr
}

// cpuColor / memColor mirror the bar color thresholds so numbers keep the
// same green→amber→red severity cue the bars used to give.
func cpuColor(pct float64) lipgloss.Color {
	switch {
	case pct >= 80:
		return colorRed
	case pct >= 50:
		return colorAmber
	default:
		return colorGreen
	}
}

func memColor(pct float32) lipgloss.Color {
	switch {
	case pct >= 10:
		return colorRed
	case pct >= 5:
		return colorAmber
	default:
		return colorGreen
	}
}

func (l *SelectableList) actionBar(innerWidth int) string {
	item := l.SelectedItem()
	var text string
	switch item.Kind {
	case KindOllamaModel:
		if item.Provider == "omlx" {
			text = fmt.Sprintf("  omlx model  %s", item.ModelName)
		} else {
			text = fmt.Sprintf("  k/r: unload  %s", item.ModelName)
		}
	case KindProcess:
		text = fmt.Sprintf("  k: kill  PID %d · %s    r: SIGHUP", item.PID, item.Label)
	default:
		text = "  no selectable rows"
	}
	return styleWarn.Render(truncate(text, innerWidth))
}

// ScrollInfo returns the number of items hidden above and below the viewport.
func (l *SelectableList) ScrollInfo() (above, below int) {
	above = l.offset
	end := l.offset + l.lastVisible
	if end > len(l.items) {
		end = len(l.items)
	}
	below = len(l.items) - end
	if below < 0 {
		below = 0
	}
	return
}

func (l *SelectableList) move(delta int) {
	if len(l.items) == 0 {
		return
	}
	idx := l.selected
	for {
		idx += delta
		if idx < 0 || idx >= len(l.items) {
			return
		}
		if l.items[idx].selectable() {
			l.selected = idx
			return
		}
	}
}

func (l *SelectableList) clampOffset(visibleRows int) {
	if l.selected < l.offset {
		l.offset = l.selected
	}
	if l.selected >= l.offset+visibleRows {
		l.offset = l.selected - visibleRows + 1
	}
	if l.offset < 0 {
		l.offset = 0
	}
	if maxOffset := len(l.items) - visibleRows; l.offset > maxOffset {
		l.offset = max(0, maxOffset)
	}
}

func (l *SelectableList) firstSelectable() int {
	for i, item := range l.items {
		if item.selectable() {
			return i
		}
	}
	return 0
}

func (l *SelectableList) selectableIndexes() []int {
	var indexes []int
	for i, item := range l.items {
		if item.selectable() {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (i ListItem) selectable() bool {
	return i.Kind == KindOllamaModel || i.Kind == KindProcess
}

func modelHeat(name string) string {
	switch {
	case isHotModel(name):
		return styleGood.Render("hot")
	case isWarmModel(name):
		return styleWarn.Render("warm")
	default:
		return styleDim.Render("cold")
	}
}
