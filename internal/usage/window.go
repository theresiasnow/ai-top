package usage

import "time"

// DefaultWindow is the rolling window over which token usage is summed.
const DefaultWindow = 5 * time.Hour

// DefaultClaudeCap is the token ceiling Claude usage is measured against to
// produce a percentage.
//
// This number is a placeholder, not a measurement — it was not derived from any
// observed limit, because Claude's logs do not report one. It is the reason
// Claude's percentage renders as an estimate: the numerator is real, the
// denominator is a guess. Calibrate against real usage before trusting it.
const DefaultClaudeCap int64 = 1_800_000

// WindowStart returns the beginning of the rolling window ending at now.
func WindowStart(now time.Time, window time.Duration) time.Time {
	return now.Add(-window)
}

// PercentOfCap expresses tokens as a percentage of cap, clamped to [0, 100].
// A non-positive cap yields 0 rather than a division by zero.
func PercentOfCap(tokens, cap int64) float64 {
	if cap <= 0 || tokens <= 0 {
		return 0
	}
	pct := float64(tokens) / float64(cap) * 100
	if pct > 100 {
		return 100
	}
	return pct
}

// SumTokens totals tokens across harnesses, skipping unavailable ones so that
// a harness with no readable logs cannot inflate the total.
func SumTokens(rows []HarnessUsage) int64 {
	var total int64
	for _, r := range rows {
		if r.Available {
			total += r.TotalTokens
		}
	}
	return total
}
