# ai-top

A lightweight TUI (Text User Interface) monitor for AI development environments. Real-time tracking of:
- **Local LLM models** (Ollama and others)
- **AI services** (OpenClaw, API servers)
- **Node.js processes** (memory, CPU, uptime)
- **Cron jobs** and background tasks

Like `top` or `btop`, but designed for AI developers. Single binary, minimal dependencies, inspired by system monitoring tools.

## Quick Start

```bash
go build -o bin/ai-top ./cmd/ai-top
./bin/ai-top
```

## Features

- [x] TUI dashboard with real-time updates
- [x] Process monitoring (Node.js, Ollama)
- [x] Service status (OpenClaw, Ollama)
- [x] Color-coded status indicators
- [ ] Interactive controls (pause, filter, sort)
- [ ] Cron job integration
- [ ] JSON export

## Keybinds

| Key | Action |
|-----|--------|
| `q` | Quit |
| `space` | Pause/resume |
| `c` | Sort by CPU |
| `m` | Sort by memory |
| `s` | Sort by name |

## Configuration

Edit `.ai-top.conf` in your home directory:

```json
{
  "refresh_rate": 2,
  "ollama_port": 11434,
  "openclaw_port": 3000
}
```
