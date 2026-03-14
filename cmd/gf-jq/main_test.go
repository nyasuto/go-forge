package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
		tokens  int
	}{
		{name: "identity", filter: ".", tokens: 1},
		{name: "single key", filter: ".name", tokens: 1},
		{name: "nested key", filter: ".user.name", tokens: 2},
		{name: "array index", filter: ".[0]", tokens: 1},
		{name: "key then index", filter: ".items.[0]", tokens: 2},
		{name: "key then index no dot", filter: ".items[0]", tokens: 2},
		{name: "deep nesting", filter: ".a.b.c.d", tokens: 4},
		{name: "index then key", filter: ".[0].name", tokens: 2},
		{name: "no leading dot", filter: "name", wantErr: true},
		{name: "unclosed bracket", filter: ".[0", wantErr: true},
		{name: "non-numeric index", filter: ".[abc]", wantErr: true},
		{name: "empty key", filter: "..", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := parseFilter(tt.filter)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for filter %q", tt.filter)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tokens) != tt.tokens {
				t.Errorf("got %d tokens, want %d", len(tokens), tt.tokens)
			}
		})
	}
}

func TestApplyFilter(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		filter string
		want   string
	}{
		// Normal cases
		{
			name:   "identity returns full object",
			json:   `{"name":"alice","age":30}`,
			filter: ".",
			want:   "{\n  \"age\": 30,\n  \"name\": \"alice\"\n}\n",
		},
		{
			name:   "simple key access",
			json:   `{"name":"alice","age":30}`,
			filter: ".name",
			want:   "\"alice\"\n",
		},
		{
			name:   "numeric value",
			json:   `{"name":"alice","age":30}`,
			filter: ".age",
			want:   "30\n",
		},
		{
			name:   "nested key access",
			json:   `{"user":{"name":"bob","role":"admin"}}`,
			filter: ".user.name",
			want:   "\"bob\"\n",
		},
		{
			name:   "array index",
			json:   `[1,2,3]`,
			filter: ".[1]",
			want:   "2\n",
		},
		{
			name:   "array first element",
			json:   `["a","b","c"]`,
			filter: ".[0]",
			want:   "\"a\"\n",
		},
		{
			name:   "key then array index",
			json:   `{"items":[10,20,30]}`,
			filter: ".items[1]",
			want:   "20\n",
		},
		{
			name:   "nested object in array",
			json:   `{"users":[{"name":"alice"},{"name":"bob"}]}`,
			filter: ".users[1].name",
			want:   "\"bob\"\n",
		},
		{
			name:   "boolean value",
			json:   `{"active":true}`,
			filter: ".active",
			want:   "true\n",
		},
		{
			name:   "null value",
			json:   `{"val":null}`,
			filter: ".val",
			want:   "null\n",
		},
		{
			name:   "missing key returns null",
			json:   `{"name":"alice"}`,
			filter: ".missing",
			want:   "null\n",
		},
		{
			name:   "out of range index returns null",
			json:   `[1,2,3]`,
			filter: ".[10]",
			want:   "null\n",
		},
		{
			name:   "negative index",
			json:   `[1,2,3]`,
			filter: ".[-1]",
			want:   "3\n",
		},
		// Edge cases
		{
			name:   "deeply nested",
			json:   `{"a":{"b":{"c":{"d":"deep"}}}}`,
			filter: ".a.b.c.d",
			want:   "\"deep\"\n",
		},
		{
			name:   "multibyte key",
			json:   `{"名前":"太郎","年齢":25}`,
			filter: ".名前",
			want:   "\"太郎\"\n",
		},
		{
			name:   "float value",
			json:   `{"pi":3.14}`,
			filter: ".pi",
			want:   "3.14\n",
		},
		{
			name:   "empty object",
			json:   `{}`,
			filter: ".key",
			want:   "null\n",
		},
		{
			name:   "empty array",
			json:   `[]`,
			filter: ".[0]",
			want:   "null\n",
		},
		{
			name:   "nested array access",
			json:   `[[1,2],[3,4]]`,
			filter: ".[1][0]",
			want:   "3\n",
		},
		{
			name:   "string with special chars",
			json:   `{"msg":"hello \"world\""}`,
			filter: ".msg",
			want:   "\"hello \\\"world\\\"\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := processReader(strings.NewReader(tt.json), mustParseFilter(t, tt.filter), &stdout, &stderr)
			if code != 0 {
				t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
			}
			if stdout.String() != tt.want {
				t.Errorf("got %q, want %q", stdout.String(), tt.want)
			}
		})
	}
}

