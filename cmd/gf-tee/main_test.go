package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		input      string
		wantStdout string
		wantExit   int
		checkFiles map[string]string // filename -> expected content
	}{
		// 正常系
		{
			name:       "stdin to stdout only (no files)",
			args:       []string{},
			input:      "hello world\n",
			wantStdout: "hello world\n",
			wantExit:   0,
		},
		{
			name:       "stdin to stdout and one file",
			args:       []string{"out.txt"},
			input:      "line1\nline2\n",
			wantStdout: "line1\nline2\n",
			wantExit:   0,
			checkFiles: map[string]string{"out.txt": "line1\nline2\n"},
		},
		{
			name:       "multiline input",
			args:       []string{"out.txt"},
			input:      "aaa\nbbb\nccc\n",
			wantStdout: "aaa\nbbb\nccc\n",
			wantExit:   0,
			checkFiles: map[string]string{"out.txt": "aaa\nbbb\nccc\n"},
		},
		// 異常系
		{
			name:     "cannot create file in nonexistent directory",
			args:     []string{"/nonexistent/dir/file.txt"},
			input:    "data\n",
			wantExit: 1,
		},
		{
			name:     "unknown flag",
			args:     []string{"--invalid"},
			input:    "",
			wantExit: 2,
		},
		// エッジケース
		{
			name:       "empty input",
			args:       []string{"out.txt"},
			input:      "",
			wantStdout: "",
			wantExit:   0,
			checkFiles: map[string]string{"out.txt": ""},
		},
		{
			name:       "multibyte input",
			args:       []string{"out.txt"},
			input:      "こんにちは世界\n日本語テスト\n",
			wantStdout: "こんにちは世界\n日本語テスト\n",
			wantExit:   0,
			checkFiles: map[string]string{"out.txt": "こんにちは世界\n日本語テスト\n"},
		},
		{
			name:       "large input",
			args:       []string{"out.txt"},
			input:      strings.Repeat("x", 100000) + "\n",
			wantStdout: strings.Repeat("x", 100000) + "\n",
			wantExit:   0,
			checkFiles: map[string]string{"out.txt": strings.Repeat("x", 100000) + "\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Resolve file args to tmpDir
			resolvedArgs := make([]string, len(tt.args))
			for i, a := range tt.args {
				if a == "--invalid" || strings.HasPrefix(a, "-") || strings.HasPrefix(a, "/") {
					resolvedArgs[i] = a
				} else {
					resolvedArgs[i] = filepath.Join(tmpDir, a)
				}
			}

			stdin := strings.NewReader(tt.input)
			var stdout, stderr bytes.Buffer

			exitCode := run(resolvedArgs, stdin, &stdout, &stderr)

			if exitCode != tt.wantExit {
				t.Errorf("exit code = %d, want %d (stderr: %s)", exitCode, tt.wantExit, stderr.String())
			}

			if tt.wantExit == 0 && stdout.String() != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantStdout)
			}

			for name, want := range tt.checkFiles {
				path := filepath.Join(tmpDir, name)
				data, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("failed to read %s: %v", name, err)
					continue
				}
				if string(data) != want {
					t.Errorf("file %s = %q, want %q", name, string(data), want)
				}
			}
		})
	}
}

func TestVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"--version"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout.String(), "0.1.0") {
		t.Errorf("version output = %q, want containing 0.1.0", stdout.String())
	}
}

func TestMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")

	input := "hello\nworld\n"
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{f1, f2}, strings.NewReader(input), &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", exitCode, stderr.String())
	}
	if stdout.String() != input {
		t.Errorf("stdout = %q, want %q", stdout.String(), input)
	}

	for _, path := range []string{f1, f2} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		if string(data) != input {
			t.Errorf("file %s = %q, want %q", path, string(data), input)
		}
	}
}
