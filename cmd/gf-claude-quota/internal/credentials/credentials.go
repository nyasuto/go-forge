package credentials

import (
	"fmt"
	"os"
)

// CredentialProvider retrieves an OAuth access token.
type CredentialProvider interface {
	GetToken() (string, error)
}

// GetToken retrieves a valid OAuth access token (read-only).
// It checks the CLAUDE_OAUTH_TOKEN environment variable first, then falls
// back to the platform-specific method (macOS Keychain or Linux file).
//
// This function intentionally does NOT refresh expired tokens. Anthropic's
// OAuth server uses refresh token rotation (single-use refresh tokens), so
// refreshing here would invalidate Claude Code's stored refresh token and
// force the user to re-login. Let Claude Code manage its own token lifecycle.
func GetToken() (string, error) {
	if token := os.Getenv("CLAUDE_OAUTH_TOKEN"); token != "" {
		return token, nil
	}

	creds, err := getFullPlatformCredentials()
	if err != nil {
		return "", err
	}

	refresher := NewTokenRefresher(nil)
	if refresher.IsExpired(creds.ExpiresAt) {
		return "", fmt.Errorf("token expired — run any `claude` command to refresh, or `claude login` to re-authenticate")
	}

	return creds.AccessToken, nil
}
