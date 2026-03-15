//go:build !darwin && !linux

package credentials

import "fmt"

func getPlatformToken() (string, error) {
	return "", fmt.Errorf("unsupported platform: set CLAUDE_OAUTH_TOKEN environment variable")
}

func getFullPlatformCredentials() (*FullCredentials, error) {
	return nil, fmt.Errorf("unsupported platform: set CLAUDE_OAUTH_TOKEN environment variable")
}

func savePlatformCredentials(creds *FullCredentials) error {
	return fmt.Errorf("unsupported platform: cannot save credentials")
}
