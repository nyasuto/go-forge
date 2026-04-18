package credentials

import (
	"errors"
	"fmt"
	"os"
)

// CredentialProvider retrieves an OAuth access token.
type CredentialProvider interface {
	GetToken() (string, error)
}

// ErrClaudeRunning is returned when the stored token is expired and a
// `claude` process is currently running. Callers should surface this as a
// short-lived state (1 minute menu-bar poll resolves it automatically once
// Claude Code refreshes on its next API call).
// ErrClaudeRunning message is intentionally capitalized for user-facing
// display in the menu bar / error output.
//
//nolint:staticcheck // ST1005: user-facing message, capitalization is deliberate
var ErrClaudeRunning = errors.New("Claude Code is running; its next API call will refresh the token — retry in a moment")

// lockReleaser is the subset of FileLock used by getTokenWithDeps.
type lockReleaser interface {
	Release() error
}

// getTokenDeps bundles the side-effecting operations GetToken orchestrates.
// All fields are injected for testability; GetToken binds the real ones.
type getTokenDeps struct {
	loadCredentials func() (*FullCredentials, error)
	saveCredentials func(*FullCredentials) error
	isClaudeRunning func() bool
	acquireLock     func() (lockReleaser, error)
	refresher       *TokenRefresher
}

// GetToken retrieves a valid OAuth access token.
//
// Resolution order:
//  1. CLAUDE_OAUTH_TOKEN environment variable (CI / tests)
//  2. Platform credential store (macOS Keychain / Linux credentials.json)
//
// If the stored token is expired:
//   - If a `claude` process is running, return ErrClaudeRunning (Claude Code
//     will refresh on its next API call; our refresh would invalidate its
//     in-memory refresh_token and force re-login).
//   - Otherwise, take an exclusive refresh lock, re-read credentials, and if
//     still expired, refresh via the OAuth endpoint and persist.
func GetToken() (string, error) {
	return getTokenWithDeps(&getTokenDeps{
		loadCredentials: getFullPlatformCredentials,
		saveCredentials: savePlatformCredentials,
		isClaudeRunning: IsClaudeRunning,
		acquireLock:     acquireRefreshLock,
		refresher:       NewTokenRefresher(nil),
	})
}

func getTokenWithDeps(d *getTokenDeps) (string, error) {
	if token := os.Getenv("CLAUDE_OAUTH_TOKEN"); token != "" {
		return token, nil
	}

	creds, err := d.loadCredentials()
	if err != nil {
		return "", err
	}

	if !d.refresher.IsExpired(creds.ExpiresAt) {
		return creds.AccessToken, nil
	}

	// Expired.
	if d.isClaudeRunning() {
		return "", ErrClaudeRunning
	}

	lock, err := d.acquireLock()
	if err != nil {
		return "", fmt.Errorf("acquiring refresh lock: %w", err)
	}
	defer func() { _ = lock.Release() }()

	// Re-read inside lock in case another instance just refreshed.
	creds, err = d.loadCredentials()
	if err != nil {
		return "", err
	}
	if !d.refresher.IsExpired(creds.ExpiresAt) {
		return creds.AccessToken, nil
	}

	if creds.RefreshToken == "" {
		return "", fmt.Errorf("token expired and no refresh token available — run `claude login`")
	}

	newEntry, err := d.refresher.Refresh(creds.RefreshToken)
	if err != nil {
		return "", err
	}

	updated := &FullCredentials{
		AccessToken:      newEntry.AccessToken,
		RefreshToken:     newEntry.RefreshToken,
		ExpiresAt:        newEntry.ExpiresAt,
		Scopes:           creds.Scopes,
		SubscriptionType: creds.SubscriptionType,
	}
	// Best-effort save. If it fails, the current caller still gets a valid
	// token; the next invocation will prompt re-login via `claude login`.
	_ = d.saveCredentials(updated)
	return updated.AccessToken, nil
}

// acquireRefreshLock opens and locks the default refresh lock file,
// returning a lockReleaser for deferred release.
func acquireRefreshLock() (lockReleaser, error) {
	path, err := defaultLockPath()
	if err != nil {
		return nil, err
	}
	lock := NewFileLock(path)
	if err := lock.Acquire(); err != nil {
		return nil, err
	}
	return lock, nil
}
