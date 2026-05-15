package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const defaultOmlxPort = 8000

// OmlxStatus holds current omlx server state.
type OmlxStatus struct {
	Running     bool
	Models      []OmlxModelInfo
	LoadedCount int
	MaxMemory   uint64 // bytes
	UsedMemory  uint64 // bytes
}

// OmlxModelInfo represents a model known to omlx.
type OmlxModelInfo struct {
	ID     string
	Loaded bool
}

// OmlxClient talks to the omlx HTTP API.
type OmlxClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// SupportsOmlx reports whether omlx should be probed on this platform.
func SupportsOmlx() bool {
	return runtime.GOOS == "darwin"
}

type omlxSettings struct {
	Server struct {
		Port int `json:"port"`
	} `json:"server"`
	Auth struct {
		APIKey string `json:"api_key"`
	} `json:"auth"`
}

type omlxHealthResponse struct {
	Status     string `json:"status"`
	EnginePool struct {
		ModelCount         int    `json:"model_count"`
		LoadedCount        int    `json:"loaded_count"`
		MaxModelMemory     uint64 `json:"max_model_memory"`
		CurrentModelMemory uint64 `json:"current_model_memory"`
	} `json:"engine_pool"`
}

type omlxModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// NewOmlxClient reads ~/.omlx/settings.json and builds a client.
// Falls back to port 8000 and no API key if settings are unreadable.
func NewOmlxClient() *OmlxClient {
	port := defaultOmlxPort
	apiKey := ""

	home, err := os.UserHomeDir()
	if err == nil {
		data, err := os.ReadFile(filepath.Join(home, ".omlx", "settings.json"))
		if err == nil {
			var s omlxSettings
			if json.Unmarshal(data, &s) == nil {
				if s.Server.Port != 0 {
					port = s.Server.Port
				}
				apiKey = s.Auth.APIKey
			}
		}
	}

	return &OmlxClient{
		baseURL: fmt.Sprintf("http://localhost:%d", port),
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 2 * time.Second},
	}
}

func (c *OmlxClient) get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	return c.client.Do(req)
}

// IsRunning returns true if the /health endpoint responds.
func (c *OmlxClient) IsRunning() bool {
	resp, err := c.client.Get(c.baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

// GetStatus returns a populated OmlxStatus.
func (c *OmlxClient) GetStatus() (OmlxStatus, error) {
	status := OmlxStatus{}

	// Health (no auth needed)
	resp, err := c.client.Get(c.baseURL + "/health")
	if err != nil {
		return status, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return status, fmt.Errorf("omlx health returned %d", resp.StatusCode)
	}

	var health omlxHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return status, err
	}
	status.Running = health.Status == "healthy"
	status.LoadedCount = health.EnginePool.LoadedCount
	status.MaxMemory = health.EnginePool.MaxModelMemory
	status.UsedMemory = health.EnginePool.CurrentModelMemory

	// Model list (requires auth)
	if status.Running {
		if mResp, err := c.get("/v1/models"); err == nil {
			defer mResp.Body.Close()
			var modelsResp omlxModelsResponse
			if json.NewDecoder(mResp.Body).Decode(&modelsResp) == nil {
				for _, m := range modelsResp.Data {
					status.Models = append(status.Models, OmlxModelInfo{ID: m.ID, Loaded: true})
				}
			}
		}
	}

	return status, nil
}