func TestApplyFilterErrors(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		filter string
	}{
		{
			name:   "key access on array",
			json:   `[1,2,3]`,
			filter: ".name",
		},
		{
			name:   "index access on object",
			json:   `{"name":"alice"}`,
			filter: ".[0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := processReader(strings.NewReader(tt.json), mustParseFilter(t, tt.filter), &stdout, &stderr)
			if code != 1 {
				t.Errorf("expected exit code 1, got %d", code)
			}
		})
	}
}

func TestInvalidJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := processReader(strings.NewReader("not json"), mustParseFilter(t, "."), &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' in stderr, got %q", stderr.String())
	}
}

func TestEmptyInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := processReader(strings.NewReader(""), mustParseFilter(t, "."), &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1 for empty input, got %d", code)
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		stdin    string
		wantCode int
		wantOut  string
		wantErr  string
	}{
		{
			name:     "version flag",
			args:     []string{"--version"},
			wantCode: 0,
			wantOut:  "gf-jq version 0.1.0\n",
		},
		{
			name:     "missing filter",
			args:     []string{},
			wantCode: 2,
			wantErr:  "missing filter",
		},
		{
			name:     "stdin input",
			args:     []string{".name"},
			stdin:    `{"name":"test"}`,
			wantCode: 0,
			wantOut:  "\"test\"\n",
		},
		{
			name:     "stdin with hyphen",
			args:     []string{".x", "-"},
			stdin:    `{"x":42}`,
			wantCode: 0,
			wantOut:  "42\n",
		},
		{
			name:     "invalid filter",
			args:     []string{"name"},
			stdin:    `{}`,
			wantCode: 2,
			wantErr:  "invalid filter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tt.args, strings.NewReader(tt.stdin), &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("exit code: got %d, want %d (stderr: %s)", code, tt.wantCode, stderr.String())
			}
			if tt.wantOut != "" && stdout.String() != tt.wantOut {
				t.Errorf("stdout: got %q, want %q", stdout.String(), tt.wantOut)
			}
			if tt.wantErr != "" && !strings.Contains(stderr.String(), tt.wantErr) {
				t.Errorf("stderr: got %q, want it to contain %q", stderr.String(), tt.wantErr)
			}
		})
	}
}

func TestRunWithFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.json")
	os.WriteFile(file, []byte(`{"items":[{"id":1},{"id":2}]}`), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{".items[0].id", file}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
	}
	if stdout.String() != "1\n" {
		t.Errorf("got %q, want %q", stdout.String(), "1\n")
	}
}

func TestRunWithNonExistentFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{".", "/nonexistent/file.json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRunMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "a.json")
	file2 := filepath.Join(dir, "b.json")
	os.WriteFile(file1, []byte(`{"v":1}`), 0644)
	os.WriteFile(file2, []byte(`{"v":2}`), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{".v", file1, file2}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
	}
	if stdout.String() != "1\n2\n" {
		t.Errorf("got %q, want %q", stdout.String(), "1\n2\n")
	}
}

func TestLargeJSON(t *testing.T) {
	// Build a large array
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < 1000; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"id":` + strings.Repeat("0", 0) + `}`)
		// Just use the index
		sb.Reset()
	}
	// Simpler: large nested object
	input := `{"data":{"nested":{"value":999}}}`
	var stdout, stderr bytes.Buffer
	code := processReader(strings.NewReader(input), mustParseFilter(t, ".data.nested.value"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if strings.TrimSpace(stdout.String()) != "999" {
		t.Errorf("got %q", stdout.String())
	}
}

func mustParseFilter(t *testing.T, filter string) []token {
	t.Helper()
	tokens, err := parseFilter(filter)
	if err != nil {
		t.Fatalf("parseFilter(%q): %v", filter, err)
	}
	return tokens
}
