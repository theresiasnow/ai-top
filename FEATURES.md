# ai-top - Features & Usage

## What It Monitors

### 🔍 System Metrics
- **CPU Usage** - Real-time percentage and core count
- **Memory** - Used/Total with percentage
- **System Load** - Historical context

### 📊 Services
- **OpenClaw** - Status indicator (online/offline)
- **Ollama** - Status + loaded models count
- **OpenClaw Processes** - Node.js processes in .openclaw directory

### 📈 Process Monitoring
- **Node.js Processes** - All running Node processes
- **Ollama Processes** - Ollama system processes
- **Custom Filtering** - View only specific process types

### ⏱️ Cron Jobs (OpenClaw)
- **Status** - From .openclaw/cron/jobs.json
- **Schedule** - Cron expression
- **Last Run** - Timestamp of last execution
- **Execution Status** - Success/failed/running

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| **q** | Quit |
| **space** | Pause/Resume updates |
| **s** | Sort by Name |
| **c** | Sort by CPU |
| **m** | Sort by Memory |
| **Tab** | Switch between tabs |
| **Shift+Tab** | Previous tab |

## Visual Design

- **btop-inspired** - Box drawing characters, colored output
- **Real-time updates** - 500ms-1s refresh intervals
- **Color-coded** - Green (good), Yellow (warning), Red (critical)
- **Multi-panel** - System stats, services, process list all visible

## Launch

```bash
~/projects/private/ai-top/bin/ai-top
```

Or create a symlink:
```bash
ln -s ~/projects/private/ai-top/bin/ai-top /usr/local/bin/ai-top
ai-top
```

## Performance

- **Binary Size**: 9.8 MB (single static binary)
- **Memory Overhead**: ~20 MB
- **Startup Time**: <100ms
- **CPU Usage**: <1% at rest
- **Dependencies**: None (statically compiled)
