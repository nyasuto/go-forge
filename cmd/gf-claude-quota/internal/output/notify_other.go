//go:build !darwin && !linux

package output

func sendPlatformNotification(windowName string, utilization float64) {
	// No notification support on this platform
}
