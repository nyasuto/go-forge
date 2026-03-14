package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"gf-claude-quota/internal/api"
	"gf-claude-quota/internal/cache"
	"gf-claude-quota/internal/credentials"
	"gf-claude-quota/internal/output"
	"gf-claude-quota/internal/setup"
)

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Stdin))
}

func run(args []string, stdout, stderr *os.File, stdin io.Reader) int {
	// Check for "setup" subcommand
	if len(args) > 0 && args[0] == "setup" {
		return runSetup(args[1:], stdout, stderr)
	}

	fs := flag.NewFlagSet("gf-claude-quota", flag.ContinueOnError)
	fs.SetOutput(stderr)

	showVersion := fs.Bool("version", false, "show version")
	cacheTTL := fs.Int("cache-ttl", 60, "cache TTL in seconds")
	noCache := fs.Bool("no-cache", false, "disable cache")
	jsonMode := fs.Bool("json", false, "output in JSON format")
	onelineMode := fs.Bool("oneline", false, "output in oneline format")
	statuslineMode := fs.Bool("statusline", false, "output in statusLine format")
	formatTmpl := fs.String("format", "", "custom output template")
	colorFlag := fs.String("color", "auto", "color mode: auto|always|never")
	watchMode := fs.Bool("watch", false, "continuous monitoring mode")
	interval := fs.Int("interval", 60, "watch interval in seconds")
	notifyAt := fs.Float64("notify-at", -1, "notify when usage reaches this percentage (0-100)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "gf-claude-quota version "+version)
		return 0
	}

	// Validate color mode
	colorMode, err := output.ParseColorMode(*colorFlag)
	if err != nil {
		fmt.Fprintf(stderr, "gf-claude-quota: %v\n", err)
		return 2
	}

	// Validate interval
	if *interval <= 0 {
		fmt.Fprintln(stderr, "gf-claude-quota: --interval must be positive")
		return 2
	}

	// Validate notify-at
	if *notifyAt != -1 && (*notifyAt < 0 || *notifyAt > 100) {
		fmt.Fprintln(stderr, "gf-claude-quota: --notify-at must be between 0 and 100")
		return 2
	}

	// Validate mutually exclusive output modes
	modeCount := 0
	if *jsonMode {
		modeCount++
	}
	if *onelineMode {
		modeCount++
	}
	if *statuslineMode {
		modeCount++
	}
	if *formatTmpl != "" {
		modeCount++
	}
	if modeCount > 1 {
		fmt.Fprintln(stderr, "gf-claude-quota: --json, --oneline, --statusline, and --format are mutually exclusive")
		return 2
	}

	// Read stdin data for statusline/format modes
	var stdinData []byte
	if *statuslineMode || *formatTmpl != "" {
		stdinData, _ = io.ReadAll(stdin)
	}

	opts := &runOptions{
		jsonMode:       *jsonMode,
		onelineMode:    *onelineMode,
		statuslineMode: *statuslineMode,
		formatTmpl:     *formatTmpl,
		stdinData:      stdinData,
		colorMode:      colorMode,
		noCache:        *noCache,
		cacheTTL:       time.Duration(*cacheTTL) * time.Second,
	}

	if *watchMode {
		return runWatch(stdout, stderr, opts, *interval, *notifyAt)
	}

	return runOnce(stdout, stderr, opts)
}

type runOptions struct {
	jsonMode       bool
	onelineMode    bool
	statuslineMode bool
	formatTmpl     string
	stdinData      []byte
	colorMode      output.ColorMode
	noCache        bool
	cacheTTL       time.Duration
}

func fetchUsage(stderr *os.File, opts *runOptions) (*api.UsageResponse, error) {
	// Try cache first
	if !opts.noCache {
		fc := cache.NewFileCache("", opts.cacheTTL)
		if usage, err := fc.Get(); err == nil && usage != nil {
			return usage, nil
		}
	}

	token, err := credentials.GetToken()
	if err != nil {
		return nil, err
	}

	client := api.NewClient(nil)
	usage, err := client.FetchUsage(token)
	if err != nil {
		return nil, err
	}

	// Store in cache (best-effort)
	if !opts.noCache {
		fc := cache.NewFileCache("", opts.cacheTTL)
		_ = fc.Set(usage)
	}

	return usage, nil
}

func runOnce(stdout, stderr *os.File, opts *runOptions) int {
	usage, err := fetchUsage(stderr, opts)
	if err != nil {
		fmt.Fprintf(stderr, "gf-claude-quota: %v\n", err)
		return 1
	}

	printUsage(stdout, usage, opts)
	return 0
}

// sleepFunc is replaceable for testing.
var sleepFunc = func(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func runWatch(stdout, stderr *os.File, opts *runOptions, intervalSec int, notifyAt float64) int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var notifier *output.Notifier
	if notifyAt >= 0 {
		notifier = output.NewNotifier(notifyAt)
	}

	dur := time.Duration(intervalSec) * time.Second
	first := true

	for {
		if !first {
			if err := sleepFunc(ctx, dur); err != nil {
				return 0 // interrupted
			}
		}
		first = false

		usage, err := fetchUsage(stderr, &runOptions{
			noCache:  true, // always fetch fresh in watch mode
			cacheTTL: opts.cacheTTL,
		})
		if err != nil {
			fmt.Fprintf(stderr, "gf-claude-quota: %v\n", err)
			// Continue watching on error
			continue
		}

		// Clear terminal
		fmt.Fprint(stdout, output.ClearTerminalSeq())

		printUsage(stdout, usage, opts)

		// Check notification thresholds
		if notifier != nil {
			if usage.FiveHour != nil {
				notifier.Check("5h Session", usage.FiveHour.Utilization)
			}
			if usage.SevenDay != nil {
				notifier.Check("7d Weekly", usage.SevenDay.Utilization)
			}
			if usage.SevenDayOpus != nil {
				notifier.Check("7d Opus", usage.SevenDayOpus.Utilization)
			}
		}

		// Check if context cancelled
		select {
		case <-ctx.Done():
			return 0
		default:
		}
	}
}

func runSetup(args []string, stdout, stderr *os.File) int {
	fs := flag.NewFlagSet("gf-claude-quota setup", flag.ContinueOnError)
	fs.SetOutput(stderr)

	tmux := fs.Bool("tmux", false, "output tmux statusbar configuration")
	starship := fs.Bool("starship", false, "output starship module configuration")
	dryRun := fs.Bool("dry-run", false, "preview changes without applying")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	return setup.Run(stdout, stderr, &setup.SetupOptions{
		Tmux:     *tmux,
		Starship: *starship,
		DryRun:   *dryRun,
	})
}

func printUsage(out *os.File, usage *api.UsageResponse, opts *runOptions) {
	switch {
	case opts.jsonMode:
		_ = output.FormatJSON(out, usage)
	case opts.onelineMode:
		output.FormatOneline(out, usage)
	case opts.statuslineMode:
		output.FormatStatusLine(out, usage, opts.stdinData)
	case opts.formatTmpl != "":
		output.FormatTemplate(out, usage, opts.stdinData, opts.formatTmpl)
	default:
		useColor := output.ShouldColorize(opts.colorMode, out)
		output.FormatText(out, usage, useColor)
	}
}
