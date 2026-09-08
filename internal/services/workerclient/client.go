package workerclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{baseURL: baseURL, token: token, http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) do(ctx context.Context, method, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("X-Internal-Token", c.token)
	}
	return c.http.Do(req)
}

type HealthStatus struct {
	State         string `json:"state"`
	EventsDropped int64  `json:"events_dropped"`
	EventsSpilled int64  `json:"events_spilled"`
	SpoolBacklog  int    `json:"spool_backlog"`
	Reachable     bool   `json:"-"`
}

// Health never returns an error — if the worker is unreachable (mid-restart,
// which is expected and fine), Reachable is simply false. Callers show that
// state in the UI instead of failing the whole status request.
func (c *Client) Health(ctx context.Context) HealthStatus {
	resp, err := c.do(ctx, "GET", "/internal/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		return HealthStatus{State: "unreachable"}
	}
	defer resp.Body.Close()

	var h HealthStatus
	json.NewDecoder(resp.Body).Decode(&h)
	h.Reachable = true
	return h
}

type LLMHealth struct {
	Reachable    bool      `json:"reachable"`
	Model        string    `json:"model"`
	LastReportAt time.Time `json:"last_report_at,omitzero"`
}

func (c *Client) LLMHealth(ctx context.Context) LLMHealth {
	resp, err := c.do(ctx, "GET", "/internal/llm-health")
	if err != nil || resp.StatusCode != http.StatusOK {
		return LLMHealth{}
	}
	defer resp.Body.Close()

	var h LLMHealth
	json.NewDecoder(resp.Body).Decode(&h)
	return h
}

func (c *Client) Reload(ctx context.Context) error {
	resp, err := c.do(ctx, "POST", "/internal/reload")
	if err != nil {
		return fmt.Errorf("worker unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("worker returned %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GenerateReport(ctx context.Context) (int64, error) {
	resp, err := c.do(ctx, "POST", "/internal/reports/generate")
	if err != nil {
		return 0, fmt.Errorf("worker unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("worker returned %d", resp.StatusCode)
	}
	var out struct {
		ID int64 `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.ID, nil
}
