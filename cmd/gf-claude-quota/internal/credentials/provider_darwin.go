//go:build darwin

package credentials

func getPlatformToken() (string, error) {
	return GetTokenFromKeychain(nil)
}

func getFullPlatformCredentials() (*FullCredentials, error) {
	return GetFullCredentialsFromKeychain(nil)
}

func savePlatformCredentials(creds *FullCredentials) error {
	// Read current raw JSON to preserve unknown fields.
	runner := DefaultCommandRunner
	output, err := runner("security", "find-generic-password", "-s", "Claude Code-credentials", "-w")
	if err != nil {
		return err
	}
	return SaveToKeychain(nil, creds, trimSpace(string(output)))
}
