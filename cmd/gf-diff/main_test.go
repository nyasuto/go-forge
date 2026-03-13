package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Unit tests for myersDiff ---

func TestMyersDiff(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []string
		wantOps  []editOp
		wantDiff bool
	}{
		{
			name:     "identical files",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			wantOps:  []editOp{opEqual, opEqual, opEqual},
			wantDiff: false,
		},
		{
			name:     "insert one line",
			a:        []string{"a", "c"},
			b:        []string{"a", "b", "c"},
			wantOps:  []editOp{opEqual, opInsert, opEqual},
			wantDiff: true,
		},
		{
			name:     "delete one line",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "c"},
			wantOps:  []editOp{opEqual, opDelete, opEqual},
			wantDiff: true,
		},
		{
			name:     "replace one line",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "x", "c"},
			wantOps:  []editOp{opEqual, opDelete, opInsert, opEqual},
			wantDiff: true,
		},
		{
			name:     "both empty",
			a:        nil,
			b:        nil,
			wantOps:  nil,
			wantDiff: false,
		},
		{
			name:     "first empty",
			a:        nil,
			b:        []string{"a", "b"},
			wantOps:  []editOp{opInsert, opInsert},
			wantDiff: true,
		},
		{
			name:     "second empty",
			a:        []string{"a", "b"},
			b:        nil,
			wantOps:  []editOp{opDelete, opDelete},
			wantDiff: true,
		},
		{
			name:     "completely different",
			a:        []string{"a", "b"},
			b:        []string{"x", "y"},
			wantOps:  []editOp{opDelete, opDelete, opInsert, opInsert},
			wantDiff: true,
		},
		{
			name:     "multibyte lines",
			a:        []string{"こんにちは", "世界"},
			b:        []string{"こんにちは", "日本"},
			wantOps:  []editOp{opEqual, opDelete, opInsert},
			wantDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edits := myersDiff(tt.a, tt.b)

			if tt.wantOps == nil {
				if len(edits) != 0 {
					t.Errorf("expected no edits, got %d", len(edits))
				}
				return
			}

			if len(edits) != len(tt.wantOps) {
				t.Fatalf("expected %d edits, got %d: %v", len(tt.wantOps), len(edits), edits)
			}

			for i, e := range edits {
				if e.op != tt.wantOps[i] {
					t.Errorf("edit[%d]: expected op %d, got %d", i, tt.wantOps[i], e.op)
				}
			}

			if got := hasDifferences(edits); got != tt.wantDiff {
				t.Errorf("hasDifferences: expected %v, got %v", tt.wantDiff, got)
			}
		})
	}
}

// --- Unit tests for readLinesFromString ---

func TestReadLinesFromString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"single line", "hello", []string{"hello"}},
		{"multiple lines", "a\nb\nc", []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readLinesFromString(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d lines, got %d", len(tt.want), len(got))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line[%d]: expected %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}

// --- Integration tests ---

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		file1      string
		file2      string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "identical files returns 0",
			file1:      "a\nb\nc\n",
			file2:      "a\nb\nc\n",
			wantCode:   0,
			wantStdout: "",
		},
		{
			name:     "different files returns 1 with diff output",
			file1:    "a\nb\nc\n",
			file2:    "a\nx\nc\n",
			wantCode: 1,
			wantStdout: strings.Join([]string{
				"  a",
				"< b",
				"> x",
				"  c",
				"",
			}, "\n"),
		},
		{
			name:     "insert lines",
			file1:    "a\nc\n",
			file2:    "a\nb\nc\n",
			wantCode: 1,
			wantStdout: strings.Join([]string{
				"  a",
				"> b",
				"  c",
				"",
			}, "\n"),
		},
		{
			name:     "delete lines",
			file1:    "a\nb\nc\n",
			file2:    "a\nc\n",
			wantCode: 1,
			wantStdout: strings.Join([]string{
				"  a",
				"< b",
				"  c",
				"",
			}, "\n"),
		},
		{
			name:     "empty first file",
			file1:    "",
			file2:    "a\nb\n",
			wantCode: 1,
			wantStdout: strings.Join([]string{
				"> a",
				"> b",
				"",
			}, "\n"),
		},
		{
			name:       "empty second file",
			file1:      "a\nb\n",
			file2:      "",
			wantCode:   1,
			wantStdout: "< a\n< b\n",
		},
		{
			name:     "multibyte content",
			file1:    "こんにちは\n世界\n",
			file2:    "こんにちは\n日本\n",
			wantCode: 1,
			wantStdout: strings.Join([]string{
				"  こんにちは",
				"< 世界",
				"> 日本",
				"",
			}, "\n"),
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f1 := writeTestFile(t, tmpDir, fmt.Sprintf("a%d.txt", i), tt.file1)
			f2 := writeTestFile(t, tmpDir, fmt.Sprintf("b%d.txt", i), tt.file2)

			var stdout, stderr bytes.Buffer
			code := run([]string{f1, f2}, &stdout, &stderr)

			if code != tt.wantCode {
				t.Errorf("exit code: expected %d, got %d (stderr: %s)", tt.wantCode, code, stderr.String())
			}
			if tt.wantStdout != "" && stdout.String() != tt.wantStdout {
				t.Errorf("stdout:\nexpected:\n%s\ngot:\n%s", tt.wantStdout, stdout.String())
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), version) {
		t.Errorf("expected version in output, got %q", stdout.String())
	}
}

func TestRunUsageError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", nil},
		{"one arg", []string{"file1"}},
		{"three args", []string{"file1", "file2", "file3"}},
		{"unknown flag", []string{"--unknown"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)
			if code != 2 {
				t.Errorf("expected exit code 2, got %d", code)
			}
		})
	}
}

func TestRunFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	existing := writeTestFile(t, tmpDir, "exists.txt", "hello\n")

	t.Run("first file missing", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"/nonexistent", existing}, &stdout, &stderr)
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "nonexistent") {
			t.Errorf("expected file name in stderr, got %q", stderr.String())
		}
	})

	t.Run("second file missing", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{existing, "/nonexistent"}, &stdout, &stderr)
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	})
}

func TestRunLargeInput(t *testing.T) {
	tmpDir := t.TempDir()

	var b1, b2 strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&b1, "line %d\n", i)
		fmt.Fprintf(&b2, "line %d\n", i)
	}
	// Change one line in the middle
	lines2 := strings.Replace(b2.String(), "line 500\n", "modified 500\n", 1)

	f1 := writeTestFile(t, tmpDir, "large1.txt", b1.String())
	f2 := writeTestFile(t, tmpDir, "large2.txt", lines2)

	var stdout, stderr bytes.Buffer
	code := run([]string{f1, f2}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), "< line 500") {
		t.Error("expected deleted line in output")
	}
	if !strings.Contains(stdout.String(), "> modified 500") {
		t.Error("expected inserted line in output")
	}
}
