//go:build linux

package credentials

func getPlatformToken() (string, error) {
	return GetTokenFromFile("")
}

func getFullPlatformCredentials() (*FullCredentials, error) {
	return GetFullCredentialsFromFile("")
}

func savePlatformCredentials(creds *FullCredentials) error {
	return SaveToFile("", creds)
}
