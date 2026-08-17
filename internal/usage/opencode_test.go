package usage

import (
	"path/filepath"
	"testing"
	"time"
)

// OpenCode stores model as a JSON object, not a bare string. Rendering the raw
// column would put `{"id":"qwen3-coder:30b","providerID":"ollama"}` in the table.
func TestParseOpenCodeModel(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"json object", `{"id":"qwen3-coder:30b","providerID":"ollama"}`, "qwen3-coder:30b"},
		{"empty string", "", ""},
		{"null column", "null", ""},
		{"malformed json falls back to raw", "qwen3-coder:30b", "qwen3-coder:30b"},
	}

	for _, tt := range tests {
		if got := parseOpenCodeModel(tt.in); got != tt.want {
			t.Errorf("%s: parseOpenCodeModel(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
		}
	}
}

// time_updated is milliseconds since epoch. Treating it as seconds would put
// every row ~56000 years in the future and silently empty the window.
func TestUnixMillis(t *testing.T) {
	ts := time.Date(2026, 8, 17, 17, 34, 16, 0, time.UTC)
	got := unixMillis(ts)

	if got != 1786988056000 {
		t.Errorf("unixMillis = %d, want 1786988056000", got)
	}
}

func TestOpenCodeSourceMissingDB(t *testing.T) {
	src := NewOpenCodeSource(filepath.Join("testdata", "nonexistent.db"))
	got, err := src.Collect(time.Now().Add(-DefaultWindow))
	if err != nil {
		t.Fatalf("Collect on missing db should not error: %v", err)
	}
	if got.Available {
		t.Error("missing db should report Available=false")
	}
}

// OpenCode is the only harness reporting real cost, so HasCost must be set
// even when the amount is zero (local models are free but still measured).
func TestOpenCodeReportsCostAvailability(t *testing.T) {
	src := NewOpenCodeSource(DefaultOpenCodeDB())
	got, err := src.Collect(time.Now().Add(-DefaultWindow))
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if !got.Available {
		t.Skip("OpenCode not installed on this machine")
	}
	if !got.HasCost {
		t.Error("HasCost should be true when the db was read")
	}
	if got.QuotaSource != QuotaNone {
		t.Errorf("QuotaSource = %v, want QuotaNone (local harness has no quota)", got.QuotaSource)
	}
	t.Logf("model=%q total=%d cost=%.4f", got.Model, got.TotalTokens, got.CostUSD)
}
