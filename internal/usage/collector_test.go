package usage

import (
	"testing"
	"time"
)

// stubSource lets the collector be tested without touching the filesystem.
type stubSource struct {
	name string
	row  HarnessUsage
	err  error
}

func (s stubSource) Name() string { return s.name }
func (s stubSource) Collect(time.Time) (HarnessUsage, error) {
	return s.row, s.err
}

func TestCollectorRefreshGathersEveryHarness(t *testing.T) {
	c := &Collector{
		window: DefaultWindow,
		sources: []Source{
			stubSource{name: "A", row: HarnessUsage{Harness: "A", Available: true, TotalTokens: 10}},
			stubSource{name: "B", row: HarnessUsage{Harness: "B", Available: true, TotalTokens: 20}},
		},
	}
	c.Refresh()

	rows := c.Rows()
	if len(rows) != 2 {
		t.Fatalf("len(Rows()) = %d, want 2", len(rows))
	}
	if SumTokens(rows) != 30 {
		t.Errorf("SumTokens = %d, want 30", SumTokens(rows))
	}
}

// A source that errors must still occupy a row: a harness that is present but
// unreadable is information, and dropping it would silently shrink the table.
func TestCollectorKeepsRowForFailingSource(t *testing.T) {
	c := &Collector{
		window: DefaultWindow,
		sources: []Source{
			stubSource{name: "Broken", err: errFake},
			stubSource{name: "Fine", row: HarnessUsage{Harness: "Fine", Available: true, TotalTokens: 5}},
		},
	}
	c.Refresh()

	rows := c.Rows()
	if len(rows) != 2 {
		t.Fatalf("len(Rows()) = %d, want 2", len(rows))
	}
	if rows[0].Harness != "Broken" || rows[0].Available {
		t.Errorf("failing source row = %+v, want Broken/unavailable", rows[0])
	}
}

// Rows must hand out a copy; a caller mutating it must not corrupt the cache.
func TestCollectorRowsReturnsCopy(t *testing.T) {
	c := &Collector{
		window:  DefaultWindow,
		sources: []Source{stubSource{name: "A", row: HarnessUsage{Harness: "A", Available: true, TotalTokens: 10}}},
	}
	c.Refresh()

	rows := c.Rows()
	rows[0].TotalTokens = 99999

	if again := c.Rows(); again[0].TotalTokens != 10 {
		t.Errorf("cache mutated through returned slice: got %d, want 10", again[0].TotalTokens)
	}
}

// OpenClaw has no usage log, so it must never claim availability — an empty
// cell is honest where a zero would read as "used nothing".
func TestOpenClawNeverReportsFabricatedUsage(t *testing.T) {
	src := NewOpenClawSource()
	got, err := src.Collect(time.Now().Add(-DefaultWindow))
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if got.Available {
		t.Error("OpenClaw must report Available=false; it has no usage log")
	}
	if got.TotalTokens != 0 || got.QuotaSource != QuotaNone {
		t.Errorf("OpenClaw reported usage it cannot know: %+v", got)
	}
	if got.Harness != "OpenClaw" {
		t.Errorf("Harness = %q, want OpenClaw", got.Harness)
	}
}

var errFake = fakeErr("source unavailable")

type fakeErr string

func (e fakeErr) Error() string { return string(e) }
