package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type ItemKind int

const (
	KindSectionHead ItemKind = iota
	KindOllamaModel
	KindProcess
)

type ListItem struct {
	Kind      ItemKind
	Label     string
	PID       int
	ModelName string
	CPU       float64
	Memory    uint64
	MemoryPct float32
	Extra     string
}

type Action int

const (
	ActionNone Action = iota
	ActionKill
	ActionRestart
	ActionUnload
)

type SelectableList struct {
	items    []ListItem
	selected int
	offset   int
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
			return ActionUnload
		}
		if item.Kind == KindProcess {
			return ActionKill
		}
	case "r":
		item := l.SelectedItem()
		if item.Kind == KindOllamaModel {
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

func (l *SelectableList) header(innerWidth int) string {
	const (
		sizeW   = 14
		cpuW    = 15
		memW    = 17
		spacers = 6
	)
	nameW := innerWidth - sizeW - cpuW - memW - spacers
	if nameW < 14 {
		nameW = 14
	}
	return styleColHead.Render(fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s",
		nameW, "NAME", sizeW, "SIZE / USER", cpuW, "CPU", memW, "MEMORY"))
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

	const (
		sizeW   = 14
		cpuW    = 15
		memW    = 17
		spacers = 6
	)
	nameW := innerWidth - sizeW - cpuW - memW - spacers - 2
	if nameW < 12 {
		nameW = 12
	}

	cursor := " "
	if l.selected == index {
		cursor = styleTitle.Render("▶")
	}

	name := item.Label
	if item.PID != 0 {
		name = fmt.Sprintf("%d  %s", item.PID, item.Label)
	}
	nameStr := lipgloss.NewStyle().Width(nameW).Foreground(colorText).Render(truncate(name, nameW))
	extraStr := lipgloss.NewStyle().Width(sizeW).Foreground(colorDim).Render(truncate(item.Extra, sizeW))

	if item.Kind == KindOllamaModel {
		cpuStr := lipgloss.NewStyle().Width(cpuW).Render(modelHeat(item.ModelName))
		memStr := lipgloss.NewStyle().Width(memW).Render(styleGood.Render("● loaded"))
		return cursor + " " + nameStr + "  " + extraStr + "  " + cpuStr + "  " + memStr
	}

	return cursor + " " + nameStr + "  " + extraStr + "  " + cpuBar(item.CPU) + "  " + memBar(item.Memory, item.MemoryPct)
}

func (l *SelectableList) actionBar(innerWidth int) string {
	item := l.SelectedItem()
	var text string
	switch item.Kind {
	case KindOllamaModel:
		text = fmt.Sprintf("  k/r: unload  %s", item.ModelName)
	case KindProcess:
		text = fmt.Sprintf("  k: kill  PID %d · %s    r: SIGHUP", item.PID, item.Label)
	default:
		text = "  no selectable rows"
	}
	return styleWarn.Render(truncate(text, innerWidth))
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
