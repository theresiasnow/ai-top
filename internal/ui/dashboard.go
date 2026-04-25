package ui

import (
"fmt"
"sort"
"strings"
"time"

"github.com/gdamore/tcell/v2"
"github.com/rivo/tview"
"github.com/tess/ai-top/internal/monitor"
)

type Dashboard struct {
app     *tview.Application
root    *tview.Flex
monitor *monitor.Monitor
paused  bool
sortBy  string
}

func NewDashboard(mon *monitor.Monitor, app *tview.Application) *Dashboard {
d := &Dashboard{
app:     app,
monitor: mon,
sortBy:  "memory",
}

d.root = tview.NewFlex().SetDirection(tview.FlexRow)
d.root.AddItem(d.createHeader(), 2, 0, false)
d.root.AddItem(d.createStatsBox(), 4, 0, false)
d.root.AddItem(d.createServicesBox(), 3, 0, false)
d.root.AddItem(d.createContent(), 0, 1, true)
d.root.AddItem(d.createFooter(), 2, 0, false)

d.setupKeyHandlers()
return d
}

func (d *Dashboard) Root() tview.Primitive {
return d.root
}

func (d *Dashboard) createHeader() tview.Primitive {
text := tview.NewTextView().SetDynamicColors(true)
text.SetText("[cyan]┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓[-]\n" +
"[cyan]┃[-] [::b]📊 ai-top[-::-] - AI Development Monitor " +
strings.Repeat(" ", 25) + "[cyan]┃[-]\n" +
"[cyan]┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛[-]")
return text
}

func (d *Dashboard) createStatsBox() tview.Primitive {
statsView := tview.NewTextView().SetDynamicColors(true)

go func() {
ticker := time.NewTicker(1 * time.Second)
defer ticker.Stop()

for {
select {
case <-ticker.C:
info, _ := monitor.GetSystemInfo()
cpu := fmt.Sprintf("[yellow]%.1f%%[-]", info.CPUUsage)
mem := fmt.Sprintf("[lightblue]%s[-] / [white]%s[-] [yellow](%.1f%%)[-]",
monitor.FormatMemory(info.MemUsed),
monitor.FormatMemory(info.MemTotal),
float64(info.MemPercent))

content := fmt.Sprintf(
"  [green]CPU[-]: %s (%d cores)    [green]Memory[-]: %s\n",
cpu, info.CPUCores, mem)

d.app.QueueUpdateDraw(func() {
statsView.SetText(content)
})
}
}
}()

return statsView
}

func (d *Dashboard) createServicesBox() tview.Primitive {
flex := tview.NewFlex()

openclaw := tview.NewTextView().SetDynamicColors(true).SetText("[gray](loading)[-]")
ollama := tview.NewTextView().SetDynamicColors(true).SetText("[gray](loading)[-]")

flex.AddItem(openclaw, 0, 1, false)
flex.AddItem(ollama, 0, 1, false)

go func() {
ticker := time.NewTicker(2 * time.Second)
defer ticker.Stop()

for {
select {
case <-ticker.C:
metrics := d.monitor.GetMetrics()

oc := "  [red]●[-] OpenClaw: offline"
if metrics.OpenClaw.Running {
oc = "  [green]●[-] OpenClaw: online"
}

ol := "  [red]●[-] Ollama: offline"
if metrics.Ollama.Running {
ol = fmt.Sprintf("  [green]●[-] Ollama: online [lightblue](%d models)[-]", len(metrics.Ollama.Models))
}

d.app.QueueUpdateDraw(func() {
openclaw.SetText(oc)
ollama.SetText(ol)
})
}
}
}()

return flex
}

func (d *Dashboard) createContent() tview.Primitive {
flex := tview.NewFlex().SetDirection(tview.FlexRow)

tabBar := tview.NewTextView().
SetDynamicColors(true).
SetText("  [white:green] Node.js [-::-]  [lightblue]Ollama[-]  [lightblue]OpenClaw[-]  [lightblue]Cron[-]")

flex.AddItem(tabBar, 1, 0, false)
flex.AddItem(d.createProcessTable("Node.js", func() []monitor.ProcessInfo {
procs, _ := d.monitor.GetNodeProcesses()
return procs
}), 0, 1, true)

return flex
}

func (d *Dashboard) createProcessTable(name string, getProcs func() []monitor.ProcessInfo) tview.Primitive {
table := tview.NewTable().
SetBorders(false).
SetSelectable(true, false).
SetFixed(1, 0)

headers := []string{"PID", "NAME", "USER", "CPU", "MEMORY"}
for col, h := range headers {
cell := tview.NewTableCell(" " + h + " ").
SetTextColor(tcell.ColorYellow).
SetSelectable(false)
table.SetCell(0, col, cell)
}

updateTable := func() {
processes := getProcs()

switch d.sortBy {
case "cpu":
sort.Slice(processes, func(i, j int) bool {
return processes[i].CPU > processes[j].CPU
})
case "memory":
sort.Slice(processes, func(i, j int) bool {
return processes[i].Memory > processes[j].Memory
})
default:
sort.Slice(processes, func(i, j int) bool {
return processes[i].Name < processes[j].Name
})
}

if len(processes) > 30 {
processes = processes[:30]
}

for r := table.GetRowCount() - 1; r > 0; r-- {
table.RemoveRow(r)
}

if len(processes) == 0 {
cell := tview.NewTableCell("(no processes)")
cell.SetTextColor(tcell.ColorGray)
table.SetCell(1, 0, cell)
return
}

for idx, p := range processes {
row := idx + 1
data := []string{
fmt.Sprintf("%d", p.PID),
p.Name,
p.User,
fmt.Sprintf("%.1f%%", p.CPU),
monitor.FormatMemory(p.Memory),
}

for col, text := range data {
color := tcell.ColorWhite
if col == 4 {
color = tcell.ColorLightBlue
}

cell := tview.NewTableCell(" " + padRight(text, 12) + " ").
SetTextColor(color)
table.SetCell(row, col, cell)
}
}
}

updateTable()

go func() {
ticker := time.NewTicker(1 * time.Second)
defer ticker.Stop()

for {
select {
case <-ticker.C:
if !d.paused {
d.app.QueueUpdateDraw(updateTable)
}
}
}
}()

return table
}

func (d *Dashboard) createFooter() tview.Primitive {
text := tview.NewTextView().
SetDynamicColors(true).
SetText("  [lightblue]q[-] Quit  [lightblue]space[-] Pause  [lightblue]s[-] Sort (name)  " +
"[lightblue]c[-] Sort (CPU)  [lightblue]m[-] Sort (memory)")

return text
}

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

func (d *Dashboard) Update() {}

func padRight(str string, length int) string {
if len(str) >= length {
return str
}
return str + strings.Repeat(" ", length-len(str))
}
