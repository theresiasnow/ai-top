package usage

import (
	"path/filepath"
	"testing"
	"time"
)

// windowStart is before the in-window fixture records (17:00 onward) and after
// the stale 2026-01-01 one.
var claudeSince = time.Date(2026, 8, 17, 16, 0, 0, 0, time.UTC)

func TestScanClaudeFile(t *testing.T) {
	got, err := scanClaudeFile(filepath.Join("testdata", "claude-session.jsonl"), claudeSince)
	if err != nil {
		t.Fatalf("scanClaudeFile: %v", err)
	}

	// Only the two real in-window records count: 1200+800 input, 340+160 output.
	if got.InputTokens != 2000 {
		t.Errorf("InputTokens = %d, want 2000", got.InputTokens)
	}
	if got.OutputTokens != 500 {
		t.Errorf("OutputTokens = %d, want 500", got.OutputTokens)
	}
	if got.CacheRead != 12000 {
		t.Errorf("CacheRead = %d, want 12000", got.CacheRead)
	}
	if got.CacheWrite != 500 {
		t.Errorf("CacheWrite = %d, want 500", got.CacheWrite)
	}
	if got.Model != "claude-opus-5" {
		t.Errorf("Model = %q, want claude-opus-5", got.Model)
	}
}

// Records with model "<synthetic>" are not real API calls. Counting them would
// inflate the totals that Claude's estimated percentage is computed from.
func TestScanClaudeFileExcludesSynthetic(t *testing.T) {
	got, err := scanClaudeFile(filepath.Join("testdata", "claude-session.jsonl"), claudeSince)
	if err != nil {
		t.Fatalf("scanClaudeFile: %v", err)
	}

	if got.InputTokens >= 99999 {
		t.Errorf("synthetic record leaked into totals: InputTokens = %d", got.InputTokens)
	}
	if got.Model == "<synthetic>" {
		t.Error("Model reported as <synthetic>")
	}
}

func TestScanClaudeFileExcludesOutOfWindow(t *testing.T) {
	got, err := scanClaudeFile(filepath.Join("testdata", "claude-session.jsonl"), claudeSince)
	if err != nil {
		t.Fatalf("scanClaudeFile: %v", err)
	}

	if got.InputTokens == 9777 || got.OutputTokens == 8277 {
		t.Errorf("stale 2026-01-01 record counted: in=%d out=%d",
			got.InputTokens, got.OutputTokens)
	}
}

// A corrupt line must not abort the scan — session logs are appended live and
// can be caught mid-write.
func TestScanClaudeFileToleratesCorruptLines(t *testing.T) {
	got, err := scanClaudeFile(filepath.Join("testdata", "claude-session.jsonl"), claudeSince)
	if err != nil {
		t.Fatalf("corrupt line aborted scan: %v", err)
	}
	if got.InputTokens != 2000 {
		t.Errorf("records after corrupt line were dropped: InputTokens = %d", got.InputTokens)
	}
}

// Claude's logs carry no quota figure, so any percentage must be marked as
// derived rather than reported.
func TestClaudeSourceQuotaIsEstimated(t *testing.T) {
	src := NewClaudeSource(filepath.Join("testdata", "nonexistent-root"), DefaultClaudeCap)
	got, err := src.Collect(claudeSince)
	if err != nil {
		t.Fatalf("Collect on missing root should not error: %v", err)
	}
	if got.Available {
		t.Error("missing root should report Available=false")
	}
	if got.QuotaSource == QuotaReported {
		t.Error("Claude quota must never be marked as reported")
	}
}
