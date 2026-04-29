package monitor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultOllamaURL = "http://localhost:11434"
)

// OllamaClient interacts with Ollama API
type OllamaClient struct {
	baseURL string
	client  *http.Client
}

// ollamaTagsResponse represents the /api/tags response
type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	} `json:"models"`
}

// ollamaPsResponse represents the /api/ps response (currently loaded models)
type ollamaPsResponse struct {
	Models []struct {
		Name      string    `json:"name"`
		Size      uint64    `json:"size"`
		SizeVRAM  uint64    `json:"size_vram"`
		ExpiresAt time.Time `json:"expires_at"`
	} `json:"models"`
}

// NewOllamaClient creates a client for Ollama API
func NewOllamaClient(baseURL string) *OllamaClient {
	if baseURL == "" {
		baseURL = DefaultOllamaURL
	}

	return &OllamaClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// GetModels returns list of loaded models
func (oc *OllamaClient) GetModels() ([]ModelInfo, error) {
	resp, err := oc.client.Get(oc.baseURL + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error: %s", string(body))
	}

	var result ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	models := make([]ModelInfo, 0, len(result.Models))
	for _, m := range result.Models {
		models = append(models, ModelInfo{
			Name:      m.Name,
			Size:      FormatMemory(uint64(m.Size)),
			SizeBytes: m.Size,
			Modified:  time.Now(),
			Loaded:    true,
		})
	}

	return models, nil
}

// IsRunning checks if Ollama service is accessible
func (oc *OllamaClient) IsRunning() bool {
	resp, err := oc.client.Get(oc.baseURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

// GetRunningModels returns models currently loaded in memory via /api/ps.
// Each entry includes VRAM and RAM allocation from Ollama.
func (oc *OllamaClient) GetRunningModels() ([]RunningModel, error) {
	resp, err := oc.client.Get(oc.baseURL + "/api/ps")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error: %s", string(body))
	}

	var result ollamaPsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse ollama /api/ps response: %w", err)
	}

	models := make([]RunningModel, 0, len(result.Models))
	for _, m := range result.Models {
		sizeRAM := uint64(0)
		if m.Size > m.SizeVRAM {
			sizeRAM = m.Size - m.SizeVRAM
		}
		models = append(models, RunningModel{
			Name:      m.Name,
			SizeVRAM:  m.SizeVRAM,
			SizeRAM:   sizeRAM,
			ExpiresAt: m.ExpiresAt,
		})
	}
	return models, nil
}
