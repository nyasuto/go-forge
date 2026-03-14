package output

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
// and sends a desktop notification if it hasn't already been sent for that window.
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
// Default is the platform-specific notification function (notify_*.go).
var sendNotificationFunc = sendPlatformNotification

func (n *Notifier) sendNotification(windowName string, utilization float64) {
	sendNotificationFunc(windowName, utilization)
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
