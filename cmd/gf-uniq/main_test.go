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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			processReader(strings.NewReader(tt.input), w)
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
			code := run(args, stdin, &stdout, &stderr)

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
