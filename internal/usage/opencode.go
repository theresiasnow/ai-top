package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// OpenCode stores usage in sqlite. Rather than take on a cgo sqlite driver as a
// top-level dependency — which would also cost cross-compilation — this source
// shells out to the sqlite3 CLI, in the same spirit as monitor's process
// actions. If the CLI is absent the harness simply reports unavailable.
const sqliteBin = "sqlite3"

// sqliteTimeout bounds the query. The database belongs to another live process,
// so a hung read must not stall a refresh.
const sqliteTimeout = 3 * time.Second

// OpenCodeSource reads token usage and cost from the OpenCode sqlite database.
type OpenCodeSource struct {
	dbPath string
}

// NewOpenCodeSource creates a source reading the database at dbPath.
func NewOpenCodeSource(dbPath string) *OpenCodeSource {
	return &OpenCodeSource{dbPath: dbPath}
}

// DefaultOpenCodeDB returns the standard OpenCode database location.
func DefaultOpenCodeDB() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode", "opencode.db")
}

func (o *OpenCodeSource) Name() string { return "OpenCode" }

// openCodeRow is one aggregate result row from the sqlite CLI's JSON output.
type openCodeRow struct {
	TokensInput      int64   `json:"ti"`
	TokensOutput     int64   `json:"to_"`
	TokensCacheRead  int64   `json:"tcr"`
	TokensCacheWrite int64   `json:"tcw"`
	Cost             float64 `json:"cost"`
	Model            string  `json:"model"`
}

// Collect aggregates session usage within the window.
//
// The database is opened read-only and immutable so ai-top can never disturb a
// live OpenCode session, and so a concurrently-held write lock cannot block us.
func (o *OpenCodeSource) Collect(since time.Time) (HarnessUsage, error) {
	out := HarnessUsage{Harness: o.Name(), QuotaSource: QuotaNone}

	if o.dbPath == "" {
		return out, nil
	}
	if _, err := os.Stat(o.dbPath); err != nil {
		return out, nil
	}
	if _, err := exec.LookPath(sqliteBin); err != nil {
		return out, nil
	}

	uri := fmt.Sprintf("file:%s?mode=ro&immutable=1", o.dbPath)
	cutoff := unixMillis(since)

	// One query for the totals, one for the most recent model. Keeping them
	// separate avoids a GROUP BY whose row we would have to pick from anyway.
	query := fmt.Sprintf(`select
		coalesce(sum(tokens_input),0) as ti,
		coalesce(sum(tokens_output),0) as to_,
		coalesce(sum(tokens_cache_read),0) as tcr,
		coalesce(sum(tokens_cache_write),0) as tcw,
		coalesce(sum(cost),0) as cost
		from session where time_updated >= %d;`, cutoff)

	rows, err := o.query(uri, query)
	if err != nil {
		return out, nil // treat an unreadable db as absent, not fatal
	}

	out.Available = true
	out.HasCost = true
	if len(rows) > 0 {
		r := rows[0]
		out.InputTokens = r.TokensInput
		out.OutputTokens = r.TokensOutput
		out.CacheRead = r.TokensCacheRead
		out.CacheWrite = r.TokensCacheWrite
		out.CostUSD = r.Cost
		out.TotalTokens = r.TokensInput + r.TokensOutput
	}

	modelQuery := fmt.Sprintf(`select model from session
		where model is not null and time_updated >= %d
		order by time_updated desc limit 1;`, cutoff)
	if modelRows, err := o.query(uri, modelQuery); err == nil && len(modelRows) > 0 {
		out.Model = parseOpenCodeModel(modelRows[0].Model)
	}

	return out, nil
}

// query runs one statement through the sqlite CLI in JSON mode.
func (o *OpenCodeSource) query(uri, sql string) ([]openCodeRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), sqliteTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, sqliteBin, "-readonly", "-json", uri, sql)
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if len(stdout) == 0 {
		return nil, nil // no rows: sqlite prints nothing at all
	}

	var rows []openCodeRow
	if err := json.Unmarshal(stdout, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// parseOpenCodeModel extracts the display name from OpenCode's model column,
// which holds a JSON object such as {"id":"qwen3-coder:30b","providerID":"ollama"}
// rather than a bare model name. Anything unparseable is passed through as-is.
func parseOpenCodeModel(raw string) string {
	if raw == "" || raw == "null" {
		return ""
	}

	var m struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return raw
	}
	if m.ID == "" {
		return raw
	}
	return m.ID
}

// unixMillis converts a time to milliseconds since epoch, the unit OpenCode
// stores in time_updated.
func unixMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}
