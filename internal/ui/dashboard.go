package ui

import (
"fmt"
"time"

"github.com/gdamore/tcell/v2"
"github.com/rivo/tview"
"github.com/tess/ai-top/internal/monitor"
)

// Dashboard is the main UI
type Dashboard struct {
app     *tview.Application
root    tview.Primitive
monitor *monitor.Monitor
grid    *tview.Grid
content *tview.TextView
paused  bool
sortBy  string // cpu, memory, name
}

// NewDashboard creates the dashboard
func NewDashboard(mon *monitor.Monitor, app *tview.Application) *Dashboard {
grid := tview.NewGrid().
SetRows(3, 0, 3).
SetColumns(0)

// Header
header := tview.NewTextView().
SetDynamicColors(true).
SetText("[yellow]ai-top[-] | Node.js & Ollama Status | Press 'q' to quit")

// Main content area
content := tview.NewTextView().
SetDynamicColors(true).
SetText("Loading metrics...")

// Footer with help
footer := tview.NewTextView().
SetDynamicColors(true).
SetText("[gray]↑↓ Navigate | [yellow]space[-] Pause | [yellow]s[-] Sort | [yellow]c[-] CPU | [yellow]m[-] Memory | [yellow]q[-] Quit")

grid.AddItem(header, 0, 0, 1, 1, 0, 0, false).
AddItem(content, 1, 0, 1, 1, 0, 0, true).
AddItem(footer, 2, 0, 1, 1, 0, 0, false)

d := &Dashboard{
app:     app,
root:    grid,
monitor: mon,
grid:    grid,
content: content,
sortBy:  "name",
}

// Set up key handling
d.setupKeyHandlers()

return d
}

// Root returns the root widget
func (d *Dashboard) Root() tview.Primitive {
return d.root
}

// setupKeyHandlers configures keyboard input
func (d *Dashboard) setupKeyHandlers() {
d.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
switch event.Rune() {
case 'q', 'Q':
d.app.Stop()
return nil
case ' ':
d.paused = !d.paused
return nil
case 's', 'S':
d.sortBy = "name"
return nil
case 'c', 'C':
d.sortBy = "cpu"
return nil
case 'm', 'M':
d.sortBy = "memory"
return nil
}
return event
})
}

// Update refreshes the displayed content
func (d *Dashboard) Update() {
if d.paused {
return
}

metrics := d.monitor.GetMetrics()
text := d.formatMetrics(metrics)
d.content.SetText(text)
}

// formatMetrics converts metrics to display text
func (d *Dashboard) formatMetrics(m *monitor.SystemMetrics) string {
var output string

// Updated at
output += fmt.Sprintf("[gray]Last update: %s[-]\n\n", m.UpdatedAt.Format("15:04:05"))

// OpenClaw status
output += "[yellow]OpenClaw[-]\n"
if m.OpenClaw.Running {
output += fmt.Sprintf("  [green]●[-] Running (PID: %d, Memory: %s, Uptime: %s)\n",
m.OpenClaw.PID, monitor.FormatMemory(m.OpenClaw.Memory), monitor.GetProcessUptime(time.Now().Add(-m.OpenClaw.Uptime)))
} else {
output += "  [red]●[-] Not running\n"
}

// Ollama status
output += "\n[yellow]Ollama[-]\n"
if m.Ollama.Running {
output += fmt.Sprintf("  [green]●[-] Running (%d models loaded)\n", len(m.Ollama.Models))
for _, model := range m.Ollama.Models {
output += fmt.Sprintf("    • %s (%s)\n", model.Name, model.Size)
}
} else {
output += "  [red]●[-] Not running\n"
}

// Processes
output += "\n[yellow]Node.js Processes[-]\n"
if len(m.Processes) == 0 {
output += "  (none)\n"
} else {
// Header
output += fmt.Sprintf("  [cyan]%-8s %-30s %-8s %-10s[-]\n", "PID", "Name", "User", "Memory")
output += "  " + "─────────────────────────────────────────────────\n"

for _, p := range m.Processes {
output += fmt.Sprintf("  %-8d %-30s %-8s %-10s\n",
p.PID, p.Name, p.User, monitor.FormatMemory(p.Memory))
}
}

// Cron jobs (placeholder)
output += "\n[yellow]Cron Jobs[-]\n"
if len(m.CronJobs) == 0 {
output += "  (none currently tracked)\n"
} else {
for _, cron := range m.CronJobs {
output += fmt.Sprintf("  • %s: %s\n", cron.Name, cron.Status)
}
}

return output
}
