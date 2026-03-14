package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gf-claude-quota/internal/api"
)

// ColorMode represents the color output mode.
type ColorMode int

const (
	ColorAuto   ColorMode = iota
	ColorAlways
	ColorNever
)

// ParseColorMode parses a color mode string.
func ParseColorMode(s string) (ColorMode, error) {
	switch s {
	case "auto":
		return ColorAuto, nil
	case "always":
		return ColorAlways, nil
	case "never":
		return ColorNever, nil
	default:
		return ColorAuto, fmt.Errorf("invalid color mode: %q (must be auto, always, or never)", s)
	}
}

// IsTerminal checks if a file descriptor is a terminal.
func IsTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// ShouldColorize determines whether to use colors based on mode and output destination.
func ShouldColorize(mode ColorMode, out *os.File) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default: // ColorAuto
		return IsTerminal(out)
	}
}

// FormatText writes usage data in human-readable text format with optional colors.
func FormatText(w io.Writer, usage *api.UsageResponse, useColor bool) {
	fmt.Fprintln(w, "Claude Code Usage")
	fmt.Fprintln(w, strings.Repeat("\u2500", 45))

	if usage.FiveHour != nil {
		printWindow(w, "5h Session", usage.FiveHour, useColor)
	}
	if usage.SevenDay != nil {
		printWindow(w, "7d Weekly ", usage.SevenDay, useColor)
	}
	if usage.SevenDayOpus != nil {
		printWindow(w, "7d Opus   ", usage.SevenDayOpus, useColor)
	}
}

func printWindow(w io.Writer, label string, win *api.UsageWindow, useColor bool) {
	bar := BuildBar(win.Utilization, 10)
	resetStr := ""
	if win.ResetsAt != nil {
		resetStr = FormatResetTime(*win.ResetsAt)
	}

	pctStr := fmt.Sprintf("%3.0f%%", win.Utilization)
	barStr := fmt.Sprintf("[%s]", bar)

	if useColor {
		barStr = Colorize(barStr, win.Utilization)
		pctStr = Colorize(pctStr, win.Utilization)
	}

	fmt.Fprintf(w, "%s  %s  %s", label, barStr, pctStr)
	if resetStr != "" {
		fmt.Fprintf(w, "  resets in %s", resetStr)
	}
	fmt.Fprintln(w)
}
