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

// --- Unit tests for buildHunks ---

func TestBuildHunks(t *testing.T) {
	tests := []struct {
		name        string
		a, b        []string
		context     int
		wantHunks   int
		wantHeaders []string // "@@ -x,y +a,b @@"
	}{
		{
			name:        "single change with context",
			a:           []string{"a", "b", "c", "d", "e"},
			b:           []string{"a", "b", "x", "d", "e"},
			context:     3,
			wantHunks:   1,
			wantHeaders: []string{"@@ -1,5 +1,5 @@"},
		},
		{
			name:        "two distant changes become two hunks",
			a:           []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15"},
			b:           []string{"1", "X", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "Y"},
			context:     2,
			wantHunks:   2,
			wantHeaders: []string{"@@ -1,4 +1,4 @@", "@@ -13,3 +13,3 @@"},
		},
		{
			name:        "insert at beginning",
			a:           []string{"b", "c"},
			b:           []string{"a", "b", "c"},
			context:     3,
			wantHunks:   1,
			wantHeaders: []string{"@@ -1,2 +1,3 @@"},
		},
		{
			name:        "delete at end",
			a:           []string{"a", "b", "c"},
			b:           []string{"a", "b"},
			context:     3,
			wantHunks:   1,
			wantHeaders: []string{"@@ -1,3 +1,2 @@"},
		},
		{
			name:        "zero context",
			a:           []string{"a", "b", "c"},
			b:           []string{"a", "x", "c"},
			context:     0,
			wantHunks:   1,
			wantHeaders: []string{"@@ -2,1 +2,1 @@"},
		},
		{
			name:        "nearby changes merge into single hunk",
			a:           []string{"1", "2", "3", "4", "5", "6", "7"},
			b:           []string{"1", "X", "3", "4", "Y", "6", "7"},
			context:     2,
			wantHunks:   1,
			wantHeaders: []string{"@@ -1,7 +1,7 @@"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edits := myersDiff(tt.a, tt.b)
			hunks := buildHunks(edits, tt.context)
			if len(hunks) != tt.wantHunks {
				t.Fatalf("expected %d hunks, got %d", tt.wantHunks, len(hunks))
			}
			for i, h := range hunks {
				header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.oldStart, h.oldCount, h.newStart, h.newCount)
				if header != tt.wantHeaders[i] {
					t.Errorf("hunk[%d]: expected %q, got %q", i, tt.wantHeaders[i], header)
				}
			}
		})
	}
}

// --- Integration tests for unified format ---

func TestRunUnified(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		file1      string
		file2      string
		wantCode   int
		wantOutput []string // lines to check in output
	}{
		{
			name:     "basic unified diff",
			file1:    "a\nb\nc\n",
			file2:    "a\nx\nc\n",
			wantCode: 1,
			wantOutput: []string{
				"---",
				"+++",
				"@@ -1,3 +1,3 @@",
				" a",
				"-b",
				"+x",
				" c",
			},
		},
		{
			name:       "identical files no output",
			file1:      "a\nb\nc\n",
			file2:      "a\nb\nc\n",
			wantCode:   0,
			wantOutput: nil,
		},
		{
			name:     "insert lines unified",
			file1:    "a\nc\n",
			file2:    "a\nb\nc\n",
			wantCode: 1,
			wantOutput: []string{
				"@@ -1,2 +1,3 @@",
				" a",
				"+b",
				" c",
			},
		},
		{
			name:     "delete lines unified",
			file1:    "a\nb\nc\n",
			file2:    "a\nc\n",
			wantCode: 1,
			wantOutput: []string{
				"@@ -1,3 +1,2 @@",
				" a",
				"-b",
				" c",
			},
		},
		{
			name:     "multibyte unified",
			file1:    "こんにちは\n世界\n",
			file2:    "こんにちは\n日本\n",
			wantCode: 1,
			wantOutput: []string{
				"-世界",
				"+日本",
			},
		},
		{
			name:     "file headers contain filenames",
			file1:    "a\n",
			file2:    "b\n",
			wantCode: 1,
			wantOutput: []string{
				"--- ",
				"+++ ",
			},
		},
		{
			name:     "empty first file unified",
			file1:    "",
			file2:    "a\nb\n",
			wantCode: 1,
			wantOutput: []string{
				"@@ -1,0 +1,2 @@",
				"+a",
				"+b",
			},
		},
		{
			name:     "empty second file unified",
			file1:    "a\nb\n",
			file2:    "",
			wantCode: 1,
			wantOutput: []string{
				"@@ -1,2 +1,0 @@",
				"-a",
				"-b",
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f1 := writeTestFile(t, tmpDir, fmt.Sprintf("u_a%d.txt", i), tt.file1)
			f2 := writeTestFile(t, tmpDir, fmt.Sprintf("u_b%d.txt", i), tt.file2)

			var stdout, stderr bytes.Buffer
			code := run([]string{"-u", f1, f2}, &stdout, &stderr)

			if code != tt.wantCode {
				t.Errorf("exit code: expected %d, got %d (stderr: %s)", tt.wantCode, code, stderr.String())
			}
			output := stdout.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\ngot:\n%s", want, output)
				}
			}
		})
	}
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

