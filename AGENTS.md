# AGENTS.md — ai-top

## Project

`ai-top` is a btop-inspired TUI monitor for AI development environments.
Written in Go using [Bubbletea](https://github.com/charmbracelet/bubbletea) + Lipgloss.

Binary: `bin/ai-top` (committed pre-built for quick use).

## Stack

| Layer    | Library                        |
|----------|-------------------------------|
| TUI loop | `github.com/charmbracelet/bubbletea` |
| Styling  | `github.com/charmbracelet/lipgloss` |
| Metrics  | `github.com/shirou/gopsutil/v3` |

## Layout

```
cmd/ai-top/main.go          — entrypoint
internal/
  monitor/
    monitor.go              — Monitor struct, Refresh, StartAutoRefresh
    processes.go            — Node.js / Ollama process discovery, FormatMemory
    ollama.go               — Ollama HTTP API client
    ollama_extended.go      — Model priority, load status helpers
    openclaw.go             — OpenClaw service detection (HTTP + process)
    cron.go                 — Read ~/.openclaw/cron/jobs.json
    system.go               — CPU / memory via gopsutil
  ui/
    model.go                — Bubbletea Model, all rendering
```

## Build & Run

```bash
go build -o bin/ai-top ./cmd/ai-top/
./bin/ai-top
```

Tests:
```bash
go test ./...
go vet ./...
```

## Key Bindings

| Key         | Action              |
|-------------|---------------------|
| 1-4 / Tab   | Switch tabs         |
| m / c / s   | Sort by mem/cpu/name |
| Space       | Pause refresh       |
| q / Ctrl-C  | Quit                |

## UI Architecture

All rendering lives in `internal/ui/model.go`.

- `renderHeader(metrics)` — title + clock + service status (2 lines)
- `renderMainPanel(metrics)` — single rounded-border panel with tab bar embedded in top border
- `renderFooter()` — key hint bar
- `boxTop / boxLine / boxDivider / boxBottom` — btop-style box drawing helpers
- `miniBar / cpuBar / memBar` — coloured progress bar helpers

## Colour Palette (btop-inspired)

| Role       | Hex       |
|------------|-----------|
| Cyan/title | `#00C8FF` |
| Green/good | `#60FF60` |
| Amber/warn | `#FFB300` |
| Red/bad    | `#FF4444` |
| Blue/heads | `#7AAFFF` |
| Dim/border | `#445566` |

## Known Issues / Backlog

- Data race: `StartAutoRefresh` reads `m.refreshRate` before taking the lock
- `SetRefreshRate` has no effect after auto-refresh starts (ticker not reset)
- `GetMetrics` does not deep-copy `Ollama.Models` slice
- Goroutine leak: no shutdown path for the auto-refresh goroutine
- `io.ReadAll` on Ollama error body is unbounded (use `io.LimitReader`)
- `MemoryPct` is not populated in `getProcessInfo` — process memory bars always show 0%

## Conventions

- Match existing style (`gofmt`/`goimports`)
- No new top-level dependencies without asking
- Rendering functions return `[]string` (lines) — `renderMainPanel` wraps them in box borders
- Column widths are computed from `m.width - 4` (inner box width)
