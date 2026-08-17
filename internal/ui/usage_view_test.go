package ui

import (
	"testing"
	"time"

	"github.com/tess/ai-top/internal/usage"
)

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{512, "512"},
		{1_000, "1K"},
		{340_000, "340K"},
		{1_200_000, "1.2M"},
	}

	for _, tt := range tests {
		if got := formatTokens(tt.in); got != tt.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// The "~" prefix is the whole point of QuotaSource: Codex's percentage comes
// from the server, Claude's is derived from a local cap, and the table must
// not present them as the same kind of fact.
func TestFormatQuotaDistinguishesEstimatedFromReported(t *testing.T) {
	reported := usage.HarnessUsage{Available: true, QuotaPct: 71, QuotaSource: usage.QuotaReported}
	if got := formatQuota(reported); got != "71%" {
		t.Errorf("reported quota = %q, want 71%%", got)
	}

	estimated := usage.HarnessUsage{Available: true, QuotaPct: 68, QuotaSource: usage.QuotaEstimated}
	if got := formatQuota(estimated); got != "~68%" {
		t.Errorf("estimated quota = %q, want ~68%%", got)
	}
}

// A local harness has no quota to report; "local" says so, where "0%" would
// imply an unused allowance.
func TestFormatQuotaLocalAndUnavailable(t *testing.T) {
	local := usage.HarnessUsage{Available: true, QuotaSource: usage.QuotaNone}
	if got := formatQuota(local); got != "local" {
		t.Errorf("local quota = %q, want local", got)
	}

	absent := usage.HarnessUsage{Available: false, QuotaSource: usage.QuotaNone}
	if got := formatQuota(absent); got != emptyCell {
		t.Errorf("unavailable quota = %q, want %q", got, emptyCell)
	}
}

// Cost zero and cost unknown are different claims.
func TestFormatCostSeparatesFreeFromUnknown(t *testing.T) {
	free := usage.HarnessUsage{Available: true, HasCost: true, CostUSD: 0}
	if got := formatCost(free); got != "$0.00" {
		t.Errorf("free cost = %q, want $0.00", got)
	}

	unknown := usage.HarnessUsage{Available: true, HasCost: false}
	if got := formatCost(unknown); got != emptyCell {
		t.Errorf("unknown cost = %q, want %q", got, emptyCell)
	}
}

func TestFormatResets(t *testing.T) {
	now := time.Date(2026, 8, 17, 18, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		resets time.Time
		want   string
	}{
		{"unset", time.Time{}, ""},
		{"two days out", now.Add(48 * time.Hour), "resets 2d"},
		{"three hours out", now.Add(3 * time.Hour), "resets 3h"},
		{"ten minutes out", now.Add(10 * time.Minute), "resets 10m"},
		{"already passed", now.Add(-time.Hour), "resets now"},
	}

	for _, tt := range tests {
		if got := formatResets(tt.resets, now); got != tt.want {
			t.Errorf("%s: formatResets = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// An unavailable harness must never render a token count, even if some field
// happens to be non-zero: it would be a number we cannot stand behind.
func TestUsageLinesShowsDashForUnavailableHarness(t *testing.T) {
	m := Model{width: 100}
	metrics := fakeMetricsWithUsage([]usage.HarnessUsage{
		{Harness: "OpenClaw", Available: false, QuotaSource: usage.QuotaNone},
	})

	lines := m.usageLines(metrics, 96)
	joined := stripANSI(lines)

	if !containsAll(joined, "OpenClaw", emptyCell) {
		t.Errorf("expected OpenClaw row with %q, got:\n%s", emptyCell, joined)
	}
	if containsAll(joined, "OpenClaw  0 ") {
		t.Error("unavailable harness rendered a zero token count")
	}
	if !containsAll(joined, "reports no usage log") {
		t.Errorf("missing footnote explaining the empty row, got:\n%s", joined)
	}
}

func TestUsageLinesEmptyBeforeFirstCollection(t *testing.T) {
	m := Model{width: 100}
	metrics := fakeMetricsWithUsage(nil)

	lines := m.usageLines(metrics, 96)
	if len(lines) != 1 || !containsAll(stripANSI(lines), "collecting") {
		t.Errorf("expected a collecting placeholder, got %v", lines)
	}
}
