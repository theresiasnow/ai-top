package main

import (
"fmt"
"log"
"os"

tea "github.com/charmbracelet/bubbletea"
"github.com/tess/ai-top/internal/monitor"
"github.com/tess/ai-top/internal/ui"
)

func main() {
// Create monitor
mon := monitor.NewMonitor()

// Initial refresh
if err := mon.Refresh(); err != nil {
log.Printf("Initial refresh error: %v", err)
}

// Start background refresh
mon.StartAutoRefresh()

// Create model
model := ui.NewModel(mon)

// Create program
p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

if _, err := p.Run(); err != nil {
fmt.Fprintf(os.Stderr, "Error: %v\n", err)
os.Exit(1)
}
}
