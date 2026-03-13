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

func TestSplitBatches(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		n     int
		want  [][]string
	}{
		{"n=0 single batch", []string{"a", "b", "c"}, 0, [][]string{{"a", "b", "c"}}},
		{"n=1", []string{"a", "b", "c"}, 1, [][]string{{"a"}, {"b"}, {"c"}}},
		{"n=2 even", []string{"a", "b", "c", "d"}, 2, [][]string{{"a", "b"}, {"c", "d"}}},
		{"n=2 odd", []string{"a", "b", "c"}, 2, [][]string{{"a", "b"}, {"c"}}},
		{"n=5 larger than items", []string{"a", "b"}, 5, [][]string{{"a", "b"}}},
		{"n=3 exact", []string{"a", "b", "c"}, 3, [][]string{{"a", "b", "c"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitBatches(tt.items, tt.n)
			if len(got) != len(tt.want) {
				t.Fatalf("splitBatches len = %d, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if len(got[i]) != len(tt.want[i]) {
					t.Fatalf("batch[%d] len = %d, want %d", i, len(got[i]), len(tt.want[i]))
				}
				for j := range got[i] {
					if got[i][j] != tt.want[i][j] {
						t.Errorf("batch[%d][%d] = %q, want %q", i, j, got[i][j], tt.want[i][j])
					}
				}
			}
		})
	}
}

func TestRunWithN(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		stdin     string
		wantCode  int
		wantCalls int
		wantArgs  [][]string
	}{
		{
			name:      "n=1 splits into individual calls",
			args:      []string{"-n", "1", "echo"},
			stdin:     "a\nb\nc\n",
			wantCode:  0,
			wantCalls: 3,
			wantArgs:  [][]string{{"a"}, {"b"}, {"c"}},
		},
		{
			name:      "n=2 splits into batches",
			args:      []string{"-n", "2", "echo"},
			stdin:     "a b c d e\n",
			wantCode:  0,
			wantCalls: 3,
			wantArgs:  [][]string{{"a", "b"}, {"c", "d"}, {"e"}},
		},
		{
			name:      "n=0 means all in one call",
			args:      []string{"-n", "0", "echo"},
			stdin:     "a b c\n",
			wantCode:  0,
			wantCalls: 1,
			wantArgs:  [][]string{{"a", "b", "c"}},
		},
		{
			name:      "n with extra command args",
			args:      []string{"-n", "2", "grep", "-l"},
			stdin:     "f1 f2 f3\n",
			wantCode:  0,
			wantCalls: 2,
			wantArgs:  [][]string{{"-l", "f1", "f2"}, {"-l", "f3"}},
		},
		{
			name:     "negative n",
			args:     []string{"-n", "-1", "echo"},
			stdin:    "a\n",
			wantCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecutor{}
			var stdout, stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(tt.stdin), &stdout, &stderr, mock)

			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d (stderr: %s)", code, tt.wantCode, stderr.String())
			}
			if tt.wantCode != 0 {
				return
			}

			if len(mock.calls) != tt.wantCalls {
				t.Fatalf("calls = %d, want %d: %v", len(mock.calls), tt.wantCalls, mock.calls)
			}

			for i, wantArgs := range tt.wantArgs {
				gotArgs := mock.calls[i].Args
				if len(gotArgs) != len(wantArgs) {
					t.Fatalf("call[%d] args = %v, want %v", i, gotArgs, wantArgs)
				}
				for j := range gotArgs {
					if gotArgs[j] != wantArgs[j] {
						t.Errorf("call[%d] args[%d] = %q, want %q", i, j, gotArgs[j], wantArgs[j])
					}
				}
			}
		})
	}
}

