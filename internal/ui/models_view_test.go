package ui

import (
	"testing"
	"time"

	"github.com/tess/ai-top/internal/monitor"
)

func TestFormatContext(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, emptyCell},  // omlx reports no context length
		{-1, emptyCell}, // never render a negative window
		{2048, "2K"},
		{32768, "32K"},
		{4000, "4000"}, // not a clean multiple of 1024, so show it verbatim
	}

	for _, tt := range tests {
		if got := formatContext(tt.in); got != tt.want {
			t.Errorf("formatContext(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatTTL(t *testing.T) {
	now := time.Date(2026, 8, 17, 18, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		model monitor.LoadedModel
		want  string
	}{
		{"no expiry reported", monitor.LoadedModel{}, emptyCell},
		{"four minutes", monitor.LoadedModel{ExpiresAt: now.Add(4*time.Minute + 12*time.Second)}, "4m12s"},
		{"seconds", monitor.LoadedModel{ExpiresAt: now.Add(30 * time.Second)}, "30s"},
		{"hours", monitor.LoadedModel{ExpiresAt: now.Add(2*time.Hour + 5*time.Minute)}, "2h5m"},
		{"already expired", monitor.LoadedModel{ExpiresAt: now.Add(-time.Minute)}, "expiring"},
	}

	for _, tt := range tests {
		if got := formatTTL(tt.model, now); got != tt.want {
			t.Errorf("%s: formatTTL = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestFormatOwner(t *testing.T) {
	if got := formatOwner(monitor.LoadedModel{}); got != emptyCell {
		t.Errorf("unknown owner = %q, want %q", got, emptyCell)
	}
	if got := formatOwner(monitor.LoadedModel{OwnerPID: 501, OwnerUser: "ollama"}); got != "ollama/501" {
		t.Errorf("owner = %q, want ollama/501", got)
	}
	if got := formatOwner(monitor.LoadedModel{OwnerPID: 501}); got != "501" {
		t.Errorf("owner without user = %q, want 501", got)
	}
}

// A backend that reports no context or quantization must show "—" in those
// columns rather than a fabricated or zero value.
func TestLoadedModelLinesMarksUnknownOmlxFields(t *testing.T) {
	m := Model{width: 120}
	models := []monitor.LoadedModel{{
		Name:     "mlx-community/Qwen3-8B",
		Backend:  monitor.BackendOmlx,
		SizeVRAM: 8_000_000_000,
	}}

	out := stripANSI(m.loadedModelLines(models, 116))

	if !containsAll(out, "omlx", emptyCell) {
		t.Errorf("expected omlx row with %q for unknown fields, got:\n%s", emptyCell, out)
	}
}

func TestLoadedModelLinesRendersOllamaDetail(t *testing.T) {
	m := Model{width: 120}
	models := []monitor.LoadedModel{{
		Name:          "qwen3-coder:30b",
		Backend:       monitor.BackendOllama,
		SizeVRAM:      18_600_000_000,
		ContextLength: 32768,
		Quantization:  "Q4_K_M",
		OwnerPID:      501,
		OwnerUser:     "ollama",
	}}

	out := stripANSI(m.loadedModelLines(models, 116))

	if !containsAll(out, "qwen3-coder:30b", "ollama", "32K", "Q4_K_M", "ollama/501") {
		t.Errorf("missing expected detail, got:\n%s", out)
	}
}

func TestLoadedModelLinesEmpty(t *testing.T) {
	m := Model{width: 120}
	out := stripANSI(m.loadedModelLines(nil, 116))

	if !containsAll(out, "no models currently loaded") {
		t.Errorf("expected empty-state message, got:\n%s", out)
	}
}
