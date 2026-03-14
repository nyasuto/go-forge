package credentials

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetTokenFromFile retrieves the OAuth access token from the Linux
// credentials file at ~/.config/claude-code/credentials.json.
// If path is empty, the default location is used.
func GetTokenFromFile(path string) (string, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		path = filepath.Join(home, ".config", "claude-code", "credentials.json")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read credentials file %s: %w (is Claude Code installed and logged in?)", path, err)
	}

	return ParseKeychainJSON(string(data))
}
