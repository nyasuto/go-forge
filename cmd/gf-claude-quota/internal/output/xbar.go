package output

import (
	"fmt"
	"io"

	"gf-claude-quota/internal/api"
)

// FormatXbar outputs usage data in xbar/SwiftBar plugin format.
func FormatXbar(w io.Writer, usage *api.UsageResponse) {
	type windowInfo struct {
		label string
		win   *api.UsageWindow
	}

	windows := []windowInfo{
		{"5h Session", usage.FiveHour},
		{"7d Weekly", usage.SevenDay},
		{"7d OAuth", usage.SevenDayOAuth},
		{"7d Opus", usage.SevenDayOpus},
	}

	// Build menu bar title
	maxPct := maxUtilization(usage)
	titleColor := ColorLevel(maxPct)

	var has5h, has7d bool
	var pct5h, pct7d float64
	if usage.FiveHour != nil {
		has5h = true
		pct5h = usage.FiveHour.Utilization
	}
	if usage.SevenDay != nil {
		has7d = true
		pct7d = usage.SevenDay.Utilization
	}

	if !has5h && !has7d {
		fmt.Fprintln(w, "Claude \u23f3")
		return
	}

	// Title line: ⚡42% 📅18% | color=green
	title := ""
	if has5h {
		title += fmt.Sprintf("\u26a1%.0f%%", pct5h)
	}
	if has7d {
		if title != "" {
			title += " "
		}
		title += fmt.Sprintf("\U0001f4c5%.0f%%", pct7d)
	}
	fmt.Fprintf(w, "%s | color=%s\n", title, titleColor)

	// Separator
	fmt.Fprintln(w, "---")

	// Dropdown: each window
	for _, wi := range windows {
		if wi.win == nil {
			continue
		}
		color := ColorLevel(wi.win.Utilization)
		fmt.Fprintf(w, "%s | color=%s\n", wi.label, color)

		bar := BuildBar(wi.win.Utilization, 10)
		fmt.Fprintf(w, "--[%s] %.0f%% | font=Menlo size=12\n", bar, wi.win.Utilization)

		if wi.win.ResetsAt != nil {
			resetTime := FormatResetTime(*wi.win.ResetsAt)
			fmt.Fprintf(w, "--resets in %s | size=11\n", resetTime)
		}
	}

	// Footer
	fmt.Fprintln(w, "---")
	fmt.Fprintln(w, "Refresh | refresh=true")
	fmt.Fprintln(w, "Open Claude | href=https://claude.ai")
}

// maxUtilization returns the highest utilization across all non-nil windows.
func maxUtilization(usage *api.UsageResponse) float64 {
	max := 0.0
	for _, w := range []*api.UsageWindow{usage.FiveHour, usage.SevenDay, usage.SevenDayOAuth, usage.SevenDayOpus} {
		if w != nil && w.Utilization > max {
			max = w.Utilization
		}
	}
	return max
}
