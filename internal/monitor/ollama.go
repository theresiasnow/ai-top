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

// OllamaResponse represents the API response
type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
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
			Modified:  time.Now(), // Ollama API doesn't provide modification time
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