// --- Unit tests for splitWords ---

func TestSplitWords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"single word", "hello", []string{"hello"}},
		{"two words", "hello world", []string{"hello", " ", "world"}},
		{"leading space", "  hello", []string{"  ", "hello"}},
		{"trailing space", "hello  ", []string{"hello", "  "}},
		{"multiple spaces", "a  b  c", []string{"a", "  ", "b", "  ", "c"}},
		{"tabs and spaces", "a\t b", []string{"a", "\t ", "b"}},
		{"multibyte words", "こんにちは 世界", []string{"こんにちは", " ", "世界"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitWords(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d tokens, got %d: %q", len(tt.want), len(got), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("token[%d]: expected %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}

// --- Unit tests for wordDiffLine ---

func TestWordDiffLine(t *testing.T) {
	tests := []struct {
		name    string
		oldLine string
		newLine string
		wantOld string
		wantNew string
	}{
		{
			name:    "single word change",
			oldLine: "hello world",
			newLine: "hello earth",
			wantOld: "hello [-world-]",
			wantNew: "hello [+earth+]",
		},
		{
			name:    "word insertion",
			oldLine: "a c",
			newLine: "a b c",
			wantOld: "a c",
			wantNew: "a [+b+][+ +]c",
		},
		{
			name:    "word deletion",
			oldLine: "a b c",
			newLine: "a c",
			wantOld: "a [-b-][- -]c",
			wantNew: "a c",
		},
		{
			name:    "completely different",
			oldLine: "foo bar",
			newLine: "baz qux",
			wantOld: "[-foo-] [-bar-]",
			wantNew: "[+baz+] [+qux+]",
		},
		{
			name:    "identical lines",
			oldLine: "same text",
			newLine: "same text",
			wantOld: "same text",
			wantNew: "same text",
		},
		{
			name:    "multibyte word change",
			oldLine: "こんにちは 世界",
			newLine: "こんにちは 日本",
			wantOld: "こんにちは [-世界-]",
			wantNew: "こんにちは [+日本+]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOld, gotNew := wordDiffLine(tt.oldLine, tt.newLine)
			if gotOld != tt.wantOld {
				t.Errorf("old: expected %q, got %q", tt.wantOld, gotOld)
			}
			if gotNew != tt.wantNew {
				t.Errorf("new: expected %q, got %q", tt.wantNew, gotNew)
			}
		})
	}
}

// --- Integration tests for color output ---

func TestRunColorAlways(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := writeTestFile(t, tmpDir, "color_a.txt", "a\nb\nc\n")
	f2 := writeTestFile(t, tmpDir, "color_b.txt", "a\nx\nc\n")

	t.Run("normal mode with color", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"--color=always", f1, f2}, &stdout, &stderr)
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		output := stdout.String()
		if !strings.Contains(output, colorRed) {
			t.Error("expected red color code in output")
		}
		if !strings.Contains(output, colorGreen) {
			t.Error("expected green color code in output")
		}
		if !strings.Contains(output, colorReset) {
			t.Error("expected reset code in output")
		}
	})

	t.Run("unified mode with color", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-u", "--color=always", f1, f2}, &stdout, &stderr)
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
		output := stdout.String()
		if !strings.Contains(output, colorBoldRed) {
			t.Error("expected bold red for --- header")
		}
		if !strings.Contains(output, colorBoldGrn) {
			t.Error("expected bold green for +++ header")
		}
		if !strings.Contains(output, colorCyan) {
			t.Error("expected cyan for @@ header")
		}
	})
}

