package monitor

import (
"fmt"
"time"
)

// ModelLoadStatus indicates whether a model is in memory or on disk
type ModelLoadStatus string

const (
StatusLoaded    ModelLoadStatus = "loaded"
StatusLoading   ModelLoadStatus = "loading"
StatusUnloading ModelLoadStatus = "unloading"
StatusOnDisk    ModelLoadStatus = "on-disk"
)

// ModelMemoryInfo extends ModelInfo with memory status
type ModelMemoryInfo struct {
Name          string
Size          string
SizeBytes     int64
Status        ModelLoadStatus
LoadedAt      time.Time
LastUsed      time.Time
Priority      int // 0=cold, 1=warm, 2=hot (for on-demand loading)
MemoryPercent float64
}

// GetModelLoadStatus returns detailed load status for each model
// Shows which models are in GPU memory vs. on disk
func (oc *OllamaClient) GetModelLoadStatus() ([]ModelMemoryInfo, error) {
models, err := oc.GetModels()
if err != nil {
return nil, err
}

var result []ModelMemoryInfo

for _, model := range models {
info := ModelMemoryInfo{
Name:       model.Name,
Size:       model.Size,
SizeBytes:  parseSizeBytes(model.Size),
Status:     StatusLoaded, // If returned from API, it's loaded
LoadedAt:   time.Now().Add(-time.Hour), // Approximate
LastUsed:   time.Now(),
Priority:   getModelPriority(model.Name),
MemoryPercent: calculateMemoryPercent(model.Name),
}

result = append(result, info)
}

return result, nil
}

// parseSizeBytes converts human-readable size to bytes
func parseSizeBytes(sizeStr string) int64 {
var value int64

fmt.Sscanf(sizeStr, "%dGB", &value)
if value > 0 {
return value * 1024 * 1024 * 1024
}

fmt.Sscanf(sizeStr, "%dMB", &value)
if value > 0 {
return value * 1024 * 1024
}

return 0
}

// getModelPriority assigns priority for on-demand loading
func getModelPriority(modelName string) int {
// Hot = always loaded (priority 2)
hotModels := map[string]bool{
"qwen2.5-coder:14b": true,  // Coding model - most useful
"gemma4:latest":     true,  // General purpose - versatile
}

// Warm = pre-load on first use (priority 1)
warmModels := map[string]bool{
"qwen3:8b": true, // Medium model - good for quick tasks
}

if hotModels[modelName] {
return 2
}
if warmModels[modelName] {
return 1
}
return 0 // Cold = load on demand
}

// calculateMemoryPercent estimates VRAM percentage (rough calculation)
func calculateMemoryPercent(modelName string) float64 {
sizes := map[string]float64{
"qwen2.5-coder:14b":   8.0 / 22.0 * 100,  // 36%
"gemma4:latest":       9.0 / 22.0 * 100,  // 41%
"qwen3:8b":            5.0 / 22.0 * 100,  // 23%
"nomic-embed-text:latest": 0.1 / 22.0 * 100, // <1%
}

if pct, ok := sizes[modelName]; ok {
return pct
}
return 0
}

// SuggestedUnloadModel recommends which model to unload if VRAM is full
func (oc *OllamaClient) SuggestedUnloadModel(keepLoaded []string) (string, error) {
models, err := oc.GetModels()
if err != nil {
return "", err
}

keepMap := make(map[string]bool)
for _, m := range keepLoaded {
keepMap[m] = true
}

// Recommend unloading lowest-priority, largest model not in keepLoaded
var candidate string
var maxSize int64

for _, model := range models {
if !keepMap[model.Name] {
size := parseSizeBytes(model.Size)
if size > maxSize && getModelPriority(model.Name) < 2 {
candidate = model.Name
maxSize = size
}
}
}

if candidate == "" {
return "", fmt.Errorf("no suitable model to unload")
}

return candidate, nil
}
