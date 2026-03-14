//go:build darwin

package output

import (
	"fmt"
	"os/exec"
)

func sendPlatformNotification(windowName string, utilization float64) {
	title := "gf-claude-quota"
	msg := fmt.Sprintf("%s usage at %.0f%%", windowName, utilization)
	script := fmt.Sprintf(`display notification %q with title %q`, msg, title)
	// Best-effort: ignore errors
	_ = exec.Command("osascript", "-e", script).Run()
}
