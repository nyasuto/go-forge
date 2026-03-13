package main

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- Unit Tests ---

func TestProcessReader(t *testing.T) {
	tests := []struct {
		name  string
		input string
		opts  uniqOptions
		want  string
	}{
		{
			name:  "adjacent duplicates removed",
			input: "aaa\naaa\nbbb\nbbb\naaa\n",
			want:  "aaa\nbbb\naaa\n",
		},
		{
			name:  "no duplicates",
			input: "aaa\nbbb\nccc\n",
			want:  "aaa\nbbb\nccc\n",
		},
		{
			name:  "all same lines",
			input: "xxx\nxxx\nxxx\n",
			want:  "xxx\n",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "single line",
			input: "hello\n",
			want:  "hello\n",
		},
		{
			name:  "multibyte characters",
			input: "こんにちは\nこんにちは\nさようなら\nさようなら\n",
			want:  "こんにちは\nさようなら\n",
		},
		{
			name:  "empty lines as duplicates",
			input: "\n\n\nabc\nabc\n\n",
			want:  "\nabc\n\n",
		},
		// -c count option
		{
			name:  "count: adjacent duplicates",
			input: "aaa\naaa\nbbb\nbbb\nbbb\nccc\n",
			opts:  uniqOptions{count: true},
			want:  "      2 aaa\n      3 bbb\n      1 ccc\n",
		},
		{
			name:  "count: no duplicates",
			input: "a\nb\nc\n",
			opts:  uniqOptions{count: true},
			want:  "      1 a\n      1 b\n      1 c\n",
		},
		{
			name:  "count: single line",
			input: "only\n",
			opts:  uniqOptions{count: true},
			want:  "      1 only\n",
		},
		// -d duplicates only
		{
			name:  "duplicates: show only duplicated lines",
			input: "aaa\naaa\nbbb\nccc\nccc\nccc\n",
			opts:  uniqOptions{duplicates: true},
			want:  "aaa\nccc\n",
		},
		{
			name:  "duplicates: no duplicates means no output",
			input: "a\nb\nc\n",
			opts:  uniqOptions{duplicates: true},
			want:  "",
		},
		{
			name:  "duplicates: all same",
			input: "x\nx\nx\n",
			opts:  uniqOptions{duplicates: true},
			want:  "x\n",
		},
		// -i ignore case
		{
			name:  "ignore case: adjacent case-different duplicates",
			input: "Hello\nhello\nHELLO\nWorld\n",
			opts:  uniqOptions{ignoreCase: true},
			want:  "Hello\nWorld\n",
		},
		{
			name:  "ignore case: no match without -i",
			input: "Hello\nhello\n",
			opts:  uniqOptions{ignoreCase: false},
			want:  "Hello\nhello\n",
		},
		{
			name:  "ignore case: multibyte",
			input: "Straße\nstraße\nSTRASSE\n",
			opts:  uniqOptions{ignoreCase: true},
			want:  "Straße\nSTRASSE\n",
		},
		// combined options
		{
			name:  "count + duplicates",
			input: "a\na\nb\nc\nc\nc\n",
			opts:  uniqOptions{count: true, duplicates: true},
			want:  "      2 a\n      3 c\n",
		},
		{
			name:  "count + ignore case",
			input: "Hello\nhello\nWorld\n",
			opts:  uniqOptions{count: true, ignoreCase: true},
			want:  "      2 Hello\n      1 World\n",
		},
		{
			name:  "duplicates + ignore case",
			input: "ABC\nabc\nDEF\n",
			opts:  uniqOptions{duplicates: true, ignoreCase: true},
			want:  "ABC\n",
		},
		{
			name:  "all three options",
			input: "Foo\nfoo\nFOO\nbar\nBAZ\nbaz\n",
			opts:  uniqOptions{count: true, duplicates: true, ignoreCase: true},
			want:  "      3 Foo\n      2 BAZ\n",
		},
		{
			name:  "empty input with options",
			input: "",
			opts:  uniqOptions{count: true, duplicates: true, ignoreCase: true},
			want:  "",
		},
		// --global option
		{
			name:  "global: non-adjacent duplicates removed",
			input: "a\nb\na\nc\nb\n",
			opts:  uniqOptions{global: true},
			want:  "a\nb\nc\n",
		},
		{
			name:  "global: all unique",
			input: "a\nb\nc\n",
			opts:  uniqOptions{global: true},
			want:  "a\nb\nc\n",
		},
		{
			name:  "global: all same",
			input: "x\nx\nx\n",
			opts:  uniqOptions{global: true},
			want:  "x\n",
		},
		{
			name:  "global: empty input",
			input: "",
			opts:  uniqOptions{global: true},
			want:  "",
		},
		{
			name:  "global: single line",
			input: "hello\n",
			opts:  uniqOptions{global: true},
			want:  "hello\n",
		},
		{
			name:  "global: multibyte",
			input: "あ\nい\nあ\nう\nい\n",
			opts:  uniqOptions{global: true},
			want:  "あ\nい\nう\n",
		},
		{
			name:  "global: preserves first occurrence order",
			input: "c\nb\na\nc\nb\na\n",
			opts:  uniqOptions{global: true},
			want:  "c\nb\na\n",
		},
		// --global combined options
		{
			name:  "global + count",
			input: "a\nb\na\nc\nb\na\n",
			opts:  uniqOptions{global: true, count: true},
			want:  "      3 a\n      2 b\n      1 c\n",
		},
		{
			name:  "global + duplicates",
			input: "a\nb\na\nc\n",
			opts:  uniqOptions{global: true, duplicates: true},
			want:  "a\n",
		},
		{
			name:  "global + ignore case",
			input: "Hello\nworld\nhello\nWORLD\n",
			opts:  uniqOptions{global: true, ignoreCase: true},
			want:  "Hello\nworld\n",
		},
		{
			name:  "global + count + duplicates",
			input: "a\nb\na\nc\nb\n",
			opts:  uniqOptions{global: true, count: true, duplicates: true},
			want:  "      2 a\n      2 b\n",
		},
		{
			name:  "global + count + duplicates + ignore case",
			input: "Foo\nbar\nfoo\nBAR\nbaz\n",
			opts:  uniqOptions{global: true, count: true, duplicates: true, ignoreCase: true},
			want:  "      2 Foo\n      2 bar\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			processReader(strings.NewReader(tt.input), w, tt.opts)
			w.Flush()
			got := buf.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		stdin    string
		files    map[string]string
		opts     uniqOptions
		wantOut  string
		wantErr  string
		wantCode int
	}{
		{
			name:     "stdin input",
			args:     nil,
			stdin:    "aa\naa\nbb\n",
			wantOut:  "aa\nbb\n",
			wantCode: 0,
		},
		{
			name:     "stdin via hyphen",
			args:     []string{"-"},
			stdin:    "cc\ncc\ndd\n",
			wantOut:  "cc\ndd\n",
			wantCode: 0,
		},
		{
			name:     "file input",
			args:     []string{"testfile"},
			files:    map[string]string{"testfile": "xx\nxx\nyy\nyy\nzz\n"},
			wantOut:  "xx\nyy\nzz\n",
			wantCode: 0,
		},
		{
			name:     "nonexistent file",
			args:     []string{"no_such_file"},
			wantOut:  "",
			wantErr:  "no such file or directory",
			wantCode: 1,
		},
		{
			name:     "mixed: valid file and nonexistent",
			args:     []string{"good", "bad"},
			files:    map[string]string{"good": "a\na\nb\n"},
			wantOut:  "a\nb\n",
			wantErr:  "no such file or directory",
			wantCode: 1,
		},
		{
			name:     "large repeated input",
			args:     nil,
			stdin:    strings.Repeat("same\n", 10000),
			wantOut:  "same\n",
			wantCode: 0,
		},
		{
			name:     "non-adjacent duplicates preserved",
			args:     nil,
			stdin:    "a\nb\na\nb\n",
			wantOut:  "a\nb\na\nb\n",
			wantCode: 0,
		},
		// Tier 2: run-level tests with options
		{
			name:     "count option via run",
			args:     nil,
			stdin:    "a\na\nb\n",
			opts:     uniqOptions{count: true},
			wantOut:  "      2 a\n      1 b\n",
			wantCode: 0,
		},
		{
			name:     "duplicates option via run",
			args:     nil,
			stdin:    "a\na\nb\nc\nc\n",
			opts:     uniqOptions{duplicates: true},
			wantOut:  "a\nc\n",
			wantCode: 0,
		},
		{
			name:     "ignore case option via run",
			args:     nil,
			stdin:    "AAA\naaa\nbbb\n",
			opts:     uniqOptions{ignoreCase: true},
			wantOut:  "AAA\nbbb\n",
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create test files
			args := make([]string, len(tt.args))
			for i, a := range tt.args {
				if content, ok := tt.files[a]; ok {
					path := filepath.Join(tmpDir, a)
					os.WriteFile(path, []byte(content), 0644)
					args[i] = path
				} else {
					if a == "-" {
						args[i] = a
					} else {
						args[i] = filepath.Join(tmpDir, a)
					}
				}
			}

			var stdout, stderr bytes.Buffer
			stdin := strings.NewReader(tt.stdin)
			code := run(args, stdin, &stdout, &stderr, tt.opts)

			if code != tt.wantCode {
				t.Errorf("exit code: got %d, want %d", code, tt.wantCode)
			}
			if stdout.String() != tt.wantOut {
				t.Errorf("stdout: got %q, want %q", stdout.String(), tt.wantOut)
			}
			if tt.wantErr != "" && !strings.Contains(stderr.String(), tt.wantErr) {
				t.Errorf("stderr: got %q, want to contain %q", stderr.String(), tt.wantErr)
			}
		})
	}
}

