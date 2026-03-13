package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWc(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLines int
		wantWords int
		wantBytes int
	}{
		{
			name:      "単一行",
			input:     "hello world\n",
			wantLines: 1,
			wantWords: 2,
			wantBytes: 12,
		},
		{
			name:      "複数行",
			input:     "line one\nline two\nline three\n",
			wantLines: 3,
			wantWords: 6,
			wantBytes: 29,
		},
		{
			name:      "空入力",
			input:     "",
			wantLines: 0,
			wantWords: 0,
			wantBytes: 0,
		},
		{
			name:      "空行のみ",
			input:     "\n\n\n",
			wantLines: 3,
			wantWords: 0,
			wantBytes: 3,
		},
		{
			name:      "タブ区切り",
			input:     "a\tb\tc\n",
			wantLines: 1,
			wantWords: 3,
			wantBytes: 6,
		},
		{
			name:      "連続スペース",
			input:     "  hello   world  \n",
			wantLines: 1,
			wantWords: 2,
			wantBytes: 18,
		},
		{
			name:      "マルチバイト文字",
			input:     "こんにちは 世界\n",
			wantLines: 1,
			wantWords: 2,
			wantBytes: 23,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			c, err := wc(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.lines != tt.wantLines {
				t.Errorf("lines: got %d, want %d", c.lines, tt.wantLines)
			}
			if c.words != tt.wantWords {
				t.Errorf("words: got %d, want %d", c.words, tt.wantWords)
			}
			if c.bytes != tt.wantBytes {
				t.Errorf("bytes: got %d, want %d", c.bytes, tt.wantBytes)
			}
		})
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"空文字列", "", 0},
		{"1単語", "hello", 1},
		{"複数単語", "hello world foo", 3},
		{"先頭末尾にスペース", "  hello  ", 1},
		{"タブ混在", "a\t b\t c", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countWords([]byte(tt.input))
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

// 統合テスト
func buildBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "gf-wc")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return binary
}

func runCmd(t *testing.T, binary string, args []string, stdin string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Stdin = strings.NewReader(stdin)
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
	return stdout.String(), stderr.String(), exitCode
}

func TestIntegration(t *testing.T) {
	binary := buildBinary(t)

	t.Run("stdin入力", func(t *testing.T) {
		stdout, _, code := runCmd(t, binary, nil, "hello world\nfoo bar baz\n")
		if code != 0 {
			t.Errorf("exit code: got %d, want 0", code)
		}
		if !strings.Contains(stdout, "2") && !strings.Contains(stdout, "5") {
			t.Errorf("unexpected output: %q", stdout)
		}
	})

	t.Run("ファイル入力", func(t *testing.T) {
		tmpDir := t.TempDir()
		f := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(f, []byte("one two three\nfour five\n"), 0644)

		stdout, _, code := runCmd(t, binary, []string{f}, "")
		if code != 0 {
			t.Errorf("exit code: got %d, want 0", code)
		}
		// 2 lines, 5 words, 24 bytes
		if !strings.Contains(stdout, "2") {
			t.Errorf("expected line count 2 in output: %q", stdout)
		}
		if !strings.Contains(stdout, "5") {
			t.Errorf("expected word count 5 in output: %q", stdout)
		}
	})

	t.Run("-l フラグ", func(t *testing.T) {
		stdout, _, code := runCmd(t, binary, []string{"-l"}, "a\nb\nc\n")
		if code != 0 {
			t.Errorf("exit code: got %d, want 0", code)
		}
		trimmed := strings.TrimSpace(stdout)
		if trimmed != "3" {
			t.Errorf("got %q, want 3", trimmed)
		}
	})

	t.Run("-w フラグ", func(t *testing.T) {
		stdout, _, code := runCmd(t, binary, []string{"-w"}, "one two three\n")
		if code != 0 {
			t.Errorf("exit code: got %d, want 0", code)
		}
		trimmed := strings.TrimSpace(stdout)
		if trimmed != "3" {
			t.Errorf("got %q, want 3", trimmed)
		}
	})

	t.Run("-c フラグ", func(t *testing.T) {
		stdout, _, code := runCmd(t, binary, []string{"-c"}, "hello\n")
		if code != 0 {
			t.Errorf("exit code: got %d, want 0", code)
		}
		trimmed := strings.TrimSpace(stdout)
		if trimmed != "6" {
			t.Errorf("got %q, want 6", trimmed)
		}
	})

	t.Run("--version", func(t *testing.T) {
		stdout, _, code := runCmd(t, binary, []string{"--version"}, "")
		if code != 0 {
			t.Errorf("exit code: got %d, want 0", code)
		}
		if !strings.Contains(stdout, "0.1.0") {
			t.Errorf("version not found: %q", stdout)
		}
	})

	t.Run("存在しないファイル", func(t *testing.T) {
		_, stderr, code := runCmd(t, binary, []string{"/nonexistent/file"}, "")
		if code != 1 {
			t.Errorf("exit code: got %d, want 1", code)
		}
		if !strings.Contains(stderr, "gf-wc:") {
			t.Errorf("expected error message: %q", stderr)
		}
	})

	t.Run("ハイフンでstdin", func(t *testing.T) {
		stdout, _, code := runCmd(t, binary, []string{"-"}, "test line\n")
		if code != 0 {
			t.Errorf("exit code: got %d, want 0", code)
		}
		if !strings.Contains(stdout, "1") {
			t.Errorf("expected line count in output: %q", stdout)
		}
	})
}
