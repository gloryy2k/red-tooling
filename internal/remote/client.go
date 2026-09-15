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

type FindingReq struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	Host        string   `json:"host"`
	Mitre       []string `json:"mitre"`
	EvidenceIDs []int64  `json:"evidence_ids"`
	Operator    string   `json:"operator"`
}

type FindingResp struct {
	ID int64 `json:"id"`
}

func (c *Client) SubmitFinding(req FindingReq) (*FindingResp, error) {
	var resp FindingResp
	if err := c.postJSON("/api/findings", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
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

type ScopeResp struct {
	Added int `json:"added"`
}

func (c *Client) AddScope(hosts string) (*ScopeResp, error) {
	var resp ScopeResp
	if err := c.postJSON("/api/scope", map[string]string{"hosts": hosts}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) MarkScopeTested(host, sessionID string) error {
	req := map[string]string{"host": host, "session_id": sessionID}
	return c.postJSON("/api/scope/tested", req, nil)
}

func (c *Client) GetScope() (map[string]interface{}, error) {
	r, err := http.NewRequest("GET", c.BaseURL+"/api/scope", nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("cannot reach server: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode scope: %w", err)
	}
	return result, nil
}

func (c *Client) ListFindings() ([]map[string]interface{}, error) {
	return c.getJSON("/api/findings")
}

func (c *Client) VerifyFinding(id int64, status, note string) error {
	return c.postJSON(fmt.Sprintf("/api/findings/%d/verify", id), map[string]string{
		"status": status,
		"note":   note,
	}, nil)
}

func (c *Client) SetRecommendation(id int64, rec string) error {
	return c.postJSON(fmt.Sprintf("/api/findings/%d/recommend", id), map[string]string{
		"recommendation": rec,
	}, nil)
}

func (c *Client) UpdateFinding(id int64, data map[string]interface{}) error {
	return c.putJSON(fmt.Sprintf("/api/findings/%d", id), data)
}

func (c *Client) ListCreds() ([]map[string]interface{}, error) {
	return c.getJSON("/api/creds")
}

func (c *Client) GetChecklist() (map[string]interface{}, error) {
	r, err := http.NewRequest("GET", c.BaseURL+"/api/checklist", nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("X-API-Key", c.APIKey)
	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("cannot reach server: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode checklist: %w", err)
	}
	return result, nil
}

func (c *Client) LoadChecklistPreset(preset string) error {
	return c.postJSON("/api/checklist", map[string]string{"preset": preset}, nil)
}

func (c *Client) AddChecklistItem(category, item string) error {
	return c.postJSON("/api/checklist", map[string]string{"category": category, "item": item}, nil)
}

func (c *Client) ToggleChecklist(id int64, checked bool, evidenceID int64) error {
	return c.postJSON("/api/checklist/toggle", map[string]interface{}{
		"id":          id,
		"checked":     checked,
		"evidence_id": evidenceID,
	}, nil)
}

func (c *Client) GetTimeline() ([]map[string]interface{}, error) {
	return c.getJSON("/api/timeline")
}

func (c *Client) GetAudit(limit int) ([]map[string]interface{}, error) {
	return c.getJSON(fmt.Sprintf("/api/audit?limit=%d", limit))
}

func (c *Client) GetReport(format string, opts map[string]string) (string, error) {
	q := fmt.Sprintf("/api/report?format=%s", format)
	for k, v := range opts {
		if v == "true" {
			q += "&" + k + "=true"
		}
	}
	r, err := http.NewRequest("GET", c.BaseURL+q, nil)
	if err != nil {
		return "", err
	}
	r.Header.Set("X-API-Key", c.APIKey)
	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return "", fmt.Errorf("cannot reach server: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}

func (c *Client) ListAttachments(evidenceID string) ([]map[string]interface{}, error) {
	path := "/api/attachments"
	if evidenceID != "" {
		path += "?evidence_id=" + evidenceID
	}
	return c.getJSON(path)
}

func (c *Client) getJSON(path string) ([]map[string]interface{}, error) {
	r, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("X-API-Key", c.APIKey)
	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("cannot reach server: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

func (c *Client) putJSON(path string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequest("PUT", c.BaseURL+path, bytes.NewReader(data))
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
		var errResp struct{ Error string `json:"error"` }
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("server error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
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
