package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"syscall"
)

// KillProcess sends SIGTERM to a process.
func KillProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

// RestartProcess sends SIGHUP to a process.
func RestartProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGHUP)
}

// UnloadOllamaModel asks Ollama to unload a model by setting keep_alive to 0.
func UnloadOllamaModel(client *OllamaClient, name string) error {
	if client == nil {
		return fmt.Errorf("ollama client is nil")
	}

	body, err := json.Marshal(map[string]any{
		"model":      name,
		"prompt":     "",
		"keep_alive": 0,
		"stream":     false,
	})
	if err != nil {
		return err
	}

	resp, err := client.client.Post(client.baseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to unload ollama model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("ollama unload failed: %s", resp.Status)
	}
	return nil
}
