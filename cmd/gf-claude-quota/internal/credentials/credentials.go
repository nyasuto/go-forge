package credentials

import "os"

// CredentialProvider retrieves an OAuth access token.
type CredentialProvider interface {
	GetToken() (string, error)
}

// GetToken retrieves the OAuth access token using the appropriate method.
// It checks the CLAUDE_OAUTH_TOKEN environment variable first, then falls
// back to the platform-specific method (macOS Keychain or Linux file).
func GetToken() (string, error) {
	if token := os.Getenv("CLAUDE_OAUTH_TOKEN"); token != "" {
		return token, nil
	}
	return getPlatformToken()
}
