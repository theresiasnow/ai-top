package usage

import (
	"path/filepath"
	"testing"
	"time"
)

// Codex reports total_token_usage as a running total for the session, so the
// last event is the session's usage. Summing the events would multiply-count.
func TestScanCodexFileTakesLastCumulativeTotal(t *testing.T) {
	got, err := scanCodexFile(filepath.Join("testdata", "codex-rollout.jsonl"))
	if err != nil {
		t.Fatalf("scanCodexFile: %v", err)
	}

	if got.TotalTokens != 32140 {
		t.Errorf("TotalTokens = %d, want 32140 (last event, not a sum)", got.TotalTokens)
	}
	if got.InputTokens != 31500 {
		t.Errorf("InputTokens = %d, want 31500", got.InputTokens)
	}
	if got.OutputTokens != 640 {
		t.Errorf("OutputTokens = %d, want 640", got.OutputTokens)
	}
}

// info is null on some token_count events; those must be skipped without
// clobbering a previously read total.
func TestScanCodexFileToleratesNullInfo(t *testing.T) {
	got, err := scanCodexFile(filepath.Join("testdata", "codex-rollout.jsonl"))
	if err != nil {
		t.Fatalf("null info aborted scan: %v", err)
	}
	if got.TotalTokens == 0 {
		t.Error("null-info event wiped the totals")
	}
}

// Unlike Claude, Codex logs a real server-side quota percentage.
func TestScanCodexFileReadsReportedQuota(t *testing.T) {
	got, err := scanCodexFile(filepath.Join("testdata", "codex-rollout.jsonl"))
	if err != nil {
		t.Fatalf("scanCodexFile: %v", err)
	}

	if got.QuotaSource != QuotaReported {
		t.Errorf("QuotaSource = %v, want QuotaReported", got.QuotaSource)
	}
	if got.QuotaPct != 71.0 {
		t.Errorf("QuotaPct = %v, want 71.0 (latest event)", got.QuotaPct)
	}

	wantReset := time.Unix(1787246407, 0)
	if !got.QuotaResets.Equal(wantReset) {
		t.Errorf("QuotaResets = %v, want %v", got.QuotaResets, wantReset)
	}
}

// Regression: running the app with the default 5h token window showed Codex as
// "local" with 0 tokens, hiding a real reported quota. Codex's quota covers 7
// days, so it must be searched over a longer lookback than the token window —
// an idle afternoon is not evidence that the quota is unknown.
func TestCodexQuotaSurvivesShortTokenWindow(t *testing.T) {
	src := NewCodexSource(DefaultCodexRoot())

	// A one-minute token window: almost certainly no session activity, yet the
	// quota should still be found.
	got, err := src.Collect(time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if !got.Available {
		t.Skip("Codex not installed on this machine")
	}
	if got.QuotaSource != QuotaReported {
		t.Errorf("quota lost with a short token window: QuotaSource = %v, want QuotaReported", got.QuotaSource)
	}
}

func TestCodexSourceMissingRoot(t *testing.T) {
	src := NewCodexSource(filepath.Join("testdata", "nonexistent-root"))
	got, err := src.Collect(time.Now().Add(-DefaultWindow))
	if err != nil {
		t.Fatalf("Collect on missing root should not error: %v", err)
	}
	if got.Available {
		t.Error("missing root should report Available=false")
	}
}
