package usage

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// syntheticModel marks log records that are not real API calls. They carry a
// usage block like any other record, so they must be filtered by model name or
// they inflate every total computed from these logs.
const syntheticModel = "<synthetic>"

// maxLineBytes bounds a single log line. Session logs can contain very large
// pasted content, and an unbounded scanner would happily read it all into memory.
const maxLineBytes = 4 * 1024 * 1024

// claudeRecord is the subset of a Claude session log line this package needs.
type claudeRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Message   struct {
		Model string `json:"model"`
		Usage *struct {
			InputTokens         int64 `json:"input_tokens"`
			OutputTokens        int64 `json:"output_tokens"`
			CacheCreationTokens int64 `json:"cache_creation_input_tokens"`
			CacheReadTokens     int64 `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// ClaudeSource reads token usage from Claude Code session logs.
//
// Claude's logs report no quota, so any percentage this source produces is
// derived from cap and always marked QuotaEstimated.
type ClaudeSource struct {
	root string // ~/.claude/projects
	cap  int64
}

// NewClaudeSource creates a source reading session logs under root.
func NewClaudeSource(root string, cap int64) *ClaudeSource {
	return &ClaudeSource{root: root, cap: cap}
}

// DefaultClaudeRoot returns the standard Claude Code projects directory.
func DefaultClaudeRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects")
}

func (c *ClaudeSource) Name() string { return "Claude" }

// Collect sums usage across every session log modified within the window.
//
// A missing root means Claude Code is not installed here; that is reported as
// an unavailable row, not an error.
func (c *ClaudeSource) Collect(since time.Time) (HarnessUsage, error) {
	out := HarnessUsage{Harness: c.Name(), QuotaSource: QuotaEstimated}

	if c.root == "" {
		return out, nil
	}
	if _, err := os.Stat(c.root); err != nil {
		return out, nil
	}

	paths, err := filepath.Glob(filepath.Join(c.root, "*", "*.jsonl"))
	if err != nil {
		return out, err
	}

	out.Available = true
	for _, path := range paths {
		// Skip files untouched since the window opened without opening them.
		// This is what keeps a scan over hundreds of sessions cheap.
		if info, err := os.Stat(path); err == nil && info.ModTime().Before(since) {
			continue
		}

		fileUsage, err := scanClaudeFile(path, since)
		if err != nil {
			continue // an unreadable session must not sink the whole scan
		}

		out.InputTokens += fileUsage.InputTokens
		out.OutputTokens += fileUsage.OutputTokens
		out.CacheRead += fileUsage.CacheRead
		out.CacheWrite += fileUsage.CacheWrite
		if fileUsage.Model != "" {
			out.Model = fileUsage.Model
		}
	}

	out.TotalTokens = out.InputTokens + out.OutputTokens
	out.QuotaPct = PercentOfCap(out.TotalTokens, c.cap)
	return out, nil
}

// scanClaudeFile sums usage from one session log, counting only real API calls
// at or after since. Malformed lines are skipped rather than fatal: these logs
// are appended live and can be read mid-write.
func scanClaudeFile(path string, since time.Time) (HarnessUsage, error) {
	f, err := os.Open(path)
	if err != nil {
		return HarnessUsage{}, err
	}
	defer f.Close()

	var out HarnessUsage
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)

	for scanner.Scan() {
		var rec claudeRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		if rec.Message.Usage == nil || rec.Message.Model == syntheticModel {
			continue
		}
		if !rec.Timestamp.IsZero() && rec.Timestamp.Before(since) {
			continue
		}

		u := rec.Message.Usage
		out.InputTokens += u.InputTokens
		out.OutputTokens += u.OutputTokens
		out.CacheRead += u.CacheReadTokens
		out.CacheWrite += u.CacheCreationTokens
		if rec.Message.Model != "" {
			out.Model = rec.Message.Model
		}
	}
	// A truncated final line is normal for a live log; keep what we counted.
	if err := scanner.Err(); err != nil {
		return out, nil
	}

	out.TotalTokens = out.InputTokens + out.OutputTokens
	return out, nil
}
