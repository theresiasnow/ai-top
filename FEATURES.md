# ai-top Features & Usage Guide

## Dashboard Overview

The main dashboard displays:

```
🚀 ai-top  AI Development Environment Monitor
  ● OpenClaw online    ● Ollama online (4 models)
    Node.js    Ollama    OpenClaw    Cron   ▶

  ───────── ───────────────────────── ──────── ─────── ────────────
PID      NAME                      USER     CPU%    MEMORY      
  ───────── ───────────────────────── ──────── ─────── ────────────
  93609    node                      tess        1.3% 415.5MB
  87219    ollama                    tess        0.4% 37.6MB
  [... more processes ...]

  [1] Node  [2] Ollama  [3] OpenClaw  [4] Cron  [m]emory  [c]pu  [s]ort name  [space] Pause  [q] Quit
```

## Tab Features

### Tab 1: Node.js Processes (Default)

Shows all Node.js processes with:
- **PID** - Process ID
- **NAME** - Process name
- **USER** - Process owner
- **CPU%** - CPU usage percentage
- **MEMORY** - Memory usage (human-readable, color-coded)

**Sorting Options:**
- `m` - Sort by memory (default) - shows highest consumers first
- `c` - Sort by CPU usage
- `s` - Sort by process name

**Color Coding:**
- 🟢 Green memory - Under 100MB
- 🟡 Yellow memory - Over 100MB

### Tab 2: Ollama Models

Displays all loaded Ollama models:

```
  ─────────────────────────────────────── ─────────────────── ─────────────
MODEL                                     SIZE                STATUS
  ─────────────────────────────────────── ─────────────────── ─────────────
  qwen2.5-coder:14b                       8.4GB               loaded
  gemma4:latest                           8.9GB               loaded
  qwen3:8b                                4.9GB               loaded
  nomic-embed-text:latest                 261.6MB             loaded
```

**Features:**
- Shows model name, size, and status
- Only displays when Ollama is running
- Real-time updates as models load/unload
- Automatically truncates long model names

### Tab 3: OpenClaw Service

Displays OpenClaw service status:

```
  ✓ OpenClaw is online
  PID: 93609 | Memory: 415.5MB | Uptime: 2h 15m
```

**When offline:**
```
  ✗ OpenClaw is offline
```

**Information shown:**
- 🟢 Online/🔴 Offline status
- Process ID of main process
- Aggregate memory usage
- Service uptime (hours and minutes)

### Tab 4: Cron Jobs

Shows cron jobs from `~/.openclaw/cron/jobs.json`:

```
  ───────────────────────────────────────── ──────────────────── ──────────────
NAME                                        SCHEDULE             STATUS
  ───────────────────────────────────────── ──────────────────── ──────────────
  backup-database                           0 2 * * *            enabled
  sync-ollama-models                        0 */6 * * *          success
  cleanup-logs                              0 0 * * 0            enabled
```

**Features:**
- Shows job name, schedule, and status
- Color-coded status (green for success/enabled, red for failed/disabled)
- Requires properly formatted jobs.json file

## Keyboard Controls

| Shortcut | Action | Notes |
|----------|--------|-------|
| `1` | Switch to Node.js tab | Shows Node processes |
| `2` | Switch to Ollama tab | Shows loaded models |
| `3` | Switch to OpenClaw tab | Shows service status |
| `4` | Switch to Cron tab | Shows cron jobs |
| `Tab` | Next tab | Cycles through tabs |
| `Shift+Tab` | Previous tab | Cycles backwards |
| `m` | Sort by memory | Default sort for Node.js tab |
| `c` | Sort by CPU | Node.js tab only |
| `s` | Sort by name | Node.js tab only |
| `space` | Pause/resume | Toggles ▶ (running) / ⏸ (paused) |
| `q` | Quit | Exit the application |
| `Ctrl+C` | Force quit | Emergency exit |

## Status Indicators

### Service Status (Top bar)

- 🟢 **●** - Service is online
- 🔴 **●** - Service is offline

### Memory Colors (Process list)

- 🟢 Green - Normal memory usage (< 100MB)
- 🟡 Yellow - High memory usage (> 100MB)

### Cron Job Status Colors

- 🟢 Green - `enabled` or `success`
- 🔴 Red - `failed` or other status

## Real-time Updates

- Dashboard refreshes every **500ms**
- Pause with `space` to freeze the display
- Resume to continue monitoring
- CPU and memory metrics update in real-time
- Process list updates as processes start/stop

## Configuration

Currently configured with:
- **Ollama Port**: 11434 (standard Ollama API port)
- **Refresh Rate**: 500ms
- **Cron Jobs Path**: `~/.openclaw/cron/jobs.json`

Future versions will support:
- Custom configuration files
- Custom Ollama port
- Adjustable refresh rates
- Theme customization

## Tips & Tricks

1. **Monitor high CPU processes**
   - Press `c` to sort by CPU usage
   - Watch the top processes consuming CPU

2. **Find memory hogs**
   - Press `m` (default sort) to see highest memory users
   - Press `s` then `m` to compare relative sizes

3. **Watch Ollama models**
   - Switch to Tab 2 to monitor loaded models
   - See model sizes at a glance
   - Useful when loading/unloading large models

4. **Check service health**
   - Tab 3 shows OpenClaw uptime
   - Green indicator means it's operational
   - Use with Tab 1 to see associated processes

5. **Pause during analysis**
   - Press `space` to pause updates
   - Scroll or analyze metrics without interruption
   - Press `space` again to resume

## Troubleshooting

### OpenClaw shows offline but is actually running
- ai-top checks two ways: HTTP health endpoint + process detection
- If showing offline, check the OpenClaw tab for process information
- Ensure .openclaw directory is in your home path

### Ollama models not showing
- Verify Ollama is running: `ollama list`
- Check if it's listening on port 11434
- Wait a moment for the API to respond (2 second timeout)

### Process detection slow on first run
- Initial CPU usage calculation requires a brief delay
- Subsequent updates are cached and fast
- Subsequent updates refresh in real-time

## Performance Notes

- **Lightweight**: ~10MB binary, <1% CPU idle
- **Fast startup**: <100ms to first display
- **Low memory**: ~15-20MB overhead
- **Responsive**: 500ms refresh rate maintains responsiveness
