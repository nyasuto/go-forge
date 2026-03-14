package credentials

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// keychainCredentials represents the JSON structure stored in macOS Keychain.
type keychainCredentials struct {
	ClaudeAIOAuth *oauthEntry `json:"claudeAiOauth"`
}

type oauthEntry struct {
	AccessToken      string   `json:"accessToken"`
	RefreshToken     string   `json:"refreshToken"`
	ExpiresAt        int64    `json:"expiresAt"`
	Scopes           []string `json:"scopes"`
	SubscriptionType string   `json:"subscriptionType"`
}

// CommandRunner abstracts command execution for testability.
type CommandRunner func(name string, args ...string) ([]byte, error)

// DefaultCommandRunner executes a real shell command.
func DefaultCommandRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// GetTokenFromKeychain retrieves the OAuth access token from macOS Keychain.
func GetTokenFromKeychain(runner CommandRunner) (string, error) {
	if runner == nil {
		runner = DefaultCommandRunner
	}

	output, err := runner("security", "find-generic-password", "-s", "Claude Code-credentials", "-w")
	if err != nil {
		return "", fmt.Errorf("failed to read keychain: %w (is Claude Code installed and logged in?)", err)
	}

	raw := strings.TrimSpace(string(output))
	return ParseKeychainJSON(raw)
}

// ParseKeychainJSON extracts the accessToken from the Keychain JSON string.
func ParseKeychainJSON(raw string) (string, error) {
	var creds keychainCredentials
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return "", fmt.Errorf("parsing keychain credentials: %w", err)
	}

	if creds.ClaudeAIOAuth == nil {
		return "", fmt.Errorf("keychain credentials missing claudeAiOauth field")
	}

	if creds.ClaudeAIOAuth.AccessToken == "" {
		return "", fmt.Errorf("keychain credentials missing accessToken")
	}

	return creds.ClaudeAIOAuth.AccessToken, nil
}
