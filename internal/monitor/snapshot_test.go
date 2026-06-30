package monitor

import (
	"testing"
	"time"
)

// TestSnapshotMonitoredPartitions verifies a single process snapshot is sliced
// into the full list plus the first ollama/omlx process without re-enumerating.
func TestSnapshotMonitoredPartitions(t *testing.T) {
	snap := &processSnapshot{watched: []ProcessInfo{
		{PID: 1, Name: "node", CommandLine: "node openclaw/server.js"},
		{PID: 2, Name: "ollama", CommandLine: ""},
		{PID: 3, Name: "node", CommandLine: "node unrelated.js"},
	}}

	all, ollamaProc, _ := snap.monitored()
	if len(all) != 3 {
		t.Fatalf("expected 3 processes, got %d", len(all))
	}
	if ollamaProc == nil || ollamaProc.PID != 2 {
		t.Fatalf("expected ollama PID 2, got %+v", ollamaProc)
	}
}

// TestSnapshotOpenClawMatch verifies OpenClaw is found by command line within
// the shared snapshot, with no process-table sweep.
func TestSnapshotOpenClawMatch(t *testing.T) {
	snap := &processSnapshot{watched: []ProcessInfo{
		{PID: 1, Name: "node", CommandLine: "node openclaw/index.js", StartTime: time.Now()},
		{PID: 2, Name: "node", CommandLine: "node something-else.js"},
		{PID: 3, Name: "ollama"},
	}}

	matches := snap.openClaw()
	if len(matches) != 1 || matches[0].PID != 1 {
		t.Fatalf("expected only PID 1 to match openclaw, got %+v", matches)
	}
}

// TestGetStatusFromSnapshotReusesSnapshot verifies GetStatusFromSnapshot reports
// running from the provided snapshot without taking its own.
func TestGetStatusFromSnapshotReusesSnapshot(t *testing.T) {
	od := NewOpenClawDetector(0)
	snap := &processSnapshot{watched: []ProcessInfo{
		{PID: 42, Name: "node", CommandLine: "node openclaw/main.js", Memory: 123, StartTime: time.Now()},
	}}

	status, err := od.GetStatusFromSnapshot(snap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Running || status.PID != 42 || status.Memory != 123 {
		t.Fatalf("expected running openclaw PID 42 from snapshot, got %+v", status)
	}
}
