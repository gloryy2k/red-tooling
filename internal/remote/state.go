package remote

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/user/rt/internal/config"
)

type ConnectionState struct {
	ServerURL string `json:"server_url"`
	APIKey    string `json:"api_key"`
	SessionID string `json:"session_id"`
	Operator  string `json:"operator"`
	Insecure  bool   `json:"insecure"`
}

func statePath() string {
	return filepath.Join(config.Home(), "remote.json")
}

func SaveState(s *ConnectionState) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(), data, 0600)
}

func LoadState() (*ConnectionState, error) {
	data, err := os.ReadFile(statePath())
	if err != nil {
		return nil, err
	}
	var s ConnectionState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func ClearState() error {
	return os.Remove(statePath())
}

func IsJoined() bool {
	_, err := os.Stat(statePath())
	return err == nil
}