func TestRunWithP(t *testing.T) {
	t.Run("P=2 parallel execution", func(t *testing.T) {
		mock := &mockExecutor{}
		var stdout, stderr bytes.Buffer
		code := run([]string{"-n", "1", "-P", "2", "echo"}, strings.NewReader("a\nb\nc\n"), &stdout, &stderr, mock)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
		}
		if len(mock.calls) != 3 {
			t.Fatalf("calls = %d, want 3", len(mock.calls))
		}
	})

	t.Run("P=4 more workers than batches", func(t *testing.T) {
		mock := &mockExecutor{}
		var stdout, stderr bytes.Buffer
		code := run([]string{"-n", "1", "-P", "4", "echo"}, strings.NewReader("a\nb\n"), &stdout, &stderr, mock)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if len(mock.calls) != 2 {
			t.Fatalf("calls = %d, want 2", len(mock.calls))
		}
	})

	t.Run("P=0 invalid", func(t *testing.T) {
		mock := &mockExecutor{}
		var stdout, stderr bytes.Buffer
		code := run([]string{"-P", "0", "echo"}, strings.NewReader("a\n"), &stdout, &stderr, mock)
		if code != 2 {
			t.Fatalf("exit code = %d, want 2", code)
		}
	})

	t.Run("P with error propagation", func(t *testing.T) {
		mock := &mockExecutor{err: fmt.Errorf("fail")}
		var stdout, stderr bytes.Buffer
		code := run([]string{"-n", "1", "-P", "2", "cmd"}, strings.NewReader("a\nb\n"), &stdout, &stderr, mock)
		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
	})

	t.Run("P=1 sequential with n", func(t *testing.T) {
		mock := &mockExecutor{}
		var stdout, stderr bytes.Buffer
		code := run([]string{"-n", "2", "-P", "1", "echo"}, strings.NewReader("a b c d\n"), &stdout, &stderr, mock)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if len(mock.calls) != 2 {
			t.Fatalf("calls = %d, want 2", len(mock.calls))
		}
	})
}

