package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"gf-claude-quota/internal/api"
	"gf-claude-quota/internal/cache"
	"gf-claude-quota/internal/credentials"
	"gf-claude-quota/internal/output"
)

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Stdin))
}

func run(args []string, stdout, stderr *os.File, stdin io.Reader) int {
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

	// Try cache first
	var fc *cache.FileCache
	if !*noCache {
		fc = cache.NewFileCache("", time.Duration(*cacheTTL)*time.Second)
		if usage, err := fc.Get(); err == nil && usage != nil {
			printUsage(stdout, usage, *jsonMode, *onelineMode, *statuslineMode, *formatTmpl, stdinData, colorMode)
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

	printUsage(stdout, usage, *jsonMode, *onelineMode, *statuslineMode, *formatTmpl, stdinData, colorMode)
	return 0
}

func printUsage(out *os.File, usage *api.UsageResponse, jsonMode, onelineMode, statuslineMode bool, formatTmpl string, stdinData []byte, colorMode output.ColorMode) {
	switch {
	case jsonMode:
		_ = output.FormatJSON(out, usage)
	case onelineMode:
		output.FormatOneline(out, usage)
	case statuslineMode:
		output.FormatStatusLine(out, usage, stdinData)
	case formatTmpl != "":
		output.FormatTemplate(out, usage, stdinData, formatTmpl)
	default:
		useColor := output.ShouldColorize(colorMode, out)
		output.FormatText(out, usage, useColor)
	}
}
