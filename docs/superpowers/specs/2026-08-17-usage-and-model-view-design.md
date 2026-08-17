# Usage tracking and loaded-model view

**Status:** approved design, not yet implemented
**Date:** 2026-08-17

Two features for ai-top:

1. Token/usage tracking per harness (Claude Code, Codex, OpenCode, OpenClaw)
2. A merged "loaded model / VRAM / context / process owner" view across Ollama and omlx

## Verified data sources

Every claim below was checked against real files on this machine on 2026-08-17.
Anything not verified is marked as such.

| Source | Path | Tokens | Quota % | Model |
|---|---|---|---|---|
| Codex | `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` | yes | yes | yes |
| Claude Code | `~/.claude/projects/*/*.jsonl` | yes | no | yes |
| OpenCode | `~/.local/share/opencode/opencode.db` (sqlite) | yes, plus `cost` | n/a (local) | yes |
| Ollama | `GET /api/ps` | n/a | n/a | yes, plus context/quant |
| OpenClaw | `~/.openclaw/` | **no** | **no** | **no** |

### Codex

`rollout-*.jsonl` contains `token_count` events:

```json
{"type": "token_count",
 "info": {"total_token_usage": {"input_tokens": 22388, "cached_input_tokens": 11008,
          "cache_write_input_tokens": 0, "output_tokens": 221,
          "reasoning_output_tokens": 77, "total_tokens": 22609}},
 "rate_limits": {"primary": {"used_percent": 14.0, "window_minutes": 10080,
                             "resets_at": 1787246407}}}
```

`info` is sometimes `null` — the parser must tolerate it and skip. `used_percent`
is a real server-reported quota, the only one of the four harnesses that has one.
`window_minutes: 10080` is a 7-day window; `resets_at` is a Unix timestamp.

### Claude Code

Each assistant message carries `message.usage` and `message.model`:

```json
{"model": "<synthetic>",
 "usage": {"input_tokens": 0, "output_tokens": 0,
           "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}}
```

There is **no quota percentage anywhere in these logs.** Records with
`model: "<synthetic>"` are not real API calls and must be excluded from totals.

### OpenCode

Table `session`, columns: `model`, `cost` (real, USD), `tokens_input`,
`tokens_output`, `tokens_reasoning`, `tokens_cache_read`, `tokens_cache_write`,
`time_created`, `time_updated`.

Two parsing traps, both verified:

- `model` is **JSON, not a bare string**: `{"id":"qwen3-coder:30b","providerID":"ollama"}`.
  Parse it; display `id`.
- `time_updated` is **milliseconds** since epoch (`1786988056178` = 2026-08-17 17:34).

Open the database read-only and immutable so ai-top can never disturb a live
OpenCode session:

```
file:...opencode.db?mode=ro&immutable=1
```

### Ollama

`GET /api/ps` already returns everything the model view needs. Today's parser in
`internal/monitor/ollama.go` reads only four fields and discards the rest:

```json
{"name": "nomic-embed-text:latest", "size": 370031984, "size_vram": 370031984,
 "expires_at": "2026-08-17T20:41:56.560509+02:00",
 "context_length": 2048,
 "details": {"parameter_size": "137M", "quantization_level": "F16"}}
```

`context_length`, `quantization_level`, and `parameter_size` come free — they are
already in the response body.

### OpenClaw — no usage data

`~/.openclaw/` has no usage log. The only files matching `input_tokens` are
crash dumps under `logs/stability/*uncaught_exception.json`. OpenClaw therefore
appears in the usage table as a row with `running / PID / uptime` (data ai-top
already collects) and a literal `—` in the token and quota columns.

**Fabricating a number here would be worse than an empty cell.** If OpenClaw
gains a usage log later, it plugs in as another `Source` with no view changes.

## Architecture

New package `internal/usage/`. Deliberately not part of `internal/monitor/`:
this is filesystem and sqlite scanning on a different cadence, with different
failure modes, than the 2-second process/HTTP polling `monitor` does.

```
usage/harness.go    HarnessUsage type + Source interface
usage/claude.go     tails ~/.claude/projects/*/*.jsonl
usage/codex.go      tails ~/.codex/sessions/**/rollout-*.jsonl
usage/opencode.go   sqlite, read-only + immutable
usage/window.go     rolling-window aggregation + percent-of-cap
```

### The Source interface

