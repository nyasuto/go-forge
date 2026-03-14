package output

import (
	"fmt"
	"os/exec"
)

// Notifier manages threshold notifications with deduplication.
type Notifier struct {
	threshold float64
	notified  map[string]bool // tracks which windows have already triggered
}

// NewNotifier creates a Notifier that fires at the given percentage threshold.
func NewNotifier(threshold float64) *Notifier {
	return &Notifier{
		threshold: threshold,
		notified:  make(map[string]bool),
	}
}

// Check evaluates whether a usage window has crossed the threshold
// and sends a macOS notification if it hasn't already been sent for that window.
func (n *Notifier) Check(windowName string, utilization float64) {
	if utilization < n.threshold {
		// Reset notification if usage drops below threshold
		delete(n.notified, windowName)
		return
	}
	if n.notified[windowName] {
		return
	}
	n.notified[windowName] = true
	n.sendNotification(windowName, utilization)
}

// sendNotificationFunc is replaceable for testing.
var sendNotificationFunc = sendOSANotification

func (n *Notifier) sendNotification(windowName string, utilization float64) {
	sendNotificationFunc(windowName, utilization)
}

func sendOSANotification(windowName string, utilization float64) {
	title := "gf-claude-quota"
	msg := fmt.Sprintf("%s usage at %.0f%%", windowName, utilization)
	script := fmt.Sprintf(`display notification %q with title %q`, msg, title)
	// Best-effort: ignore errors (e.g., non-macOS)
	_ = exec.Command("osascript", "-e", script).Run()
}

// ExportSendNotificationFunc returns the current sendNotificationFunc (for testing).
func ExportSendNotificationFunc() func(string, float64) {
	return sendNotificationFunc
}

// SetSendNotificationFunc sets sendNotificationFunc (for testing).
func SetSendNotificationFunc(f func(string, float64)) {
	sendNotificationFunc = f
}

// ClearTerminalSeq returns ANSI escape sequences to clear the screen and move cursor home.
func ClearTerminalSeq() string {
	return "\033[2J\033[H"
}
