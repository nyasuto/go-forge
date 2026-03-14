package output

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// NowFunc is replaceable for testing.
var NowFunc = time.Now

// BuildBar creates a visual progress bar with filled (█) and empty (░) characters.
func BuildBar(pct float64, width int) string {
	filled := int(math.Round(pct / 100.0 * float64(width)))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", width-filled)
}

// FormatResetTime converts an RFC3339 reset time to a human-readable relative duration.
func FormatResetTime(resetAt string) string {
	t, err := time.Parse(time.RFC3339Nano, resetAt)
	if err != nil {
		return resetAt
	}

	diff := t.Sub(NowFunc())
	if diff <= 0 {
		return "now"
	}

	days := int(diff.Hours()) / 24
	hours := int(diff.Hours()) % 24
	minutes := int(diff.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// ColorLevel returns a color level based on utilization percentage.
// 0-49% → green, 50-79% → yellow, 80-100% → red
func ColorLevel(pct float64) string {
	switch {
	case pct >= 80:
		return "red"
	case pct >= 50:
		return "yellow"
	default:
		return "green"
	}
}

// ANSI color codes.
const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
)

// Colorize wraps text with ANSI color based on utilization percentage.
func Colorize(text string, pct float64) string {
	var code string
	switch ColorLevel(pct) {
	case "red":
		code = ansiRed
	case "yellow":
		code = ansiYellow
	default:
		code = ansiGreen
	}
	return code + text + ansiReset
}
