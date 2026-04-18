package credentials

import (
	"errors"
	"testing"
)

// stubExitErr satisfies the exitCoder interface used by isClaudeRunningWithRunner.
type stubExitErr struct{ code int }

func (s *stubExitErr) Error() string { return "exit" }
func (s *stubExitErr) ExitCode() int { return s.code }

func TestIsClaudeRunningWithRunner(t *testing.T) {
	tests := []struct {
		name    string
		runner  CommandRunner
		want    bool
	}{
		{
			name: "single bare claude match",
			runner: func(name string, args ...string) ([]byte, error) {
				return []byte("2062 claude --dangerously-skip-permissions\n"), nil
			},
			want: true,
		},
		{
			name: "absolute path claude match",
			runner: func(name string, args ...string) ([]byte, error) {
				return []byte("3001 /usr/local/bin/claude chat\n"), nil
			},
			want: true,
		},
		{
			name: "multiple matches include at least one real claude",
			runner: func(name string, args ...string) ([]byte, error) {
				return []byte("100 gf-claude-quota\n200 /usr/local/bin/claude\n"), nil
			},
			want: true,
		},
		{
			name: "exit code 1 means no matches at all",
			runner: func(name string, args ...string) ([]byte, error) {
				return nil, &stubExitErr{code: 1}
			},
			want: false,
		},
		{
			name: "only non-claude binaries match substring",
			runner: func(name string, args ...string) ([]byte, error) {
				return []byte("100 gf-claude-quota\n200 claude-code-proxy\n"), nil
			},
			want: false,
		},
		{
			name: "pgrep binary missing — fail-safe to running",
			runner: func(name string, args ...string) ([]byte, error) {
				return nil, errors.New("exec: \"pgrep\": executable file not found")
			},
			want: true,
		},
		{
			name: "pgrep unexpected exit code — fail-safe to running",
			runner: func(name string, args ...string) ([]byte, error) {
				return nil, &stubExitErr{code: 2}
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isClaudeRunningWithRunner(tt.runner)
			if got != tt.want {
				t.Errorf("isClaudeRunningWithRunner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesClaudeBinary(t *testing.T) {
	tests := []struct {
		name    string
		cmdline string
		want    bool
	}{
		{"bare claude with args", "claude --dangerously-skip-permissions", true},
		{"absolute path", "/usr/local/bin/claude chat", true},
		{"just claude", "claude", true},
		{"gf-claude-quota excluded", "gf-claude-quota", false},
		{"claude-code-proxy excluded", "/opt/claude-code-proxy serve", false},
		{"empty", "", false},
		{"random process", "node /some/script.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesClaudeBinary(tt.cmdline)
			if got != tt.want {
				t.Errorf("matchesClaudeBinary(%q) = %v, want %v", tt.cmdline, got, tt.want)
			}
		})
	}
}
