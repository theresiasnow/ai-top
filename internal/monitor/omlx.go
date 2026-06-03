package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	ID        string
	Loaded    bool
	SizeBytes uint64
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
	Model struct {
		ModelDirs []string `json:"model_dirs"`
		ModelDir  string   `json:"model_dir"`
	} `json:"model"`
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

type omlxModelsStatusResponse struct {
	Models []struct {
		ID            string `json:"id"`
		Loaded        bool   `json:"loaded"`
		EstimatedSize uint64 `json:"estimated_size"`
		ActualSize    uint64 `json:"actual_size"`
	} `json:"models"`
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

func (c *OmlxClient) modelDirs() []string {
	var dirs []string
	home, err := os.UserHomeDir()
	if err == nil {
		data, err := os.ReadFile(filepath.Join(home, ".omlx", "settings.json"))
		if err == nil {
			var s omlxSettings
			if json.Unmarshal(data, &s) == nil {
				dirs = append(dirs, s.Model.ModelDirs...)
				if s.Model.ModelDir != "" {
					dirs = append(dirs, s.Model.ModelDir)
				}
			}
		}
	}
	if len(dirs) == 0 && home != "" {
		dirs = append(dirs, filepath.Join(home, ".omlx", "models"))
	}

	seen := make(map[string]bool, len(dirs))
	unique := dirs[:0]
	for _, dir := range dirs {
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		unique = append(unique, dir)
	}
	return unique
}

func (c *OmlxClient) installedModels() []OmlxModelInfo {
	var models []OmlxModelInfo
	seen := make(map[string]bool)
	for _, dir := range c.modelDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || seen[entry.Name()] {
				continue
			}
			seen[entry.Name()] = true
			models = append(models, OmlxModelInfo{
				ID:        entry.Name(),
				SizeBytes: dirSize(filepath.Join(dir, entry.Name())),
			})
		}
	}
	return models
}

func dirSize(root string) uint64 {
	var total uint64
	_ = filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err == nil && info.Size() > 0 {
			total += uint64(info.Size())
		}
		return nil
	})
	return total
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
	status := OmlxStatus{Models: c.installedModels()}

	// Health (no auth needed)
	resp, err := c.client.Get(c.baseURL + "/health")
	if err != nil {
		return status, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return status, nil
	}

	var health omlxHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return status, err
	}
	status.Running = health.Status == "healthy"
	status.LoadedCount = health.EnginePool.LoadedCount
	status.MaxMemory = health.EnginePool.MaxModelMemory
	status.UsedMemory = health.EnginePool.CurrentModelMemory

	// Model list (requires auth). /v1/models/status carries explicit loaded
	// state; fall back to /v1/models only for IDs if an older server lacks it.
	if status.Running {
		if mResp, err := c.get("/v1/models/status"); err == nil {
			defer mResp.Body.Close()
			var modelsResp omlxModelsStatusResponse
			if json.NewDecoder(mResp.Body).Decode(&modelsResp) == nil {
				byID := make(map[string]int, len(status.Models))
				for i, model := range status.Models {
					byID[model.ID] = i
				}
				for _, m := range modelsResp.Models {
					size := m.ActualSize
					if size == 0 {
						size = m.EstimatedSize
					}
					if i, ok := byID[m.ID]; ok {
						status.Models[i].Loaded = m.Loaded
						if size > 0 {
							status.Models[i].SizeBytes = size
						}
					} else {
						status.Models = append(status.Models, OmlxModelInfo{
							ID:        m.ID,
							Loaded:    m.Loaded,
							SizeBytes: size,
						})
					}
				}
				return status, nil
			}
		}
		if mResp, err := c.get("/v1/models"); err == nil {
			defer mResp.Body.Close()
			var modelsResp omlxModelsResponse
			if json.NewDecoder(mResp.Body).Decode(&modelsResp) == nil {
				byID := make(map[string]bool, len(status.Models))
				for _, model := range status.Models {
					byID[model.ID] = true
				}
				for _, m := range modelsResp.Data {
					if !byID[m.ID] {
						status.Models = append(status.Models, OmlxModelInfo{ID: m.ID})
					}
				}
			}
		}
	}

	return status, nil
}
