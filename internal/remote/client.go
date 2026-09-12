package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type SessionReq struct {
	Action   string `json:"action"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Source   string `json:"source"`
	Operator string `json:"operator"`
}

type EvidenceReq struct {
	SessionID  string   `json:"session_id"`
	Action     string   `json:"action"`
	Input      string   `json:"input"`
	Output     string   `json:"output"`
	ExitCode   int      `json:"exit_code"`
	DurationMs int      `json:"duration_ms"`
	CWD        string   `json:"cwd"`
	Tags       []string `json:"tags"`
	Priority   string   `json:"priority"`
	Operator   string   `json:"operator"`
}

type EvidenceResp struct {
	ID        int64  `json:"id"`
	Hash      string `json:"hash"`
	PrevHash  string `json:"prev_hash"`
	Timestamp string `json:"timestamp"`
}

type CredReq struct {
	Username   string `json:"username"`
	Secret     string `json:"secret"`
	SecretType string `json:"secret_type"`
	Host       string `json:"host"`
	Operator   string `json:"operator"`
	EvidenceID int64  `json:"evidence_id"`
}

func (c *Client) StartSession(req SessionReq) error {
	req.Action = "start"
	return c.postJSON("/api/sessions", req, nil)
}

func (c *Client) StopSession(sessionID, operator string) error {
	req := SessionReq{Action: "stop", ID: sessionID, Operator: operator}
	return c.postJSON("/api/sessions", req, nil)
}

func (c *Client) SubmitEvidence(req EvidenceReq) (*EvidenceResp, error) {
	var resp EvidenceResp
	if err := c.postJSON("/api/evidence", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SubmitCredential(req CredReq) error {
	return c.postJSON("/api/creds", req, nil)
}

func (c *Client) postJSON(path string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("server error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

type ContextResp struct {
	Engagement map[string]interface{}   `json:"engagement"`
	Scope      map[string]interface{}   `json:"scope"`
	Findings   []map[string]interface{} `json:"findings"`
	Checklist  map[string]interface{}   `json:"checklist"`
}

func (c *Client) GetContext() (*ContextResp, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/api/context", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach server: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var ctx ContextResp
	if err := json.Unmarshal(body, &ctx); err != nil {
		return nil, fmt.Errorf("decode context: %w", err)
	}
	return &ctx, nil
}

func (c *Client) Ping() error {
	req, err := http.NewRequest("GET", c.BaseURL+"/api/overview", nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned %d — check API key", resp.StatusCode)
	}
	return nil
}
