package usage

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// codexRecord is the subset of a Codex rollout log line this package needs.
//
// info is a pointer because Codex emits token_count events with a null info
// block; those carry no usage and must be skipped rather than read as zeroes.
type codexRecord struct {
	Payload struct {
		Type string `json:"type"`
		Info *struct {
			TotalTokenUsage struct {
				InputTokens       int64 `json:"input_tokens"`
				CachedInputTokens int64 `json:"cached_input_tokens"`
				CacheWriteTokens  int64 `json:"cache_write_input_tokens"`
				OutputTokens      int64 `json:"output_tokens"`
				ReasoningTokens   int64 `json:"reasoning_output_tokens"`
				TotalTokens       int64 `json:"total_tokens"`
			} `json:"total_token_usage"`
		} `json:"info"`
		RateLimits *struct {
			Primary *struct {
				UsedPercent   float64 `json:"used_percent"`
				WindowMinutes int     `json:"window_minutes"`
				ResetsAt      int64   `json:"resets_at"`
			} `json:"primary"`
		} `json:"rate_limits"`
	} `json:"payload"`
}

// CodexSource reads token usage and quota from Codex rollout logs.
//
// Codex is the only harness here that logs a real server-side quota, so its
// percentage is reported rather than derived.
type CodexSource struct {
	root string // ~/.codex/sessions
}

// NewCodexSource creates a source reading rollout logs under root.
func NewCodexSource(root string) *CodexSource {
	return &CodexSource{root: root}
}

// DefaultCodexRoot returns the standard Codex sessions directory.
func DefaultCodexRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex", "sessions")
}

func (c *CodexSource) Name() string { return "Codex" }

// quotaLookback bounds the search for the newest quota reading.
//
// Codex's quota covers a 7-day window, so a session inside the caller's much
// shorter token window is not required for the quota to be current — without
// this, an idle afternoon makes a real 71% quota silently render as "local".
const quotaLookback = 8 * 24 * time.Hour

// Collect sums usage across rollout logs touched within the window.
//
// Token totals add up across sessions, but the quota percentage does not: it
// is an account-wide figure repeated in every session, so the most recently
// observed value wins — and it is searched over a longer lookback than tokens,
// since the quota window outlives any single token window.
func (c *CodexSource) Collect(since time.Time) (HarnessUsage, error) {
	out := HarnessUsage{Harness: c.Name()}

	if c.root == "" {
		return out, nil
	}
	if _, err := os.Stat(c.root); err != nil {
		return out, nil
	}

	// Rollout logs live under sessions/YYYY/MM/DD/.
	paths, err := filepath.Glob(filepath.Join(c.root, "*", "*", "*", "rollout-*.jsonl"))
	if err != nil {
		return out, err
	}

	out.Available = true
	quotaCutoff := time.Now().Add(-quotaLookback)
	var newestQuota time.Time

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		inTokenWindow := !info.ModTime().Before(since)
		inQuotaWindow := info.ModTime().After(quotaCutoff)
		if !inTokenWindow && !inQuotaWindow {
			continue
		}

		fileUsage, err := scanCodexFile(path)
		if err != nil {
			continue
		}

		if inTokenWindow {
			out.InputTokens += fileUsage.InputTokens
			out.OutputTokens += fileUsage.OutputTokens
			out.CacheRead += fileUsage.CacheRead
			out.CacheWrite += fileUsage.CacheWrite
			out.TotalTokens += fileUsage.TotalTokens
		}

		// Quota is account-wide, not per session: keep the freshest reading
		// instead of accumulating.
		if fileUsage.QuotaSource == QuotaReported && info.ModTime().After(newestQuota) {
			newestQuota = info.ModTime()
			out.QuotaPct = fileUsage.QuotaPct
			out.QuotaResets = fileUsage.QuotaResets
			out.QuotaSource = QuotaReported
		}
	}

	return out, nil
}

// scanCodexFile reads one rollout log.
//
// total_token_usage is cumulative for the session, so the last token_count
// event holds the session total — summing the events would count the same
// tokens repeatedly.
func scanCodexFile(path string) (HarnessUsage, error) {
	f, err := os.Open(path)
	if err != nil {
		return HarnessUsage{}, err
	}
	defer f.Close()

	var out HarnessUsage
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)

	for scanner.Scan() {
		var rec codexRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		if rec.Payload.Type != "token_count" {
			continue
		}

		if rec.Payload.Info != nil {
			u := rec.Payload.Info.TotalTokenUsage
			out.InputTokens = u.InputTokens
			out.OutputTokens = u.OutputTokens
			out.CacheRead = u.CachedInputTokens
			out.CacheWrite = u.CacheWriteTokens
			out.TotalTokens = u.TotalTokens
		}

		if rl := rec.Payload.RateLimits; rl != nil && rl.Primary != nil {
			out.QuotaPct = rl.Primary.UsedPercent
			out.QuotaSource = QuotaReported
			if rl.Primary.ResetsAt > 0 {
				out.QuotaResets = time.Unix(rl.Primary.ResetsAt, 0)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return out, nil
	}

	return out, nil
}
