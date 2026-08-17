// Package usage collects token and quota usage from local AI coding harnesses.
//
// Each harness writes its own session logs in its own format; a Source parses
// one of them into the shared HarnessUsage type. Parsing happens once, at the
// boundary — nothing downstream touches raw JSON or database rows.
//
// Every source is expected to work on a machine where its harness was never
// installed: a missing directory yields an unavailable result, not an error.
package usage

import "time"

// QuotaSource records where a quota percentage came from. The distinction
// matters at render time: Codex reports a real server-side percentage, while
// Claude's is derived locally from a token cap and must not be presented with
// the same authority.
type QuotaSource int

const (
	// QuotaNone means no quota figure is available for this harness.
	QuotaNone QuotaSource = iota
	// QuotaReported means the harness logged a server-side quota percentage.
	QuotaReported
	// QuotaEstimated means the percentage was computed locally against a
	// configured token cap, and is only as good as that cap.
	QuotaEstimated
)

// HarnessUsage is one row of the usage view: what a single harness has spent
// over the collection window.
//
// Available and HasCost exist so the view can tell "zero" apart from "unknown"
// without overloading the numeric fields with sentinel values. A harness that
// is not installed, or whose logs could not be read, renders as "—" rather
// than as a confident 0.
type HarnessUsage struct {
	Harness   string
	Available bool
	Model     string

	InputTokens  int64
	OutputTokens int64
	CacheRead    int64
	CacheWrite   int64
	TotalTokens  int64

	CostUSD float64
	HasCost bool

	QuotaPct    float64
	QuotaSource QuotaSource
	QuotaResets time.Time
}

// Source collects usage for a single harness.
//
// Collect reports usage for activity at or after since. Implementations return
// an unavailable HarnessUsage (not an error) when the harness is simply absent
// from this machine; errors are reserved for logs that exist but cannot be read.
type Source interface {
	Name() string
	Collect(since time.Time) (HarnessUsage, error)
}
