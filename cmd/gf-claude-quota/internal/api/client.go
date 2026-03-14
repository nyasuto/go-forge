package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	defaultEndpoint = "https://api.anthropic.com/api/oauth/usage"
	userAgent       = "claude-code/2.0.31"
	anthropicBeta   = "oauth-2025-04-20"
)

// Client is an HTTP client for the Anthropic usage API.
type Client struct {
	httpClient *http.Client
	endpoint   string
}

// NewClient creates a new API client with the given HTTP client.
// If httpClient is nil, http.DefaultClient is used.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		httpClient: httpClient,
		endpoint:   defaultEndpoint,
	}
}

// SetEndpoint overrides the API endpoint (for testing).
func (c *Client) SetEndpoint(endpoint string) {
	c.endpoint = endpoint
}

// FetchUsage retrieves usage data from the Anthropic API.
func (c *Client) FetchUsage(token string) (*UsageResponse, error) {
	req, err := http.NewRequest("GET", c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", anthropicBeta)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// success
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("authentication failed (401): token may be expired or invalid")
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limited (429): too many requests, please try again later")
	default:
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var usage UsageResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		return nil, fmt.Errorf("parsing response JSON: %w", err)
	}

	return &usage, nil
}
