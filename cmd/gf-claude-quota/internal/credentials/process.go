package credentials

import (
	"strings"
)

// exitCoder is the interface implemented by *exec.ExitError (and test stubs)
// that exposes the process exit code.
type exitCoder interface {
	ExitCode() int
}

// IsClaudeRunning reports whether any `claude` binary is currently running.
// Fails safe to true on any probe error, to avoid accidentally refreshing the
// OAuth token while Claude Code holds a live copy of the refresh_token in
// memory (which would force the user to re-login).
func IsClaudeRunning() bool {
	return isClaudeRunningWithRunner(DefaultCommandRunner)
}

func isClaudeRunningWithRunner(runner CommandRunner) bool {
	output, err := runner("pgrep", "-af", "claude")
	if err != nil {
		if ec, ok := err.(exitCoder); ok && ec.ExitCode() == 1 {
			return false // pgrep exit 1 = no match
		}
		return true // fail-safe for missing binary / perm errors / etc.
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		// Each line: "<pid> <cmdline>"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		if matchesClaudeBinary(parts[1]) {
			return true
		}
	}
	return false
}

// matchesClaudeBinary reports whether the first token of cmdline is a path
// whose basename equals "claude".
func matchesClaudeBinary(cmdline string) bool {
	cmdline = strings.TrimSpace(cmdline)
	if cmdline == "" {
		return false
	}
	first := cmdline
	if idx := strings.Index(cmdline, " "); idx >= 0 {
		first = cmdline[:idx]
	}
	base := first
	if idx := strings.LastIndex(first, "/"); idx >= 0 {
		base = first[idx+1:]
	}
	return base == "claude"
}
