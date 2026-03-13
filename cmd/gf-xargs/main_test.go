package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

// mockExecutor records calls and can simulate errors.
type mockExecutor struct {
	calls []mockCall
	err   error
}

type mockCall struct {
	Name string
	Args []string
}

func (m *mockExecutor) Execute(name string, args []string, stdout, stderr io.Writer) error {
	argsCopy := make([]string, len(args))
	copy(argsCopy, args)
	m.calls = append(m.calls, mockCall{Name: name, Args: argsCopy})
	if m.err != nil {
		return m.err
	}
	// Simulate echo: print args joined by space.
	if name == "echo" {
		fmt.Fprintln(stdout, strings.Join(args, " "))
	}
	return nil
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{"simple words", "foo bar baz", []string{"foo", "bar", "baz"}},
		{"multiple spaces", "  foo   bar  ", []string{"foo", "bar"}},
		{"tabs", "foo\tbar", []string{"foo", "bar"}},
		{"double quotes", `"hello world" foo`, []string{"hello world", "foo"}},
		{"single quotes", `'hello world' foo`, []string{"hello world", "foo"}},
		{"empty string", "", nil},
		{"only spaces", "   ", nil},
		{"mixed quotes", `"foo bar" 'baz qux'`, []string{"foo bar", "baz qux"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitArgs(tt.line)
			if len(got) != len(tt.want) {
				t.Fatalf("splitArgs(%q) = %v, want %v", tt.line, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitArgs(%q)[%d] = %q, want %q", tt.line, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		stdin    string
		wantCode int
		wantCall *mockCall
		execErr  error
	}{
		{
			name:     "default echo command",
			args:     []string{},
			stdin:    "hello world\n",
			wantCode: 0,
			wantCall: &mockCall{Name: "echo", Args: []string{"hello", "world"}},
		},
		{
			name:     "explicit command",
			args:     []string{"grep", "-l", "TODO"},
			stdin:    "file1.go\nfile2.go\n",
			wantCode: 0,
			wantCall: &mockCall{Name: "grep", Args: []string{"-l", "TODO", "file1.go", "file2.go"}},
		},
		{
			name:     "empty stdin",
			args:     []string{"echo"},
			stdin:    "",
			wantCode: 0,
			wantCall: nil, // no execution
		},
		{
			name:     "command failure",
			args:     []string{"false"},
			stdin:    "arg1\n",
			wantCode: 1,
			wantCall: &mockCall{Name: "false", Args: []string{"arg1"}},
			execErr:  fmt.Errorf("exit status 1"),
		},
		{
			name:     "version flag",
			args:     []string{"--version"},
			stdin:    "",
			wantCode: 0,
			wantCall: nil,
		},
		{
			name:     "multiple lines",
			args:     []string{"rm"},
			stdin:    "a.txt\nb.txt\nc.txt\n",
			wantCode: 0,
			wantCall: &mockCall{Name: "rm", Args: []string{"a.txt", "b.txt", "c.txt"}},
		},
		{
			name:     "quoted args in stdin",
			args:     []string{"echo"},
			stdin:    `"hello world" foo` + "\n",
			wantCode: 0,
			wantCall: &mockCall{Name: "echo", Args: []string{"hello world", "foo"}},
		},
		{
			name:     "multiple words per line",
			args:     []string{"echo"},
			stdin:    "foo bar\nbaz qux\n",
			wantCode: 0,
			wantCall: &mockCall{Name: "echo", Args: []string{"foo", "bar", "baz", "qux"}},
		},
		{
			name:     "unknown flag",
			args:     []string{"--unknown"},
			stdin:    "",
			wantCode: 2,
			wantCall: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecutor{err: tt.execErr}
			var stdout, stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(tt.stdin), &stdout, &stderr, mock)

			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, tt.wantCode, stderr.String())
			}

			if tt.wantCall == nil {
				if len(mock.calls) > 0 {
					t.Errorf("expected no calls, got %v", mock.calls)
				}
				return
			}

			if len(mock.calls) != 1 {
				t.Fatalf("expected 1 call, got %d: %v", len(mock.calls), mock.calls)
			}

			call := mock.calls[0]
			if call.Name != tt.wantCall.Name {
				t.Errorf("command = %q, want %q", call.Name, tt.wantCall.Name)
			}
			if len(call.Args) != len(tt.wantCall.Args) {
				t.Fatalf("args = %v, want %v", call.Args, tt.wantCall.Args)
			}
			for i := range call.Args {
				if call.Args[i] != tt.wantCall.Args[i] {
					t.Errorf("args[%d] = %q, want %q", i, call.Args[i], tt.wantCall.Args[i])
				}
			}
		})
	}
}

