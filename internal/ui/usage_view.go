package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/tess/ai-top/internal/monitor"
	"github.com/tess/ai-top/internal/usage"
)

// emptyCell marks a value this machine cannot know. It is deliberately not "0":
// a harness with no usage log has not used zero tokens, it has told us nothing.
const emptyCell = "—"

// formatTokens renders a token count compactly (1.2M, 340K, 512).
func formatTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.0fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// formatQuota renders a quota percentage.
//
// An estimated percentage is prefixed with "~" so it is never mistaken for the
// server-reported figure Codex provides. A harness with no quota concept at all
// renders as "local" rather than as 0%.
func formatQuota(row usage.HarnessUsage) string {
	switch row.QuotaSource {
	case usage.QuotaReported:
		return fmt.Sprintf("%.0f%%", row.QuotaPct)
	case usage.QuotaEstimated:
		return fmt.Sprintf("~%.0f%%", row.QuotaPct)
	default:
		if row.Available {
			return "local"
		}
		return emptyCell
	}
}

// formatResets renders how long until a quota window resets.
func formatResets(resets time.Time, now time.Time) string {
	if resets.IsZero() {
		return ""
	}
	d := resets.Sub(now)
	if d <= 0 {
		return "resets now"
	}
	if d >= 24*time.Hour {
		return fmt.Sprintf("resets %dd", int(d.Hours()/24))
	}
	if d >= time.Hour {
		return fmt.Sprintf("resets %dh", int(d.Hours()))
	}
	return fmt.Sprintf("resets %dm", int(d.Minutes()))
}

// formatCost renders spend, distinguishing "free" from "not reported".
func formatCost(row usage.HarnessUsage) string {
	if !row.HasCost {
		return emptyCell
	}
	return fmt.Sprintf("$%.2f", row.CostUSD)
}

func (m Model) renderUsagePanel(metrics *monitor.SystemMetrics, contentHeight int) string {
	w := max(40, m.width)
	innerW := w - 4

	var sb strings.Builder
	windowLabel := fmt.Sprintf(" %s window ", shortDuration(m.usageWindow()))
	sb.WriteString(boxTop(styleTitle.Render(" Harness Usage "), styleDim.Render(windowLabel), w))
	sb.WriteString("\n")

	for _, line := range m.usageLines(metrics, innerW) {
		if contentHeight <= 0 {
			break
		}
		sb.WriteString(boxLine(line, w))
		sb.WriteString("\n")
		contentHeight--
	}

	sb.WriteString(boxBottom(w))
	return sb.String()
}

// usageLines builds the usage table body.
func (m Model) usageLines(metrics *monitor.SystemMetrics, innerW int) []string {
	rows := metrics.Usage
	if len(rows) == 0 {
		return []string{styleDim.Render("collecting usage…")}
	}

	now := time.Now()
	lines := make([]string, 0, len(rows)+2)

	header := fmt.Sprintf("%-10s %-16s %-18s %10s %8s",
		"HARNESS", "QUOTA", "MODEL", "TOKENS", "COST")
	lines = append(lines, styleDim.Render(truncate(header, innerW)))

	for _, row := range rows {
		quota := formatQuota(row)
		quotaCell := quota
		// Only a real percentage earns a bar; "local" and "—" get none.
		if row.QuotaSource == usage.QuotaReported || row.QuotaSource == usage.QuotaEstimated {
			quotaCell = fmt.Sprintf("%-5s %s", quota, miniBar(row.QuotaPct, 8))
		}

		model, tokens, cost := emptyCell, emptyCell, formatCost(row)
		if row.Available {
			if row.Model != "" {
				model = row.Model
			}
			tokens = formatTokens(row.TotalTokens)
		}

		// Rendered separately from the numeric columns: quotaCell may contain
		// ANSI colour, which would break %-16s padding.
		left := fmt.Sprintf("%-10s ", truncate(row.Harness, 10))
		right := fmt.Sprintf(" %-18s %10s %8s",
			truncate(model, 18), tokens, cost)

		pad := 16 - lipgloss.Width(quotaCell)
		if pad < 0 {
			pad = 0
		}

		line := left + quotaCell + strings.Repeat(" ", pad) + right
		if !row.Available {
			line = styleDim.Render(left + quotaCell + strings.Repeat(" ", pad) + right)
		}
		lines = append(lines, truncate(line, innerW))
	}

	// Footnotes explain any cell the table cannot state plainly. Without these
	// a "~" or a "—" looks like a rendering glitch rather than a claim about
	// what is knowable.
	var notes []string
	for _, row := range rows {
		if row.QuotaSource == usage.QuotaEstimated {
			notes = append(notes, fmt.Sprintf("~ %s %% is estimated against a local token cap, not a reported quota",
				row.Harness))
		}
		if row.QuotaSource == usage.QuotaReported {
			if r := formatResets(row.QuotaResets, now); r != "" {
				notes = append(notes, fmt.Sprintf("%s %s", row.Harness, r))
			}
		}
		if !row.Available {
			notes = append(notes, fmt.Sprintf("%s reports no usage log", row.Harness))
		}
	}
	if len(notes) > 0 {
		lines = append(lines, "")
		for _, n := range notes {
			lines = append(lines, styleDim.Render(truncate(n, innerW)))
		}
	}

	return lines
}

// usageWindow reports the rolling window usage covers.
func (m Model) usageWindow() time.Duration {
	if m.mon == nil {
		return usage.DefaultWindow
	}
	return m.mon.UsageWindow()
}

// shortDuration renders a window length as "5h" or "30m".
func shortDuration(d time.Duration) string {
	if d >= time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}
