//go:build darwin

package credentials

func getPlatformToken() (string, error) {
	return GetTokenFromKeychain(nil)
}
