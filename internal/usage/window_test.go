package usage

import (
	"math"
	"testing"
	"time"
)

func TestPercentOfCap(t *testing.T) {
	tests := []struct {
		name   string
		tokens int64
		cap    int64
		want   float64
	}{
		{"zero tokens", 0, 1_800_000, 0},
		{"half the cap", 900_000, 1_800_000, 50},
		{"exactly at cap", 1_800_000, 1_800_000, 100},
		{"over cap clamps to 100", 3_000_000, 1_800_000, 100},
		{"negative tokens clamp to 0", -5, 1_800_000, 0},
		{"zero cap yields 0, not NaN", 500, 0, 0},
		{"negative cap yields 0", 500, -1, 0},
	}

	for _, tt := range tests {
		got := PercentOfCap(tt.tokens, tt.cap)
		if math.Abs(got-tt.want) > 0.001 {
			t.Errorf("%s: PercentOfCap(%d, %d) = %v, want %v",
				tt.name, tt.tokens, tt.cap, got, tt.want)
		}
	}
}

func TestWindowStart(t *testing.T) {
	now := time.Date(2026, 8, 17, 18, 0, 0, 0, time.UTC)
	got := WindowStart(now, 5*time.Hour)
	want := time.Date(2026, 8, 17, 13, 0, 0, 0, time.UTC)

	if !got.Equal(want) {
		t.Errorf("WindowStart = %v, want %v", got, want)
	}
}

func TestSumTokensIgnoresUnavailable(t *testing.T) {
	// An unavailable harness must contribute nothing. If it leaked in as a
	// zero-valued row it would be harmless here, but the same Available flag
	// drives whether the view prints "—" or "0", so the invariant is worth pinning.
	rows := []HarnessUsage{
		{Harness: "claude", Available: true, TotalTokens: 1000},
		{Harness: "openclaw", Available: false, TotalTokens: 999999},
		{Harness: "codex", Available: true, TotalTokens: 500},
	}

	got := SumTokens(rows)
	if got != 1500 {
		t.Errorf("SumTokens = %d, want 1500", got)
	}
}
