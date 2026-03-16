package setup

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Settings represents Claude Code's settings.json structure.
// We preserve unknown fields via the raw map approach.
type Settings map[string]interface{}

// SetupOptions holds configuration for the setup command.
type SetupOptions struct {
	Tmux     bool
	Starship bool
	Xbar     bool
	DryRun   bool
}

// settingsPath returns the path to ~/.claude/settings.json.
// Exported for testing via variable override.
var SettingsPath = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "settings.json")
}

// FindBinaryPath finds the absolute path to gf-claude-quota binary.
var FindBinaryPath = func() (string, error) {
	// First try: look for it in PATH
	path, err := exec.LookPath("gf-claude-quota")
	if err == nil {
		abs, err := filepath.Abs(path)
		if err == nil {
			return abs, nil
		}
		return path, nil
	}

	// Second try: current executable
	exe, err := os.Executable()
	if err == nil {
		return exe, nil
	}

	return "", fmt.Errorf("could not find gf-claude-quota binary: install it in PATH or run from build directory")
}

// Run executes the setup command.
func Run(w, errw io.Writer, opts *SetupOptions) int {
	if opts.Tmux {
		return printTmuxConfig(w)
	}
	if opts.Starship {
		return printStarshipConfig(w)
	}
	if opts.Xbar {
		return printXbarConfig(w)
	}
	return setupStatusLine(w, errw, opts.DryRun)
}

func printTmuxConfig(w io.Writer) int {
	binPath, err := FindBinaryPath()
	if err != nil {
		binPath = "gf-claude-quota"
	}

	fmt.Fprintln(w, "# tmux status bar configuration for gf-claude-quota")
	fmt.Fprintln(w, "# Add the following to your ~/.tmux.conf:")
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "set -g status-right '#(%s --oneline)'\n", binPath)
	fmt.Fprintln(w, "set -g status-interval 60")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "# Or with custom format:")
	fmt.Fprintf(w, "# set -g status-right '#(%s --format \"5h:{5h} 7d:{7d}\")'\n", binPath)
	return 0
}

func printStarshipConfig(w io.Writer) int {
	binPath, err := FindBinaryPath()
	if err != nil {
		binPath = "gf-claude-quota"
	}

	fmt.Fprintln(w, "# Starship module configuration for gf-claude-quota")
	fmt.Fprintln(w, "# Add the following to your ~/.config/starship.toml:")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "[custom.claude_quota]")
	fmt.Fprintf(w, "command = '%s --oneline'\n", binPath)
	fmt.Fprintln(w, "when = true")
	fmt.Fprintln(w, "format = '[$output]($style) '")
	fmt.Fprintln(w, "style = 'bold yellow'")
	fmt.Fprintln(w, "shell = ['sh']")
	return 0
}

func printXbarConfig(w io.Writer) int {
	binPath, err := FindBinaryPath()
	if err != nil {
		binPath = "gf-claude-quota"
	}

	fmt.Fprintln(w, "# xbar/SwiftBar plugin for gf-claude-quota")
	fmt.Fprintln(w, "# Save this script to your xbar/SwiftBar plugins directory:")
	fmt.Fprintln(w, "#   ~/Library/Application Support/xbar/plugins/claude-quota.5m.sh")
	fmt.Fprintln(w, "# Then make it executable: chmod +x claude-quota.5m.sh")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "#!/bin/bash")
	fmt.Fprintf(w, "%s --xbar\n", binPath)
	return 0
}

func setupStatusLine(w, errw io.Writer, dryRun bool) int {
	binPath, err := FindBinaryPath()
	if err != nil {
		fmt.Fprintf(errw, "gf-claude-quota: %v\n", err)
		return 1
	}

	settingsFile := SettingsPath()
	if settingsFile == "" {
		fmt.Fprintln(errw, "gf-claude-quota: could not determine home directory")
		return 1
	}

	statusLineCmd := binPath + " --statusline"

	// Read existing settings
	settings := make(Settings)
	existingData, err := os.ReadFile(settingsFile)
	if err == nil {
		if err := json.Unmarshal(existingData, &settings); err != nil {
			fmt.Fprintf(errw, "gf-claude-quota: failed to parse %s: %v\n", settingsFile, err)
			return 1
		}
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(errw, "gf-claude-quota: failed to read %s: %v\n", settingsFile, err)
		return 1
	}

	// Check if already configured
	if existing, ok := settings["statusLine"]; ok {
		if existingStr, ok := existing.(string); ok && existingStr == statusLineCmd {
			fmt.Fprintln(w, "statusLine is already configured.")
			return 0
		}
	}

	// Set statusLine
	settings["statusLine"] = statusLineCmd

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		fmt.Fprintf(errw, "gf-claude-quota: failed to marshal settings: %v\n", err)
		return 1
	}
	newData = append(newData, '\n')

	if dryRun {
		fmt.Fprintln(w, "Dry run: the following changes would be made:")
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "File: %s\n", settingsFile)
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, string(newData))
		return 0
	}

	// Backup existing settings
	if len(existingData) > 0 {
		backupPath := settingsFile + ".backup." + time.Now().Format("20060102-150405")
		if err := os.WriteFile(backupPath, existingData, 0600); err != nil {
			fmt.Fprintf(errw, "gf-claude-quota: failed to create backup at %s: %v\n", backupPath, err)
			return 1
		}
		fmt.Fprintf(w, "Backup created: %s\n", backupPath)
	}

	// Ensure directory exists
	dir := filepath.Dir(settingsFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		fmt.Fprintf(errw, "gf-claude-quota: failed to create directory %s: %v\n", dir, err)
		return 1
	}

	// Write new settings
	if err := os.WriteFile(settingsFile, newData, 0600); err != nil {
		fmt.Fprintf(errw, "gf-claude-quota: failed to write %s: %v\n", settingsFile, err)
		return 1
	}

	fmt.Fprintf(w, "statusLine configured in %s\n", settingsFile)
	fmt.Fprintf(w, "  statusLine: %s\n", statusLineCmd)
	return 0
}
