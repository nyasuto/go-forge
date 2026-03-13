package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gf-sort")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func runWithStdin(t *testing.T, bin string, stdin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

func TestUnit_ReadLinesFrom(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "normal lines",
			input: "banana\napple\ncherry\n",
			want:  []string{"banana", "apple", "cherry"},
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "single line no newline",
			input: "hello",
			want:  []string{"hello"},
		},
		{
			name:  "multibyte",
			input: "みかん\nりんご\nばなな\n",
			want:  []string{"みかん", "りんご", "ばなな"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readLinesFrom(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d lines, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// Integration tests
func TestIntegration_BasicSort(t *testing.T) {
	bin := buildBinary(t)

	tests := []struct {
		name     string
		stdin    string
		args     []string
		wantOut  string
		wantErr  string
		wantCode int
	}{
		{
			name:    "sort from stdin",
			stdin:   "banana\napple\ncherry\n",
			wantOut: "apple\nbanana\ncherry\n",
		},
		{
			name:    "already sorted",
			stdin:   "a\nb\nc\n",
			wantOut: "a\nb\nc\n",
		},
		{
			name:    "reverse order input",
			stdin:   "c\nb\na\n",
			wantOut: "a\nb\nc\n",
		},
		{
			name:    "empty input",
			stdin:   "",
			wantOut: "",
		},
		{
			name:    "single line",
			stdin:   "only\n",
			wantOut: "only\n",
		},
		{
			name:    "multibyte sort",
			stdin:   "みかん\nりんご\nばなな\nいちご\n",
			wantOut: "いちご\nばなな\nみかん\nりんご\n",
		},
		{
			name:    "case sensitivity",
			stdin:   "Banana\napple\nCherry\n",
			wantOut: "Banana\nCherry\napple\n",
		},
		{
			name:    "duplicate lines",
			stdin:   "b\na\nb\na\n",
			wantOut: "a\na\nb\nb\n",
		},
		{
			name:    "version flag",
			stdin:   "",
			args:    []string{"--version"},
			wantOut: "gf-sort version 0.1.0\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, code := runWithStdin(t, bin, tt.stdin, tt.args...)
			if stdout != tt.wantOut {
				t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout, tt.wantOut)
			}
			if tt.wantErr != "" && !strings.Contains(stderr, tt.wantErr) {
				t.Errorf("stderr: got %q, want containing %q", stderr, tt.wantErr)
			}
			if code != tt.wantCode {
				t.Errorf("exit code: got %d, want %d", code, tt.wantCode)
			}
		})
	}
}

func TestIntegration_FileInput(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// Create test files
	f1 := filepath.Join(dir, "file1.txt")
	os.WriteFile(f1, []byte("cherry\napple\n"), 0644)
	f2 := filepath.Join(dir, "file2.txt")
	os.WriteFile(f2, []byte("banana\ndate\n"), 0644)

	tests := []struct {
		name     string
		args     []string
		wantOut  string
		wantErr  string
		wantCode int
	}{
		{
			name:    "single file",
			args:    []string{f1},
			wantOut: "apple\ncherry\n",
		},
		{
			name:    "multiple files merged and sorted",
			args:    []string{f1, f2},
			wantOut: "apple\nbanana\ncherry\ndate\n",
		},
		{
			name:     "nonexistent file",
			args:     []string{filepath.Join(dir, "nope.txt")},
			wantOut:  "",
			wantErr:  "no such file",
			wantCode: 1,
		},
		{
			name:    "stdin via hyphen",
			args:    []string{"-"},
			wantOut: "a\nb\nc\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin := ""
			if tt.name == "stdin via hyphen" {
				stdin = "c\na\nb\n"
			}
			stdout, stderr, code := runWithStdin(t, bin, stdin, tt.args...)
			if stdout != tt.wantOut {
				t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout, tt.wantOut)
			}
			if tt.wantErr != "" && !strings.Contains(stderr, tt.wantErr) {
				t.Errorf("stderr: got %q, want containing %q", stderr, tt.wantErr)
			}
			if code != tt.wantCode {
				t.Errorf("exit code: got %d, want %d", code, tt.wantCode)
			}
		})
	}
}