// Edge case tests
func TestEdgeCases(t *testing.T) {
	t.Run("multibyte input", func(t *testing.T) {
		mock := &mockExecutor{}
		var stdout, stderr bytes.Buffer
		code := run([]string{"echo"}, strings.NewReader("こんにちは 世界\n"), &stdout, &stderr, mock)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if len(mock.calls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(mock.calls))
		}
		if mock.calls[0].Args[0] != "こんにちは" || mock.calls[0].Args[1] != "世界" {
			t.Errorf("args = %v, want [こんにちは 世界]", mock.calls[0].Args)
		}
	})

	t.Run("lines with only whitespace", func(t *testing.T) {
		mock := &mockExecutor{}
		var stdout, stderr bytes.Buffer
		code := run([]string{"echo"}, strings.NewReader("  \n  \n"), &stdout, &stderr, mock)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0 (empty input)", code)
		}
		if len(mock.calls) != 0 {
			t.Errorf("expected no calls for whitespace-only input, got %d", len(mock.calls))
		}
	})

	t.Run("large input", func(t *testing.T) {
		var sb strings.Builder
		for i := 0; i < 1000; i++ {
			fmt.Fprintf(&sb, "file%d.txt\n", i)
		}
		mock := &mockExecutor{}
		var stdout, stderr bytes.Buffer
		code := run([]string{"rm"}, strings.NewReader(sb.String()), &stdout, &stderr, mock)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if len(mock.calls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(mock.calls))
		}
		if len(mock.calls[0].Args) != 1000 {
			t.Errorf("args count = %d, want 1000", len(mock.calls[0].Args))
		}
	})

	t.Run("error message on stderr", func(t *testing.T) {
		mock := &mockExecutor{err: fmt.Errorf("command not found")}
		var stdout, stderr bytes.Buffer
		code := run([]string{"nonexistent"}, strings.NewReader("arg\n"), &stdout, &stderr, mock)
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(stderr.String(), "nonexistent") {
			t.Errorf("stderr = %q, want to contain command name", stderr.String())
		}
	})
}

// Integration-style tests using real executor
func TestIntegration(t *testing.T) {
	t.Run("echo with real executor", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"echo", "hello"}, strings.NewReader("world\n"), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
		}
		got := strings.TrimSpace(stdout.String())
		if got != "hello world" {
			t.Errorf("output = %q, want %q", got, "hello world")
		}
	})

	t.Run("echo multiple items", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"echo"}, strings.NewReader("a b c\nd e f\n"), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		got := strings.TrimSpace(stdout.String())
		if got != "a b c d e f" {
			t.Errorf("output = %q, want %q", got, "a b c d e f")
		}
	})

	t.Run("command not found", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"__nonexistent_command_12345__"}, strings.NewReader("arg\n"), &stdout, &stderr, realExecutor{})
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
	})

	t.Run("version output", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"--version"}, strings.NewReader(""), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stdout.String(), version) {
			t.Errorf("output = %q, want to contain version", stdout.String())
		}
	})

	t.Run("empty stdin no execution", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"echo"}, strings.NewReader(""), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if stdout.String() != "" {
			t.Errorf("output = %q, want empty", stdout.String())
		}
	})

	t.Run("stdin pipe with printf", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"printf", "%s,"}, strings.NewReader("hello\nworld\n"), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
		}
		// printf "%s," hello world → "hello,world," or similar depending on printf behavior
	})

	t.Run("default echo no command specified", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{}, strings.NewReader("foo bar\n"), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		got := strings.TrimSpace(stdout.String())
		if got != "foo bar" {
			t.Errorf("output = %q, want %q", got, "foo bar")
		}
	})

	t.Run("multibyte echo", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"echo"}, strings.NewReader("日本語 テスト\n"), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		got := strings.TrimSpace(stdout.String())
		if got != "日本語 テスト" {
			t.Errorf("output = %q, want %q", got, "日本語 テスト")
		}
	})
}