// --- Integration Tests ---

func TestIntegration(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "gf-uniq")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	tests := []struct {
		name     string
		args     []string
		stdin    string
		file     string
		wantOut  string
		wantErr  string
		wantCode int
	}{
		{
			name:     "basic stdin dedup",
			stdin:    "foo\nfoo\nbar\nbar\nbaz\n",
			wantOut:  "foo\nbar\nbaz\n",
			wantCode: 0,
		},
		{
			name:     "file dedup",
			file:     "a\na\nb\nb\nc\n",
			wantOut:  "a\nb\nc\n",
			wantCode: 0,
		},
		{
			name:     "version flag",
			args:     []string{"--version"},
			wantOut:  "gf-uniq version 0.1.0\n",
			wantCode: 0,
		},
		{
			name:     "nonexistent file",
			args:     []string{"/tmp/gf_uniq_nonexistent_12345"},
			wantErr:  "gf-uniq:",
			wantCode: 1,
		},
		{
			name:     "empty stdin",
			stdin:    "",
			wantOut:  "",
			wantCode: 0,
		},
		{
			name:     "pipe: echo with duplicates",
			stdin:    "x\nx\ny\ny\nz\nz\n",
			wantOut:  "x\ny\nz\n",
			wantCode: 0,
		},
		{
			name:     "multibyte stdin",
			stdin:    "日本語\n日本語\n英語\n",
			wantOut:  "日本語\n英語\n",
			wantCode: 0,
		},
		{
			name:     "non-adjacent not removed",
			stdin:    "a\nb\na\n",
			wantOut:  "a\nb\na\n",
			wantCode: 0,
		},
		// Tier 2 integration tests
		{
			name:     "-c count flag",
			args:     []string{"-c"},
			stdin:    "aaa\naaa\nbbb\nccc\nccc\nccc\n",
			wantOut:  "      2 aaa\n      1 bbb\n      3 ccc\n",
			wantCode: 0,
		},
		{
			name:     "-d duplicates only flag",
			args:     []string{"-d"},
			stdin:    "aaa\naaa\nbbb\nccc\nccc\n",
			wantOut:  "aaa\nccc\n",
			wantCode: 0,
		},
		{
			name:     "-i case insensitive flag",
			args:     []string{"-i"},
			stdin:    "Hello\nhello\nHELLO\nWorld\n",
			wantOut:  "Hello\nWorld\n",
			wantCode: 0,
		},
		{
			name:     "-c -d combined",
			args:     []string{"-c", "-d"},
			stdin:    "a\na\nb\nc\nc\nc\n",
			wantOut:  "      2 a\n      3 c\n",
			wantCode: 0,
		},
		{
			name:     "-c -i combined",
			args:     []string{"-c", "-i"},
			stdin:    "Foo\nfoo\nBar\n",
			wantOut:  "      2 Foo\n      1 Bar\n",
			wantCode: 0,
		},
		{
			name:     "-d -i combined",
			args:     []string{"-d", "-i"},
			stdin:    "ABC\nabc\nDEF\n",
			wantOut:  "ABC\n",
			wantCode: 0,
		},
		{
			name:     "-c -d -i all combined",
			args:     []string{"-c", "-d", "-i"},
			stdin:    "Foo\nfoo\nFOO\nbar\nBAZ\nbaz\n",
			wantOut:  "      3 Foo\n      2 BAZ\n",
			wantCode: 0,
		},
		{
			name:     "-c with file input",
			file:     "x\nx\ny\ny\ny\nz\n",
			args:     []string{"-c"},
			wantOut:  "      2 x\n      3 y\n      1 z\n",
			wantCode: 0,
		},
		{
			name:     "-d with empty input",
			args:     []string{"-d"},
			stdin:    "",
			wantOut:  "",
			wantCode: 0,
		},
		// Tier 3 integration tests: --global
		{
			name:     "--global removes non-adjacent duplicates",
			args:     []string{"--global"},
			stdin:    "a\nb\na\nc\nb\n",
			wantOut:  "a\nb\nc\n",
			wantCode: 0,
		},
		{
			name:     "--global with file input",
			args:     []string{"--global"},
			file:     "x\ny\nz\nx\ny\n",
			wantOut:  "x\ny\nz\n",
			wantCode: 0,
		},
		{
			name:     "--global -c count",
			args:     []string{"--global", "-c"},
			stdin:    "a\nb\na\nc\nb\na\n",
			wantOut:  "      3 a\n      2 b\n      1 c\n",
			wantCode: 0,
		},
		{
			name:     "--global -d duplicates only",
			args:     []string{"--global", "-d"},
			stdin:    "a\nb\nc\na\n",
			wantOut:  "a\n",
			wantCode: 0,
		},
		{
			name:     "--global -i case insensitive",
			args:     []string{"--global", "-i"},
			stdin:    "Hello\nworld\nhello\nWORLD\n",
			wantOut:  "Hello\nworld\n",
			wantCode: 0,
		},
		{
			name:     "--global -c -d -i all combined",
			args:     []string{"--global", "-c", "-d", "-i"},
			stdin:    "Foo\nbar\nfoo\nBAR\nbaz\n",
			wantOut:  "      2 Foo\n      2 bar\n",
			wantCode: 0,
		},
		{
			name:     "--global empty input",
			args:     []string{"--global"},
			stdin:    "",
			wantOut:  "",
			wantCode: 0,
		},
		{
			name:     "--global multibyte",
			args:     []string{"--global"},
			stdin:    "日本\n英語\n日本\n",
			wantOut:  "日本\n英語\n",
			wantCode: 0,
		},
		{
			name:     "--global large input",
			args:     []string{"--global"},
			stdin:    strings.Repeat("a\nb\n", 5000),
			wantOut:  "a\nb\n",
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			if tt.file != "" {
				tmpFile := filepath.Join(t.TempDir(), "input.txt")
				os.WriteFile(tmpFile, []byte(tt.file), 0644)
				args = append(args, tmpFile)
			}

			cmd := exec.Command(binary, args...)
			cmd.Stdin = strings.NewReader(tt.stdin)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
			}

			if exitCode != tt.wantCode {
				t.Errorf("exit code: got %d, want %d (stderr: %s)", exitCode, tt.wantCode, stderr.String())
			}
			if stdout.String() != tt.wantOut {
				t.Errorf("stdout: got %q, want %q", stdout.String(), tt.wantOut)
			}
			if tt.wantErr != "" && !strings.Contains(stderr.String(), tt.wantErr) {
				t.Errorf("stderr: got %q, want to contain %q", stderr.String(), tt.wantErr)
			}
		})
	}
}
