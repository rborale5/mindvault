package n8n

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type AgentRequest struct {
	Query     string            `json:"query"`
	SessionID string            `json:"sessionId,omitempty"`
	UserID    string            `json:"userId,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type AgentResponse struct {
	Response string `json:"response"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// CallAgent sends a request to an n8n webhook-triggered workflow and returns the response.
func (c *Client) CallAgent(ctx context.Context, agentPath string, req AgentRequest) (*AgentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/webhook/%s", c.BaseURL, strings.TrimLeft(agentPath, "/"))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call n8n webhook: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("n8n returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var agentResp AgentResponse
	if err := json.Unmarshal(respBody, &agentResp); err != nil {
		// If n8n returns a plain string or unexpected shape, wrap it
		agentResp.Response = string(respBody)
	}

	return &agentResp, nil
}
