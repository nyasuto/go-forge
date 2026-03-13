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

func TestUnit_ExtractKey(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		keyField  int
		delimiter string
		want      string
	}{
		{"no key field", "hello world", 0, "", "hello world"},
		{"field 1", "banana apple cherry", 1, "", "banana"},
		{"field 2", "banana apple cherry", 2, "", "apple"},
		{"field 3", "banana apple cherry", 3, "", "cherry"},
		{"field out of range", "banana apple", 5, "", ""},
		{"multiple spaces", "  foo   bar  ", 1, "", "foo"},
		{"tabs", "foo\tbar\tbaz", 2, "", "bar"},
		{"comma delimiter field 1", "banana,apple,cherry", 1, ",", "banana"},
		{"comma delimiter field 2", "banana,apple,cherry", 2, ",", "apple"},
		{"colon delimiter", "root:0:admin", 2, ":", "0"},
		{"tab delimiter", "foo\tbar\tbaz", 2, "\t", "bar"},
		{"delimiter field out of range", "a,b", 5, ",", ""},
		{"delimiter with empty fields", "a,,c", 2, ",", ""},
		{"pipe delimiter", "x|y|z", 3, "|", "z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractKey(tt.line, tt.keyField, tt.delimiter)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnit_ParseNumber(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want float64
	}{
		{"integer", "42", 42},
		{"negative", "-10", -10},
		{"float", "3.14", 3.14},
		{"non-numeric", "abc", 0},
		{"empty", "", 0},
		{"leading spaces", "  100  ", 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNumber(tt.s)
			if got != tt.want {
				t.Errorf("got %f, want %f", got, tt.want)
			}
		})
	}
}

