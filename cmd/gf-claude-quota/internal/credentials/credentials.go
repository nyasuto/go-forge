package credentials

import (
	"fmt"
	"os"
)

// CredentialProvider retrieves an OAuth access token.
type CredentialProvider interface {
	GetToken() (string, error)
}

// GetToken retrieves a valid OAuth access token, refreshing if necessary.
// It checks the CLAUDE_OAUTH_TOKEN environment variable first, then falls
// back to the platform-specific method (macOS Keychain or Linux file).
// If the stored token is expired, it automatically refreshes using the
// stored refresh token and persists the new credentials.
func GetToken() (string, error) {
	if token := os.Getenv("CLAUDE_OAUTH_TOKEN"); token != "" {
		return token, nil
	}

	creds, err := getFullPlatformCredentials()
	if err != nil {
		return "", err
	}

	refresher := NewTokenRefresher(nil)
	if !refresher.IsExpired(creds.ExpiresAt) {
		return creds.AccessToken, nil
	}

	if creds.RefreshToken == "" {
		return "", fmt.Errorf("token expired and no refresh token available — try re-authenticating with `claude login`")
	}

	newEntry, err := refresher.Refresh(creds.RefreshToken)
	if err != nil {
		return "", err
	}

	// Preserve original scopes and subscription type.
	updated := &FullCredentials{
		AccessToken:      newEntry.AccessToken,
		RefreshToken:     newEntry.RefreshToken,
		ExpiresAt:        newEntry.ExpiresAt,
		Scopes:           creds.Scopes,
		SubscriptionType: creds.SubscriptionType,
	}

	// Best-effort persist — don't fail the request if save fails.
	_ = savePlatformCredentials(updated)

	return updated.AccessToken, nil
}
