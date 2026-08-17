package usage

import (
	"sync"
	"time"
)

// CollectInterval is how often usage is refreshed.
//
// Deliberately far slower than the monitor's 2s tick: this scans hundreds of
// log files and a database, and none of those numbers move fast enough to
// justify reading them every couple of seconds.
const CollectInterval = 30 * time.Second

// Collector runs every harness source and caches the last result.
type Collector struct {
	mu      sync.RWMutex
	rows    []HarnessUsage
	sources []Source
	window  time.Duration
}

// NewCollector builds a collector over the default harness locations.
func NewCollector() *Collector {
	return &Collector{
		sources: []Source{
			NewClaudeSource(DefaultClaudeRoot(), DefaultClaudeCap),
			NewCodexSource(DefaultCodexRoot()),
			NewOpenCodeSource(DefaultOpenCodeDB()),
			NewOpenClawSource(),
		},
		window: DefaultWindow,
	}
}

// Rows returns a copy of the most recent collection.
func (c *Collector) Rows() []HarnessUsage {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.rows == nil {
		return nil
	}
	out := make([]HarnessUsage, len(c.rows))
	copy(out, c.rows)
	return out
}

// Window returns the rolling window usage is summed over.
func (c *Collector) Window() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.window
}

// Refresh runs every source once and replaces the cached rows.
//
// Sources run outside the lock so one slow harness cannot block readers, and a
// failing source yields an unavailable row rather than dropping out of the
// table entirely — a harness that is present but unreadable is worth showing.
func (c *Collector) Refresh() {
	c.mu.RLock()
	sources := c.sources
	window := c.window
	c.mu.RUnlock()

	since := WindowStart(time.Now(), window)
	rows := make([]HarnessUsage, 0, len(sources))

	for _, src := range sources {
		row, err := src.Collect(since)
		if err != nil {
			row = HarnessUsage{Harness: src.Name(), Available: false}
		}
		if row.Harness == "" {
			row.Harness = src.Name()
		}
		rows = append(rows, row)
	}

	c.mu.Lock()
	c.rows = rows
	c.mu.Unlock()
}

// StartAutoRefresh refreshes in the background until stop is closed.
func (c *Collector) StartAutoRefresh(stop <-chan struct{}) {
	go func() {
		c.Refresh() // populate immediately; the first tick is 30s away

		ticker := time.NewTicker(CollectInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.Refresh()
			case <-stop:
				return
			}
		}
	}()
}