func TestUnit_Dedup(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"no duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"consecutive duplicates", []string{"a", "a", "b", "b", "c"}, []string{"a", "b", "c"}},
		{"all same", []string{"x", "x", "x"}, []string{"x"}},
		{"empty", []string{}, []string{}},
		{"single", []string{"a"}, []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedup(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d lines, want %d: %v", len(got), len(tt.want), got)
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

func TestIntegration_Tier2Options(t *testing.T) {
	bin := buildBinary(t)

	tests := []struct {
		name     string
		stdin    string
		args     []string
		wantOut  string
		wantCode int
	}{
		{
			name:    "numeric sort -n",
			stdin:   "10\n2\n1\n20\n3\n",
			args:    []string{"-n"},
			wantOut: "1\n2\n3\n10\n20\n",
		},
		{
			name:    "numeric sort with non-numeric lines",
			stdin:   "10\nabc\n2\n",
			args:    []string{"-n"},
			wantOut: "abc\n2\n10\n",
		},
		{
			name:    "numeric sort negative numbers",
			stdin:   "5\n-3\n0\n-10\n7\n",
			args:    []string{"-n"},
			wantOut: "-10\n-3\n0\n5\n7\n",
		},
		{
			name:    "numeric sort floats",
			stdin:   "3.14\n1.5\n2.71\n",
			args:    []string{"-n"},
			wantOut: "1.5\n2.71\n3.14\n",
		},
		{
			name:    "reverse sort -r",
			stdin:   "apple\ncherry\nbanana\n",
			args:    []string{"-r"},
			wantOut: "cherry\nbanana\napple\n",
		},
		{
			name:    "reverse numeric sort -n -r",
			stdin:   "10\n2\n1\n20\n3\n",
			args:    []string{"-n", "-r"},
			wantOut: "20\n10\n3\n2\n1\n",
		},
		{
			name:    "unique -u",
			stdin:   "b\na\nb\na\nc\n",
			args:    []string{"-u"},
			wantOut: "a\nb\nc\n",
		},
		{
			name:    "unique numeric -n -u",
			stdin:   "2\n1\n2\n3\n1\n",
			args:    []string{"-n", "-u"},
			wantOut: "1\n2\n3\n",
		},
		{
			name:    "key field -k 2",
			stdin:   "foo 3\nbar 1\nbaz 2\n",
			args:    []string{"-k", "2"},
			wantOut: "bar 1\nbaz 2\nfoo 3\n",
		},
		{
			name:    "key field numeric -k 2 -n",
			stdin:   "foo 10\nbar 2\nbaz 20\n",
			args:    []string{"-k", "2", "-n"},
			wantOut: "bar 2\nfoo 10\nbaz 20\n",
		},
		{
			name:    "key field with reverse -k 1 -r",
			stdin:   "cherry 1\napple 2\nbanana 3\n",
			args:    []string{"-k", "1", "-r"},
			wantOut: "cherry 1\nbanana 3\napple 2\n",
		},
		{
			name:    "key out of range treated as empty",
			stdin:   "a b\nc\nd e f\n",
			args:    []string{"-k", "3"},
			wantOut: "a b\nc\nd e f\n",
		},
		{
			name:    "unique with reverse -u -r",
			stdin:   "a\nb\na\nc\nb\n",
			args:    []string{"-u", "-r"},
			wantOut: "c\nb\na\n",
		},
		{
			name:    "all options combined -k 2 -n -r -u",
			stdin:   "x 5\ny 3\nz 5\nw 1\ny 3\n",
			args:    []string{"-k", "2", "-n", "-r", "-u"},
			wantOut: "z 5\nx 5\ny 3\nw 1\n",
		},
		{
			name:    "empty input with options",
			stdin:   "",
			args:    []string{"-n", "-r", "-u"},
			wantOut: "",
		},
		{
			name:    "multibyte with reverse",
			stdin:   "みかん\nりんご\nばなな\n",
			args:    []string{"-r"},
			wantOut: "りんご\nみかん\nばなな\n",
		},
		{
			name:    "delimiter -t comma with -k",
			stdin:   "cherry,3\napple,1\nbanana,2\n",
			args:    []string{"-t", ",", "-k", "2"},
			wantOut: "apple,1\nbanana,2\ncherry,3\n",
		},
		{
			name:    "delimiter -t colon with -k -n",
			stdin:   "user:100\nadmin:1\nguest:50\n",
			args:    []string{"-t", ":", "-k", "2", "-n"},
			wantOut: "admin:1\nguest:50\nuser:100\n",
		},
		{
			name:    "delimiter -t pipe with -k -r",
			stdin:   "a|x\nb|y\nc|z\n",
			args:    []string{"-t", "|", "-k", "2", "-r"},
			wantOut: "c|z\nb|y\na|x\n",
		},
		{
			name:    "delimiter -t tab with -k",
			stdin:   "cherry\t3\napple\t1\nbanana\t2\n",
			args:    []string{"-t", "\t", "-k", "1"},
			wantOut: "apple\t1\nbanana\t2\ncherry\t3\n",
		},
		{
			name:    "delimiter -t with -k -u",
			stdin:   "a,2\nb,1\na,2\nb,1\n",
			args:    []string{"-t", ",", "-k", "2", "-n", "-u"},
			wantOut: "b,1\na,2\n",
		},
		{
			name:    "delimiter -t with empty fields",
			stdin:   "x,,3\ny,,1\nz,,2\n",
			args:    []string{"-t", ",", "-k", "3", "-n"},
			wantOut: "y,,1\nz,,2\nx,,3\n",
		},
		{
			name:    "delimiter -t without -k sorts whole line",
			stdin:   "banana\napple\ncherry\n",
			args:    []string{"-t", ","},
			wantOut: "apple\nbanana\ncherry\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, code := runWithStdin(t, bin, tt.stdin, tt.args...)
			if stdout != tt.wantOut {
				t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout, tt.wantOut)
			}
			if code != tt.wantCode {
				t.Errorf("exit code: got %d, want %d", code, tt.wantCode)
			}
		})
	}
}
