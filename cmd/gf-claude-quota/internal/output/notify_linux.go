//go:build linux

package output

import (
	"fmt"
	"os/exec"
)

func sendPlatformNotification(windowName string, utilization float64) {
	title := "gf-claude-quota"
	msg := fmt.Sprintf("%s usage at %.0f%%", windowName, utilization)
	// Best-effort: ignore errors (notify-send may not be installed)
	_ = exec.Command("notify-send", title, msg).Run()
}
