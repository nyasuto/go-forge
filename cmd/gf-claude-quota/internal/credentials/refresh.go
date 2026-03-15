package credentials

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	tokenEndpoint = "https://console.anthropic.com/v1/oauth/token"
	clientID      = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	// Refresh token if it expires within this duration.
	expiryBuffer = 5 * time.Minute
)

// tokenResponse represents the OAuth token refresh response.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// TokenRefresher refreshes OAuth tokens via the Anthropic token endpoint.
type TokenRefresher struct {
	httpClient *http.Client
	endpoint   string
	nowFunc    func() time.Time
}

// NewTokenRefresher creates a new TokenRefresher.
// If httpClient is nil, http.DefaultClient is used.
func NewTokenRefresher(httpClient *http.Client) *TokenRefresher {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &TokenRefresher{
		httpClient: httpClient,
		endpoint:   tokenEndpoint,
		nowFunc:    time.Now,
	}
}

// SetEndpoint overrides the token endpoint (for testing).
func (r *TokenRefresher) SetEndpoint(endpoint string) {
	r.endpoint = endpoint
}

// IsExpired reports whether the given credential is expired or about to expire.
func (r *TokenRefresher) IsExpired(expiresAtMs int64) bool {
	expiresAt := time.UnixMilli(expiresAtMs)
	return r.nowFunc().Add(expiryBuffer).After(expiresAt)
}

// Refresh exchanges a refresh token for a new access token.
func (r *TokenRefresher) Refresh(refreshToken string) (*oauthEntry, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}

	req, err := http.NewRequest("POST", r.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed (%d): %s — try re-authenticating with `claude login`", resp.StatusCode, string(body))
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("parsing refresh response: %w", err)
	}

	if tr.AccessToken == "" {
		return nil, fmt.Errorf("refresh response missing access_token")
	}

	nowMs := r.nowFunc().UnixMilli()
	return &oauthEntry{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    nowMs + tr.ExpiresIn*1000,
	}, nil
}
