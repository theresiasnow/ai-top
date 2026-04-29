# 🚀 ai-top

A modern, btop-inspired TUI monitor for AI development environments. Built in Go for speed and efficiency.

![ai-top screenshot](docs/screenshot.gif)

## Features

✅ **Real-time Process Monitoring**
- Node.js process tracking with memory and CPU metrics
- Ollama process monitoring
- OpenClaw service detection (online/offline status)
- Memory-sorted process list by default

✅ **AI Model Management**
- Display loaded Ollama models with their sizes
- Real-time model status tracking
- Online/offline status indicator

✅ **Service Status Dashboard**
- OpenClaw status (online/offline with uptime and memory)
- Ollama status (online/offline with model count)
- Color-coded status indicators

✅ **Cron Job Monitoring**
- Display .openclaw cron jobs from `~/.openclaw/cron/jobs.json`
- Job schedule and status tracking
- Last run information

✅ **Interactive Controls**
- Tab navigation: [1] Node.js [2] Ollama [3] OpenClaw [4] Cron
- Sort by: [m]emory (default), [c]pu, [s]ort name
- [space] to pause/resume updates
- [q] to quit

✅ **Modern UI**
- TUI built with Charm.sh (bubbletea + lipgloss)
- btop-inspired dark color scheme
- Clean, responsive layout
- Real-time updates (500ms refresh rate)

## Installation

### Build from source

```bash
git clone https://github.com/theresiasnow/ai-top.git
cd ai-top
make install
```

Or run directly:

```bash
make run
```

## Usage

```bash
ai-top
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `1` | Show Node.js processes |
| `2` | Show Ollama models |
| `3` | Show OpenClaw status |
| `4` | Show cron jobs |
| `m` | Sort by memory (default) |
| `c` | Sort by CPU |
| `s` | Sort by name |
| `space` | Pause/resume updates |
| `q` | Quit |

## Requirements

- Go 1.19+
- macOS or Linux (tested on macOS)

## Dependencies

- `github.com/shirou/gopsutil/v3` - System and process metrics
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling

## Architecture

- **cmd/ai-top/main.go** - Entry point with bubbletea event loop
- **internal/monitor/** - Core monitoring system
  - `monitor.go` - Main monitor orchestrator
  - `processes.go` - Process detection and metrics
  - `ollama.go` - Ollama API integration
  - `openclaw.go` - OpenClaw service detection
  - `cron.go` - Cron job parsing
  - `system.go` - System metrics (CPU, memory)
- **internal/ui/model.go** - TUI implementation with bubbletea

## Testing

Run the test suite:

```bash
go test -v ./internal/monitor/...
```

Sample output shows:
- 67+ Node processes detected
- 4 Ollama models loaded
- OpenClaw detection working
- Memory formatting validation

## Performance

- **Binary size**: ~10 MB (static binary, no dependencies)
- **Startup time**: <100ms
- **Memory overhead**: ~15-20 MB
- **Refresh rate**: 500ms (configurable)
- **CPU usage**: <1% idle

## Future Enhancements

- [ ] Custom configuration file (~/.ai-toprc)
- [ ] Custom Ollama port configuration
- [ ] JSON export of metrics
- [ ] Sortable columns by click
- [ ] Process filtering options
- [ ] Theme customization

## Known Issues

- OpenClaw detection requires either process-level detection or HTTP health endpoint
- Cron jobs require properly formatted ~/.openclaw/cron/jobs.json

## License

MIT