func TestRunColorNever(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := writeTestFile(t, tmpDir, "nocolor_a.txt", "a\nb\nc\n")
	f2 := writeTestFile(t, tmpDir, "nocolor_b.txt", "a\nx\nc\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"--color=never", f1, f2}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	output := stdout.String()
	if strings.Contains(output, "\033[") {
		t.Error("expected no ANSI escape codes with --color=never")
	}
}

func TestRunColorAuto(t *testing.T) {
	// With bytes.Buffer (not a terminal), auto should produce no color
	tmpDir := t.TempDir()
	f1 := writeTestFile(t, tmpDir, "auto_a.txt", "a\nb\nc\n")
	f2 := writeTestFile(t, tmpDir, "auto_b.txt", "a\nx\nc\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"--color=auto", f1, f2}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	output := stdout.String()
	if strings.Contains(output, "\033[") {
		t.Error("auto mode to non-terminal should not produce color codes")
	}
}

func TestRunColorInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := writeTestFile(t, tmpDir, "inv_a.txt", "a\n")
	f2 := writeTestFile(t, tmpDir, "inv_b.txt", "b\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"--color=invalid", f1, f2}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("expected exit code 2 for invalid color value, got %d", code)
	}
}

// --- Integration tests for --word diff ---

func TestRunWordDiffNormal(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := writeTestFile(t, tmpDir, "word_a.txt", "hello world\nfoo bar\n")
	f2 := writeTestFile(t, tmpDir, "word_b.txt", "hello earth\nfoo bar\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"--word", f1, f2}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "[-world-]") {
		t.Errorf("expected [-world-] marker, got:\n%s", output)
	}
	if !strings.Contains(output, "[+earth+]") {
		t.Errorf("expected [+earth+] marker, got:\n%s", output)
	}
}

func TestRunWordDiffUnified(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := writeTestFile(t, tmpDir, "uword_a.txt", "hello world\nfoo bar\n")
	f2 := writeTestFile(t, tmpDir, "uword_b.txt", "hello earth\nfoo bar\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"-u", "--word", f1, f2}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "[-world-]") {
		t.Errorf("expected [-world-] marker in unified word diff, got:\n%s", output)
	}
	if !strings.Contains(output, "[+earth+]") {
		t.Errorf("expected [+earth+] marker in unified word diff, got:\n%s", output)
	}
}

func TestRunWordDiffWithColor(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := writeTestFile(t, tmpDir, "wc_a.txt", "hello world\n")
	f2 := writeTestFile(t, tmpDir, "wc_b.txt", "hello earth\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"--word", "--color=always", f1, f2}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, colorRed) {
		t.Error("expected color codes in word diff with --color=always")
	}
	if !strings.Contains(output, "[-world-]") {
		t.Errorf("expected word markers in colored output, got:\n%s", output)
	}
}

func TestRunWordDiffMultibyte(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := writeTestFile(t, tmpDir, "wmb_a.txt", "こんにちは 世界\n")
	f2 := writeTestFile(t, tmpDir, "wmb_b.txt", "こんにちは 日本\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"--word", f1, f2}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "[-世界-]") {
		t.Errorf("expected [-世界-] marker, got:\n%s", output)
	}
	if !strings.Contains(output, "[+日本+]") {
		t.Errorf("expected [+日本+] marker, got:\n%s", output)
	}
}

func TestRunWordDiffInsertOnly(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := writeTestFile(t, tmpDir, "wio_a.txt", "a c\n")
	f2 := writeTestFile(t, tmpDir, "wio_b.txt", "a b c\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"--word", f1, f2}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "[+b+]") {
		t.Errorf("expected word insertion markers, got:\n%s", output)
	}
}

func TestRunWordDiffDeleteOnly(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := writeTestFile(t, tmpDir, "wdo_a.txt", "a b c\n")
	f2 := writeTestFile(t, tmpDir, "wdo_b.txt", "a c\n")

	var stdout, stderr bytes.Buffer
	code := run([]string{"--word", f1, f2}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "[-b-]") {
		t.Errorf("expected word deletion markers, got:\n%s", output)
	}
}
