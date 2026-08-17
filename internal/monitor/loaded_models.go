package monitor

import (
	"time"
)

// Backend names for loaded models.
const (
	BackendOllama = "ollama"
	BackendOmlx   = "omlx"
)

// LoadedModel is one row of the merged model view: a model currently resident
// in memory, whichever backend loaded it.
//
// The two backends report different amounts of detail — Ollama gives context
// length and quantization, omlx does not — so fields absent from a backend stay
// at their zero value and render as "—" rather than being filled with a guess.
type LoadedModel struct {
	Name          string
	Backend       string
	SizeVRAM      uint64
	SizeRAM       uint64
	ContextLength int
	Quantization  string
	ParameterSize string
	ExpiresAt     time.Time

	// Owner is the process holding the model, when one is known.
	OwnerPID  int
	OwnerUser string
}

// TTL reports how long until the backend unloads an idle model, and whether a
// TTL is known at all. omlx reports no expiry, and Ollama omits it for models
// pinned in memory.
func (l LoadedModel) TTL(now time.Time) (time.Duration, bool) {
	if l.ExpiresAt.IsZero() {
		return 0, false
	}
	d := l.ExpiresAt.Sub(now)
	if d < 0 {
		return 0, true // expired but not yet swept
	}
	return d, true
}

// CollectLoadedModels merges the loaded models of every backend into one list,
// attaching the owning process where the metrics identify it.
func CollectLoadedModels(m *SystemMetrics) []LoadedModel {
	if m == nil {
		return nil
	}

	out := make([]LoadedModel, 0, len(m.RunningModels)+len(m.Omlx.Models))

	for _, rm := range m.RunningModels {
		backend := rm.Backend
		if backend == "" {
			backend = BackendOllama
		}
		lm := LoadedModel{
			Name:          rm.Name,
			Backend:       backend,
			SizeVRAM:      rm.SizeVRAM,
			SizeRAM:       rm.SizeRAM,
			ContextLength: rm.ContextLength,
			Quantization:  rm.Quantization,
			ParameterSize: rm.ParameterSize,
			ExpiresAt:     rm.ExpiresAt,
		}
		if m.OllamaProcess != nil {
			lm.OwnerPID = m.OllamaProcess.PID
			lm.OwnerUser = m.OllamaProcess.User
		}
		out = append(out, lm)
	}

	// omlx reports every known model, loaded or not; only resident ones belong
	// in a view of what currently occupies memory.
	for _, om := range m.Omlx.Models {
		if !om.Loaded {
			continue
		}
		lm := LoadedModel{
			Name:     om.ID,
			Backend:  BackendOmlx,
			SizeVRAM: om.SizeBytes,
		}
		if m.OmlxProcess != nil {
			lm.OwnerPID = m.OmlxProcess.PID
			lm.OwnerUser = m.OmlxProcess.User
		}
		out = append(out, lm)
	}

	return out
}
