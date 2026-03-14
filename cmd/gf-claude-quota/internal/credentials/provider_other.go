//go:build !darwin && !linux

package credentials

import "fmt"

func getPlatformToken() (string, error) {
	return "", fmt.Errorf("unsupported platform: set CLAUDE_OAUTH_TOKEN environment variable")
}
