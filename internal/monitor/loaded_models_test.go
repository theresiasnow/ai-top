package monitor

import (
	"testing"
	"time"
)

func TestCollectLoadedModelsMergesBothBackends(t *testing.T) {
	m := &SystemMetrics{
		RunningModels: []RunningModel{{
			Name:          "qwen3-coder:30b",
			SizeVRAM:      18_600_000_000,
			ContextLength: 32768,
			Quantization:  "Q4_K_M",
			Backend:       BackendOllama,
		}},
		OllamaProcess: &ProcessInfo{PID: 501, User: "ollama"},
		Omlx: OmlxStatus{
			Models: []OmlxModelInfo{
				{ID: "mlx-community/Qwen3-8B", Loaded: true, SizeBytes: 8_000_000_000},
				{ID: "mlx-community/unloaded", Loaded: false, SizeBytes: 1},
			},
		},
		OmlxProcess: &ProcessInfo{PID: 777, User: "tess"},
	}

	got := CollectLoadedModels(m)

	// The unloaded omlx model must not appear: this view is what occupies
	// memory now, not what exists on disk.
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (unloaded omlx model should be excluded)", len(got))
	}

	if got[0].Backend != BackendOllama || got[0].ContextLength != 32768 {
		t.Errorf("ollama row = %+v, want backend ollama with ctx 32768", got[0])
	}
	if got[0].OwnerPID != 501 || got[0].OwnerUser != "ollama" {
		t.Errorf("ollama owner = %d/%s, want 501/ollama", got[0].OwnerPID, got[0].OwnerUser)
	}
	if got[1].Backend != BackendOmlx || got[1].OwnerPID != 777 {
		t.Errorf("omlx row = %+v, want backend omlx owned by 777", got[1])
	}
}

// omlx reports no context length or quantization. Those fields must stay empty
// so the view can render "—" rather than showing a fabricated value.
func TestCollectLoadedModelsLeavesUnknownOmlxFieldsEmpty(t *testing.T) {
	m := &SystemMetrics{
		Omlx: OmlxStatus{
			Models: []OmlxModelInfo{{ID: "mlx-community/Qwen3-8B", Loaded: true, SizeBytes: 8}},
		},
	}

	got := CollectLoadedModels(m)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ContextLength != 0 || got[0].Quantization != "" {
		t.Errorf("omlx row invented detail it cannot know: %+v", got[0])
	}
}

func TestLoadedModelTTL(t *testing.T) {
	now := time.Date(2026, 8, 17, 18, 0, 0, 0, time.UTC)

	var noExpiry LoadedModel
	if _, ok := noExpiry.TTL(now); ok {
		t.Error("zero ExpiresAt should report no TTL")
	}

	live := LoadedModel{ExpiresAt: now.Add(4 * time.Minute)}
	d, ok := live.TTL(now)
	if !ok || d != 4*time.Minute {
		t.Errorf("TTL = %v/%v, want 4m/true", d, ok)
	}

	// An expired model may linger until Ollama sweeps it; report zero, not a
	// negative duration that would render as "-3m".
	expired := LoadedModel{ExpiresAt: now.Add(-3 * time.Minute)}
	d, ok = expired.TTL(now)
	if !ok || d != 0 {
		t.Errorf("expired TTL = %v/%v, want 0/true", d, ok)
	}
}

func TestCollectLoadedModelsNilSafe(t *testing.T) {
	if got := CollectLoadedModels(nil); got != nil {
		t.Errorf("CollectLoadedModels(nil) = %v, want nil", got)
	}
}
