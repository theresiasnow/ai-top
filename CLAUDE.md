# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o bin/ai-top ./cmd/ai-top/

# Run
./bin/ai-top

# Tests
go test ./...
go test -v ./internal/monitor/...
go test -run TestFormatMemory ./internal/monitor/...

# Vet
go vet ./...
```

Tests are integration-style and tolerate missing services (Ollama, OpenClaw not running is fine).

## Architecture

**Data flow:** `Monitor.StartAutoRefresh()` (goroutine, 2s tick) → `Monitor.Refresh()` → updates `SystemMetrics` under `sync.RWMutex` → `ui.Model.View()` calls `Monitor.GetMetrics()` (snapshot copy) on each 500ms bubbletea tick.

Two layers:

- **`internal/monitor/`** — all data collection. `Monitor` is the central struct. Each file owns one concern: `processes.go` (Node/Ollama process discovery), `ollama.go` (HTTP API client), `openclaw.go` (process + HTTP health detection), `cron.go` (reads `~/.openclaw/cron/jobs.json`), `system.go` (CPU/mem via gopsutil), `ollama_extended.go` (model priority helpers).

- **`internal/ui/model.go`** — single bubbletea `Model`, all rendering. Rendering functions return `[]string` (lines); `renderMainPanel` wraps them in box borders. Column widths are derived from `m.width - 4` (inner box width).

## Conventions

- `gofmt`/`goimports` formatting throughout.
- No new top-level dependencies without discussion.
- OpenClaw port hardcoded to 3000; Ollama port to 11434.
- Model priority (`isHotModel`, `isWarmModel`) is hardcoded in `ui/model.go` — currently `qwen2.5-coder:14b`, `gemma4:latest` (hot) and `qwen3:8b` (warm).

## Known Issues (backlog)

- **Data race**: `StartAutoRefresh` reads `m.refreshRate` before acquiring the lock.
- **`SetRefreshRate`** has no effect after auto-refresh starts (ticker not reset).
- **`GetMetrics`** does not deep-copy `Ollama.Models` slice.
- **Goroutine leak**: no shutdown path for the auto-refresh goroutine.
- **`io.ReadAll`** on Ollama error body is unbounded (should use `io.LimitReader`).
- **`MemoryPct`** is not populated in `getProcessInfo` — process memory bars always show 0%.
