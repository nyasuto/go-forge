package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FullCredentials holds all credential fields needed for refresh logic.
type FullCredentials struct {
	AccessToken      string   `json:"accessToken"`
	RefreshToken     string   `json:"refreshToken"`
	ExpiresAt        int64    `json:"expiresAt"`
	Scopes           []string `json:"scopes"`
	SubscriptionType string   `json:"subscriptionType"`
}

// ParseFullCredentials extracts all credential fields from the stored JSON string.
func ParseFullCredentials(raw string) (*FullCredentials, error) {
	var creds keychainCredentials
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}

	if creds.ClaudeAIOAuth == nil {
		return nil, fmt.Errorf("credentials missing claudeAiOauth field")
	}

	if creds.ClaudeAIOAuth.AccessToken == "" {
		return nil, fmt.Errorf("credentials missing accessToken")
	}

	return &FullCredentials{
		AccessToken:      creds.ClaudeAIOAuth.AccessToken,
		RefreshToken:     creds.ClaudeAIOAuth.RefreshToken,
		ExpiresAt:        creds.ClaudeAIOAuth.ExpiresAt,
		Scopes:           creds.ClaudeAIOAuth.Scopes,
		SubscriptionType: creds.ClaudeAIOAuth.SubscriptionType,
	}, nil
}

// GetFullCredentialsFromKeychain retrieves all credential fields from macOS Keychain.
func GetFullCredentialsFromKeychain(runner CommandRunner) (*FullCredentials, error) {
	if runner == nil {
		runner = DefaultCommandRunner
	}

	output, err := runner("security", "find-generic-password", "-s", "Claude Code-credentials", "-w")
	if err != nil {
		return nil, fmt.Errorf("failed to read keychain: %w (is Claude Code installed and logged in?)", err)
	}

	raw := trimSpace(string(output))
	return ParseFullCredentials(raw)
}

// GetFullCredentialsFromFile retrieves all credential fields from a JSON file.
func GetFullCredentialsFromFile(path string) (*FullCredentials, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		path = filepath.Join(home, ".config", "claude-code", "credentials.json")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file %s: %w (is Claude Code installed and logged in?)", path, err)
	}

	return ParseFullCredentials(string(data))
}

// SaveToKeychain writes updated credentials back to macOS Keychain.
func SaveToKeychain(runner CommandRunner, creds *FullCredentials, original string) error {
	if runner == nil {
		runner = DefaultCommandRunner
	}

	// Parse original JSON to preserve any unknown fields.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(original), &raw); err != nil {
		raw = make(map[string]json.RawMessage)
	}

	entry := &oauthEntry{
		AccessToken:      creds.AccessToken,
		RefreshToken:     creds.RefreshToken,
		ExpiresAt:        creds.ExpiresAt,
		Scopes:           creds.Scopes,
		SubscriptionType: creds.SubscriptionType,
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}
	raw["claudeAiOauth"] = entryJSON

	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshaling full credentials: %w", err)
	}

	_, err = runner("security", "add-generic-password",
		"-U", // update if exists
		"-s", "Claude Code-credentials",
		"-a", "Claude Code-credentials",
		"-w", string(data),
	)
	if err != nil {
		return fmt.Errorf("failed to write keychain: %w", err)
	}
	return nil
}

// SaveToFile writes updated credentials back to a JSON file.
func SaveToFile(path string, creds *FullCredentials) error {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		path = filepath.Join(home, ".config", "claude-code", "credentials.json")
	}

	// Read existing file to preserve unknown fields.
	existing, _ := os.ReadFile(path)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(existing, &raw); err != nil {
		raw = make(map[string]json.RawMessage)
	}

	entry := &oauthEntry{
		AccessToken:      creds.AccessToken,
		RefreshToken:     creds.RefreshToken,
		ExpiresAt:        creds.ExpiresAt,
		Scopes:           creds.Scopes,
		SubscriptionType: creds.SubscriptionType,
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}
	raw["claudeAiOauth"] = entryJSON

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling full credentials: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

func trimSpace(s string) string {
	// Inline trim to avoid importing strings just for TrimSpace.
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
