package ui

import (
	"regexp"
	"strings"

	"github.com/tess/ai-top/internal/monitor"
	"github.com/tess/ai-top/internal/usage"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI joins rendered lines and removes styling, so assertions can match
// on the text a user actually reads.
func stripANSI(lines []string) string {
	return ansiPattern.ReplaceAllString(strings.Join(lines, "\n"), "")
}

func containsAll(haystack string, needles ...string) bool {
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			return false
		}
	}
	return true
}

func fakeMetricsWithUsage(rows []usage.HarnessUsage) *monitor.SystemMetrics {
	return &monitor.SystemMetrics{Usage: rows}
}