```go
type Source interface {
    Name() string
    Collect(since time.Time) (HarnessUsage, error)
}

type HarnessUsage struct {
    Harness      string
    Available    bool          // false = could not read; render "—", never 0
    Model        string
    InputTokens  int64
    OutputTokens int64
    CacheRead    int64
    CacheWrite   int64
    TotalTokens  int64
    CostUSD      float64       // only OpenCode reports this; 0 elsewhere
    HasCost      bool
    QuotaPct     float64       // only Codex reports a real one
    QuotaSource  QuotaSource   // Reported | Estimated | None
    QuotaResets  time.Time
}
```

`Available` and `HasCost` exist so the view can distinguish "zero" from
"unknown" without sentinel values. `QuotaSource` is what lets the UI mark
Claude's number as derived rather than authoritative.

**Parse at the boundary.** Each source converts its raw JSON/sqlite rows into
`HarnessUsage` exactly once. No view code touches raw JSON.

### Rolling window and Claude's estimated percentage

Claude has no server-reported quota, so its percentage is computed locally:
sum tokens over a rolling 5-hour window, divide by a configurable cap, clamp to 100.

The default cap ships as 1.8M tokens/5h. **This number is a placeholder, not a
measurement** — it was not derived from any observed limit. It must be
user-configurable, and the first implementation task is to calibrate it against
actual usage before the default is trusted.

This is an **estimate**, and the UI says so — `~68%` with a footnote, versus
Codex's bare `71%`. The distinction is not cosmetic: a derived number presented
as authoritative is the kind of thing that gets trusted and then acts wrong.

### Cadence and incremental reads

Usage scanning runs on its own 30-second ticker, not the 2-second one. Reading
hundreds of jsonl files every 2 seconds would waste more CPU than the rest of
ai-top combined.

Reads are incremental: remember `(path, size, mtime)` per file and read only
appended bytes. Files older than the window are skipped by mtime before opening.

## UI: two new tabs

`internal/ui/model.go` is already 1346 lines. New render functions go in their
own files — `ui/usage_view.go` and `ui/models_view.go` — rather than growing it
further. Only the tab constants and the dispatch switch change in `model.go`.

### Tab 7 — Usage

```
HARNESS    QUOTA           MODEL             TOKENS (5h)   COST
Claude     ~68% ███████░░  Opus 5            1.2M          —
Codex       71% ███████░░  GPT-5.6           340K          —
OpenCode   local           qwen3-coder:30b   890K          $0.00
OpenClaw   —               —                 —             —
```

Codex additionally shows `resets in 2d` from `resets_at`. The `~` prefix marks
an estimated percentage.

### Tab 8 — Models (Ollama + omlx merged)

```
MODEL              BACKEND  VRAM    RAM   CTX    QUANT    OWNER       TTL
qwen3-coder:30b    ollama   18.6G   0     32K    Q4_K_M   ollama/501  4m12s
nomic-embed-text   ollama   370M    0     2048   F16      ollama/501  3m
```

OWNER comes from the existing `OllamaProcess` / `OmlxProcess` in `SystemMetrics`
(`ProcessInfo` already carries PID and user). TTL is derived from `ExpiresAt`.

## Changes to existing code

- `monitor/ollama.go` — add `ContextLength`, `Quantization`, `ParameterSize` to
  `ollamaPsResponse` and `RunningModel`. Fields already arrive in the response.
- `monitor/monitor.go` — add `Usage []HarnessUsage` to `SystemMetrics`, with a
  deep copy in `GetMetrics` following the existing slice-copy pattern.
- `ui/model.go` — two tab constants, dispatch, and `availableTabs()`.

## Testing

`internal/usage/testdata/` holds anonymized fixtures: one Claude usage line, one
Codex `token_count` (and one with `info: null`), one Claude `<synthetic>` line
that must be excluded, and a small generated sqlite file.

Window/percentage math in `window.go` is a pure function — table tests.

Following the repo's existing convention, a missing harness yields an empty
result, not an error: ai-top must run fine when none of these tools are installed.

## Explicitly out of scope (YAGNI)

- **USD cost for Claude/Codex.** Requires a hardcoded price table that goes stale
  and silently misreports. OpenCode reports `cost` directly; everyone else shows `—`.
- **Historical graphs over time.** The window aggregation makes this possible
  later; nothing in this design forecloses it.
