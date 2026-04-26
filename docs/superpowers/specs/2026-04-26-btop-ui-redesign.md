# ai-top: btop-style UI redesign + kill/restart

**Date:** 2026-04-26
**Status:** Approved

## Goal

Replace the tab-based layout with a btop-inspired always-on status bar + scrollable process list. Add arrow-key row selection and immediate kill/restart/unload actions.

## Layout

```
╭─ OpenClaw ──────────────────────────────╮╭─ Ollama ─────────────────────────────────╮
│ ● online  pid 5063 · 548.8MB · up 3h2m  ││ ● online  4 models · 22.5GB loaded      │
│ 7 openclaw procs                         ││ qwen2.5-coder 🔥  gemma4 🔥  qwen3 🌡   │
╰─────────────────────────────────────────╯╰──────────────────────────────────────────╯
╭─ Processes ───────────────────────────────────── 34 procs · sorted: mem ─────────────╮
│      NAME                         SIZE / USER    CPU              MEMORY             │
│ ────────────────────────────────────────────────────────────────────────────────── │
│ ── Ollama models ──────────────────────────────────────────────────────────────────  │
│   qwen2.5-coder:14b               8.4GB          🔥 hot          ● loaded            │
│   gemma4:latest                   8.9GB          🔥 hot          ● loaded            │
│   ── ollama process ────────────────────────────────────────────────────────────── │
│   ollama                          system         ░░░░░░░░  0.2%  ██████░░  22.5GB   │
│ ── OpenClaw processes ──────────────────────────────────────────────────────────── │
│   39217  node                     tess           ░░░░░░░░  0.0%  ███░░░░░  548.8MB  │
│▶ 85707  node                     tess           ██░░░░░░ 12.3%  █░░░░░░░   97.8MB  │
│   ...                                                                               │
│  ⚡ k: kill  PID 85707 · node    ↺ r: SIGHUP                                        │
╰─────────────────────────────────────────────────────────────────────────────────────╯
  ↑↓:select row  k:kill  r:restart/unload  m/c/s:sort  space:pause  q:quit
```

- Status bar: always visible, non-interactive. Two equal-width boxes side by side.
- Process panel: scrollable list. Section headers are non-selectable (cursor skips them).
- Action bar inside the panel shows context-sensitive hint for the selected row.
- Sort (`m`/`c`/`s`) applies to OpenClaw processes only; Ollama models stay pinned at top.

## Actions per row type

| Row type | `k` | `r` |
|---|---|---|
| Ollama model | Unload via API (`keep_alive: 0`) | Same as k |
| ollama process | SIGTERM | SIGHUP |
| OpenClaw / node process | SIGTERM | SIGHUP |
| Section header | (no-op) | (no-op) |

Actions are immediate — no confirmation dialog. After any action, trigger `mon.Refresh()` immediately so the list updates without waiting for the 2s auto-refresh tick.

## New / changed files

### `internal/ui/list.go` (new)

`SelectableList` struct — owns selection state and rendering. Knows nothing about processes or Ollama.

```go
type ItemKind int
const (KindSectionHead ItemKind = iota; KindOllamaModel; KindProcess)

type ListItem struct {
    Kind      ItemKind
    Label     string    // display name
    PID       int       // processes only
    ModelName string    // ollama models only
    CPU       float64
    Memory    uint64
    MemoryPct float32
    Extra     string    // size string for models, user for processes
}

type Action int
const (ActionNone Action = iota; ActionKill; ActionRestart; ActionUnload)
```

Methods:
- `HandleKey(key string) Action` — moves cursor for `↑`/`↓`, returns action for `k`/`r`; skips section header rows
- `Render(innerWidth, height int) []string` — returns lines including action bar
- `SelectedItem() ListItem`
- `SetItems(items []ListItem)` — called each tick from model.go; clamps `selected` to `[0, len(selectableItems)-1]` so a dying process never leaves the cursor out of bounds

### `internal/monitor/actions.go` (new)

Three functions, nothing else:
- `KillProcess(pid int) error` → `syscall.Kill(pid, syscall.SIGTERM)`
- `RestartProcess(pid int) error` → `syscall.Kill(pid, syscall.SIGHUP)`
- `UnloadOllamaModel(client *OllamaClient, name string) error` → POST `/api/generate` with `{"model": "<name>", "keep_alive": 0}` using existing `OllamaClient.port`

### `internal/monitor/processes.go` (extend)

Add `GetOllamaProcess() (ProcessInfo, bool)` — finds the running `ollama` binary via gopsutil by executable name.

### `internal/monitor/monitor.go` (extend)

Add `OllamaProcess *ProcessInfo` to `SystemMetrics`. Populate it in `Refresh()` via `GetOllamaProcess()`. `GetMetrics()` must deep-copy the pointed-to struct value (not just the pointer): `if m.metrics.OllamaProcess != nil { v := *m.metrics.OllamaProcess; snapshot.OllamaProcess = &v }`.

### `internal/ui/model.go` (modify)

- Remove `activeTab int` from `Model`, add `list SelectableList`
- `renderHeader` → `renderStatusBar(metrics)`: two side-by-side boxes using existing box-drawing helpers
- `renderMainPanel` → `renderProcessPanel(metrics)`: builds `[]ListItem` each tick (Ollama models → ollama process → OpenClaw processes), calls `list.SetItems()` + `list.Render()`
- `Update()`: remove tab-switching keys, add `↑`/`↓` → `list.HandleKey()`, `k`/`r` → `list.HandleKey()` then dispatch to `monitor.KillProcess` / `monitor.RestartProcess` / `monitor.UnloadOllamaModel` then `mon.Refresh()`
- Footer: updated keybindings

## Removed features

- **Tab navigation (1–4 / Tab key)** — the tab system is gone. The single panel replaces all four tabs.
- **Cron tab** — no longer displayed. The `~/.openclaw/cron/jobs.json` reader remains in `monitor/cron.go` but is not called from the UI. Can be re-added later as a second panel if needed.

## Out of scope

- Confirmation dialogs
- Mouse support
- Custom Ollama port (existing hardcoded 11434 unchanged)