func TestReadItemsNull(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple null-separated", "a\x00b\x00c\x00", []string{"a", "b", "c"}},
		{"no trailing null", "a\x00b\x00c", []string{"a", "b", "c"}},
		{"empty items skipped", "a\x00\x00b\x00", []string{"a", "b"}},
		{"spaces preserved", "hello world\x00foo bar\x00", []string{"hello world", "foo bar"}},
		{"newlines in items", "line1\nline2\x00line3\x00", []string{"line1\nline2", "line3"}},
		{"empty input", "", nil},
		{"multibyte", "日本語\x00テスト\x00", []string{"日本語", "テスト"}},
		{"paths with spaces", "/path/to/my file.txt\x00/other dir/foo\x00", []string{"/path/to/my file.txt", "/other dir/foo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readItemsNull(strings.NewReader(tt.input))
			if len(got) != len(tt.want) {
				t.Fatalf("readItemsNull() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestShellJoin(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		args []string
		want string
	}{
		{"simple", "echo", []string{"hello", "world"}, "echo hello world"},
		{"with spaces", "echo", []string{"hello world"}, "echo 'hello world'"},
		{"with quotes", "echo", []string{"it's"}, "echo 'it'\\''s'"},
		{"empty arg", "echo", []string{""}, "echo ''"},
		{"special chars", "rm", []string{"-rf", "/tmp/my dir"}, "rm -rf '/tmp/my dir'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellJoin(tt.cmd, tt.args)
			if got != tt.want {
				t.Errorf("shellJoin() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunWithNullDelim(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		stdin    string
		wantCode int
		wantCall *mockCall
	}{
		{
			name:     "null delim basic",
			args:     []string{"-0", "echo"},
			stdin:    "hello world\x00foo\x00",
			wantCode: 0,
			wantCall: &mockCall{Name: "echo", Args: []string{"hello world", "foo"}},
		},
		{
			name:     "null delim with newlines",
			args:     []string{"-0", "echo"},
			stdin:    "line1\nline2\x00line3\x00",
			wantCode: 0,
			wantCall: &mockCall{Name: "echo", Args: []string{"line1\nline2", "line3"}},
		},
		{
			name:     "null delim with -n",
			args:     []string{"-0", "-n", "1", "echo"},
			stdin:    "a\x00b\x00c\x00",
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecutor{}
			var stdout, stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(tt.stdin), &stdout, &stderr, mock)

			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, tt.wantCode, stderr.String())
			}

			if tt.wantCall != nil {
				if len(mock.calls) < 1 {
					t.Fatalf("expected at least 1 call, got %d", len(mock.calls))
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
			}
		})
	}
}

func TestRunDryRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		stdin      string
		wantCode   int
		wantOutput string
		wantCalls  int
	}{
		{
			name:       "dry-run basic",
			args:       []string{"--dry-run", "rm"},
			stdin:      "a.txt\nb.txt\n",
			wantCode:   0,
			wantOutput: "rm a.txt b.txt\n",
			wantCalls:  0,
		},
		{
			name:       "dry-run with -n",
			args:       []string{"--dry-run", "-n", "1", "rm"},
			stdin:      "a.txt\nb.txt\n",
			wantCode:   0,
			wantOutput: "rm a.txt\nrm b.txt\n",
			wantCalls:  0,
		},
		{
			name:       "dry-run with spaces in args",
			args:       []string{"--dry-run", "rm"},
			stdin:      "\"my file.txt\"\n",
			wantCode:   0,
			wantOutput: "rm 'my file.txt'\n",
			wantCalls:  0,
		},
		{
			name:       "dry-run empty stdin",
			args:       []string{"--dry-run", "echo"},
			stdin:      "",
			wantCode:   0,
			wantOutput: "",
			wantCalls:  0,
		},
		{
			name:       "dry-run with -0",
			args:       []string{"--dry-run", "-0", "echo"},
			stdin:      "hello world\x00foo\x00",
			wantCode:   0,
			wantOutput: "echo 'hello world' foo\n",
			wantCalls:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecutor{}
			var stdout, stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(tt.stdin), &stdout, &stderr, mock)

			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, tt.wantCode, stderr.String())
			}
			if len(mock.calls) != tt.wantCalls {
				t.Errorf("calls = %d, want %d (should not execute)", len(mock.calls), tt.wantCalls)
			}
			if stdout.String() != tt.wantOutput {
				t.Errorf("output = %q, want %q", stdout.String(), tt.wantOutput)
			}
		})
	}
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

	t.Run("n=1 real echo", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-n", "1", "echo"}, strings.NewReader("hello world\n"), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
		}
		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if len(lines) != 2 {
			t.Fatalf("lines = %d, want 2: %v", len(lines), lines)
		}
		if strings.TrimSpace(lines[0]) != "hello" {
			t.Errorf("line[0] = %q, want %q", lines[0], "hello")
		}
		if strings.TrimSpace(lines[1]) != "world" {
			t.Errorf("line[1] = %q, want %q", lines[1], "world")
		}
	})

	t.Run("n=2 real echo batches", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-n", "2", "echo"}, strings.NewReader("a b c d e\n"), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if len(lines) != 3 {
			t.Fatalf("lines = %d, want 3: %v", len(lines), lines)
		}
	})

	t.Run("null delim real echo", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-0", "echo"}, strings.NewReader("hello world\x00foo bar\x00"), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
		}
		got := strings.TrimSpace(stdout.String())
		if got != "hello world foo bar" {
			t.Errorf("output = %q, want %q", got, "hello world foo bar")
		}
	})

	t.Run("dry-run real", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"--dry-run", "rm", "-f"}, strings.NewReader("a.txt\nb.txt\n"), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		got := strings.TrimSpace(stdout.String())
		if got != "rm -f a.txt b.txt" {
			t.Errorf("output = %q, want %q", got, "rm -f a.txt b.txt")
		}
	})

	t.Run("dry-run with n=1", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"--dry-run", "-n", "1", "echo"}, strings.NewReader("a b c\n"), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if len(lines) != 3 {
			t.Fatalf("lines = %d, want 3: %v", len(lines), lines)
		}
	})

	t.Run("null delim with -n=1 real", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-0", "-n", "1", "echo"}, strings.NewReader("a\x00b\x00c\x00"), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if len(lines) != 3 {
			t.Fatalf("lines = %d, want 3: %v", len(lines), lines)
		}
	})

	t.Run("P=2 real parallel echo", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-n", "1", "-P", "2", "echo"}, strings.NewReader("a\nb\nc\n"), &stdout, &stderr, realExecutor{})
		if code != 0 {
			t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
		}
		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if len(lines) != 3 {
			t.Fatalf("lines = %d, want 3: %v", len(lines), lines)
		}
		// Order may vary due to parallelism, but all items should be present.
		got := make(map[string]bool)
		for _, l := range lines {
			got[strings.TrimSpace(l)] = true
		}
		for _, want := range []string{"a", "b", "c"} {
			if !got[want] {
				t.Errorf("missing output %q in %v", want, lines)
			}
		}
	})
}
