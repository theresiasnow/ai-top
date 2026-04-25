package main

import (
"fmt"
"log"
"time"

"github.com/rivo/tview"
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

// Create app
app := tview.NewApplication()

// Create UI
dashboard := ui.NewDashboard(mon, app)

// Update loop
go func() {
ticker := time.NewTicker(500 * time.Millisecond)
defer ticker.Stop()

for range ticker.C {
app.QueueUpdateDraw(func() {
dashboard.Update()
})
}
}()

// Set root and run
if err := app.SetRoot(dashboard.Root(), true).Run(); err != nil {
log.Fatalf("Failed to run app: %v", err)
}

fmt.Println("\nai-top stopped")
}
