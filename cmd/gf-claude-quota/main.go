package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"gf-claude-quota/internal/api"
	"gf-claude-quota/internal/cache"
	"gf-claude-quota/internal/credentials"
)

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	fs := flag.NewFlagSet("gf-claude-quota", flag.ContinueOnError)
	fs.SetOutput(stderr)

	showVersion := fs.Bool("version", false, "show version")
	cacheTTL := fs.Int("cache-ttl", 60, "cache TTL in seconds")
	noCache := fs.Bool("no-cache", false, "disable cache")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "gf-claude-quota version "+version)
		return 0
	}

	// Try cache first
	var fc *cache.FileCache
	if !*noCache {
		fc = cache.NewFileCache("", time.Duration(*cacheTTL)*time.Second)
		if usage, err := fc.Get(); err == nil && usage != nil {
			formatUsage(stdout, usage)
			return 0
		}
	}

	token, err := credentials.GetTokenFromKeychain(nil)
	if err != nil {
		fmt.Fprintf(stderr, "gf-claude-quota: %v\n", err)
		return 1
	}

	client := api.NewClient(nil)
	usage, err := client.FetchUsage(token)
	if err != nil {
		fmt.Fprintf(stderr, "gf-claude-quota: %v\n", err)
		return 1
	}

	// Store in cache (best-effort)
	if fc != nil {
		_ = fc.Set(usage)
	}

	formatUsage(stdout, usage)
	return 0
}

func formatUsage(out *os.File, usage *api.UsageResponse) {
	fmt.Fprintln(out, "Claude Code Usage")
	fmt.Fprintln(out, strings.Repeat("\u2500", 45))

	if usage.FiveHour != nil {
		printWindow(out, "5h Session", usage.FiveHour)
	}
	if usage.SevenDay != nil {
		printWindow(out, "7d Weekly ", usage.SevenDay)
	}
	if usage.SevenDayOpus != nil {
		printWindow(out, "7d Opus   ", usage.SevenDayOpus)
	}
}

func printWindow(out *os.File, label string, w *api.UsageWindow) {
	bar := buildBar(w.Utilization, 10)
	resetStr := ""
	if w.ResetsAt != nil {
		resetStr = formatResetTime(*w.ResetsAt)
	}
	fmt.Fprintf(out, "%s  [%s]  %3.0f%%", label, bar, w.Utilization)
	if resetStr != "" {
		fmt.Fprintf(out, "  resets in %s", resetStr)
	}
	fmt.Fprintln(out)
}

func buildBar(pct float64, width int) string {
	filled := int(math.Round(pct / 100.0 * float64(width)))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", width-filled)
}

// nowFunc is replaceable for testing.
var nowFunc = time.Now

func formatResetTime(resetAt string) string {
	t, err := time.Parse(time.RFC3339Nano, resetAt)
	if err != nil {
		return resetAt
	}

	diff := t.Sub(nowFunc())
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
