//go:build linux

package credentials

func getPlatformToken() (string, error) {
	return GetTokenFromFile("")
}
