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
		stages  int
		tokens  int // tokens in first stage
	}{
		{name: "identity", filter: ".", stages: 1, tokens: 1},
		{name: "single key", filter: ".name", stages: 1, tokens: 1},
		{name: "nested key", filter: ".user.name", stages: 1, tokens: 2},
		{name: "array index", filter: ".[0]", stages: 1, tokens: 1},
		{name: "key then index", filter: ".items.[0]", stages: 1, tokens: 2},
		{name: "key then index no dot", filter: ".items[0]", stages: 1, tokens: 2},
		{name: "deep nesting", filter: ".a.b.c.d", stages: 1, tokens: 4},
		{name: "index then key", filter: ".[0].name", stages: 1, tokens: 2},
		// Tier 2: iterator
		{name: "iterator", filter: ".[]", stages: 1, tokens: 1},
		{name: "key then iterator", filter: ".items[]", stages: 1, tokens: 2},
		{name: "iterator then key", filter: ".[].name", stages: 1, tokens: 2},
		// Tier 2: pipe
		{name: "pipe two stages", filter: ".a | .b", stages: 2, tokens: 1},
		{name: "pipe three stages", filter: ".a | .b | .c", stages: 3, tokens: 1},
		{name: "pipe with iterator", filter: ".[] | .name", stages: 2, tokens: 1},
		{name: "pipe with identity", filter: ". | .name", stages: 2, tokens: 1},
		// Tier 2: length function
		{name: "length function", filter: "length", stages: 1, tokens: 1},
		{name: "pipe to length", filter: ". | length", stages: 2, tokens: 1},
		{name: "key pipe length", filter: ".items | length", stages: 2, tokens: 1},
		// Errors
		{name: "no leading dot", filter: "name", wantErr: true},
		{name: "unclosed bracket", filter: ".[0", wantErr: true},
		{name: "non-numeric index", filter: ".[abc]", wantErr: true},
		{name: "empty key", filter: "..", wantErr: true},
		{name: "empty pipe stage leading", filter: "| .name", wantErr: true},
		{name: "empty pipe stage trailing", filter: ".name |", wantErr: true},
		{name: "empty pipe stage middle", filter: ".a | | .b", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stages, err := parseFilter(tt.filter)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for filter %q", tt.filter)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(stages) != tt.stages {
				t.Errorf("got %d stages, want %d", len(stages), tt.stages)
			}
			if len(stages[0]) != tt.tokens {
				t.Errorf("first stage: got %d tokens, want %d", len(stages[0]), tt.tokens)
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
		// Normal cases (Tier 1)
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

func TestIterator(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		filter string
		want   string
	}{
		{
			name:   "iterate array",
			json:   `[1,2,3]`,
			filter: ".[]",
			want:   "1\n2\n3\n",
		},
		{
			name:   "iterate string array",
			json:   `["a","b","c"]`,
			filter: ".[]",
			want:   "\"a\"\n\"b\"\n\"c\"\n",
		},
		{
			name:   "iterate object values sorted by key",
			json:   `{"b":2,"a":1}`,
			filter: ".[]",
			want:   "1\n2\n",
		},
		{
			name:   "iterate nested array via key",
			json:   `{"items":[10,20,30]}`,
			filter: ".items[]",
			want:   "10\n20\n30\n",
		},
		{
			name:   "iterate then access key",
			json:   `[{"name":"alice"},{"name":"bob"}]`,
			filter: ".[].name",
			want:   "\"alice\"\n\"bob\"\n",
		},
		{
			name:   "iterate with pipe",
			json:   `[{"id":1},{"id":2}]`,
			filter: ".[] | .id",
			want:   "1\n2\n",
		},
		{
			name:   "key pipe iterate pipe key",
			json:   `{"users":[{"name":"alice"},{"name":"bob"}]}`,
			filter: ".users | .[] | .name",
			want:   "\"alice\"\n\"bob\"\n",
		},
		{
			name:   "empty array iteration",
			json:   `[]`,
			filter: ".[]",
			want:   "",
		},
		{
			name:   "empty object iteration",
			json:   `{}`,
			filter: ".[]",
			want:   "",
		},
		{
			name:   "iterate array of arrays",
			json:   `[[1,2],[3,4]]`,
			filter: ".[]",
			want:   "[\n  1,\n  2\n]\n[\n  3,\n  4\n]\n",
		},
		{
			name:   "nested iterate",
			json:   `[[1,2],[3]]`,
			filter: ".[] | .[]",
			want:   "1\n2\n3\n",
		},
		{
			name:   "iterate multibyte values",
			json:   `["あ","い","う"]`,
			filter: ".[]",
			want:   "\"あ\"\n\"い\"\n\"う\"\n",
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

func TestIteratorErrors(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		filter string
	}{
		{
			name:   "iterate over string",
			json:   `"hello"`,
			filter: ".[]",
		},
		{
			name:   "iterate over number",
			json:   `42`,
			filter: ".[]",
		},
		{
			name:   "iterate over boolean",
			json:   `true`,
			filter: ".[]",
		},
		{
			name:   "iterate over null",
			json:   `null`,
			filter: ".[]",
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

func TestLength(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		filter string
		want   string
	}{
		{
			name:   "length of array",
			json:   `[1,2,3]`,
			filter: "length",
			want:   "3\n",
		},
		{
			name:   "length of empty array",
			json:   `[]`,
			filter: "length",
			want:   "0\n",
		},
		{
			name:   "length of object",
			json:   `{"a":1,"b":2,"c":3}`,
			filter: "length",
			want:   "3\n",
		},
		{
			name:   "length of empty object",
			json:   `{}`,
			filter: "length",
			want:   "0\n",
		},
		{
			name:   "length of string",
			json:   `"hello"`,
			filter: "length",
			want:   "5\n",
		},
		{
			name:   "length of multibyte string",
			json:   `"こんにちは"`,
			filter: "length",
			want:   "5\n",
		},
		{
			name:   "length of null",
			json:   `null`,
			filter: "length",
			want:   "0\n",
		},
		{
			name:   "length of number (absolute value)",
			json:   `-42`,
			filter: "length",
			want:   "42\n",
		},
		{
			name:   "length of positive number",
			json:   `3.14`,
			filter: "length",
			want:   "3.14\n",
		},
		{
			name:   "length via pipe",
			json:   `{"items":[1,2,3,4,5]}`,
			filter: ".items | length",
			want:   "5\n",
		},
		{
			name:   "length of each element",
			json:   `["ab","cde","f"]`,
			filter: ".[] | length",
			want:   "2\n3\n1\n",
		},
		{
			name:   "length of nested array via key",
			json:   `{"data":{"list":[1,2,3]}}`,
			filter: ".data.list | length",
			want:   "3\n",
		},
		{
			name:   "length of string with emoji",
			json:   `"hello🌍"`,
			filter: "length",
			want:   "6\n",
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

func TestLengthErrors(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		filter string
	}{
		{
			name:   "length of boolean",
			json:   `true`,
			filter: "length",
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

func TestPipe(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		filter string
		want   string
	}{
		{
			name:   "identity pipe key",
			json:   `{"name":"alice"}`,
			filter: ". | .name",
			want:   "\"alice\"\n",
		},
		{
			name:   "key pipe key",
			json:   `{"user":{"name":"bob"}}`,
			filter: ".user | .name",
			want:   "\"bob\"\n",
		},
		{
			name:   "three stage pipe",
			json:   `{"a":{"b":{"c":"deep"}}}`,
			filter: ".a | .b | .c",
			want:   "\"deep\"\n",
		},
		{
			name:   "pipe fan out then collect",
			json:   `[{"x":1},{"x":2},{"x":3}]`,
			filter: ".[] | .x",
			want:   "1\n2\n3\n",
		},
		{
			name:   "pipe with length after iterate",
			json:   `[{"items":[1,2]},{"items":[3,4,5]}]`,
			filter: ".[] | .items | length",
			want:   "2\n3\n",
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
		{
			name:     "pipe via stdin",
			args:     []string{".items | .[]"},
			stdin:    `{"items":[1,2,3]}`,
			wantCode: 0,
			wantOut:  "1\n2\n3\n",
		},
		{
			name:     "length via stdin",
			args:     []string{"length"},
			stdin:    `[1,2,3,4]`,
			wantCode: 0,
			wantOut:  "4\n",
		},
		{
			name:     "empty pipe stage error",
			args:     []string{".a | | .b"},
			stdin:    `{}`,
			wantCode: 2,
			wantErr:  "empty pipeline stage",
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

func TestUnknownFunction(t *testing.T) {
	// "unknown" is not a valid function, but it starts without "." so parseFilter should reject it
	_, err := parseFilter("unknown")
	if err == nil {
		t.Fatal("expected error for unknown bare word")
	}
}

func TestKeys(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		filter string
		want   string
	}{
		{
			name:   "keys of object",
			json:   `{"b":2,"a":1,"c":3}`,
			filter: "keys",
			want:   "[\n  \"a\",\n  \"b\",\n  \"c\"\n]\n",
		},
		{
			name:   "keys of empty object",
			json:   `{}`,
			filter: "keys",
			want:   "[]\n",
		},
		{
			name:   "keys of array",
			json:   `["x","y","z"]`,
			filter: "keys",
			want:   "[\n  0,\n  1,\n  2\n]\n",
		},
		{
			name:   "keys of empty array",
			json:   `[]`,
			filter: "keys",
			want:   "[]\n",
		},
		{
			name:   "keys via pipe",
			json:   `{"data":{"x":1,"y":2}}`,
			filter: ".data | keys",
			want:   "[\n  \"x\",\n  \"y\"\n]\n",
		},
		{
			name:   "keys with multibyte keys",
			json:   `{"名前":"太郎","年齢":25}`,
			filter: "keys",
			want:   "[\n  \"名前\",\n  \"年齢\"\n]\n",
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

func TestKeysErrors(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		filter string
	}{
		{name: "keys of null", json: `null`, filter: "keys"},
		{name: "keys of string", json: `"hello"`, filter: "keys"},
		{name: "keys of number", json: `42`, filter: "keys"},
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

func TestValues(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		filter string
		want   string
	}{
		{
			name:   "values of object sorted by key",
			json:   `{"b":2,"a":1,"c":3}`,
			filter: "values",
			want:   "[\n  1,\n  2,\n  3\n]\n",
		},
		{
			name:   "values of empty object",
			json:   `{}`,
			filter: "values",
			want:   "[]\n",
		},
		{
			name:   "values of array",
			json:   `[10,20,30]`,
			filter: "values",
			want:   "[\n  10,\n  20,\n  30\n]\n",
		},
		{
			name:   "values via pipe",
			json:   `{"data":{"x":1,"y":2}}`,
			filter: ".data | values",
			want:   "[\n  1,\n  2\n]\n",
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

func TestValuesErrors(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		filter string
	}{
		{name: "values of null", json: `null`, filter: "values"},
		{name: "values of string", json: `"hello"`, filter: "values"},
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

func TestSelect(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		filter string
		want   string
	}{
		{
			name:   "select equal number",
			json:   `[{"name":"alice","age":30},{"name":"bob","age":25}]`,
			filter: `.[] | select(.age == 30)`,
			want:   "{\n  \"age\": 30,\n  \"name\": \"alice\"\n}\n",
		},
		{
			name:   "select greater than",
			json:   `[{"v":1},{"v":5},{"v":3}]`,
			filter: `.[] | select(.v > 2)`,
			want:   "{\n  \"v\": 5\n}\n{\n  \"v\": 3\n}\n",
		},
		{
			name:   "select less than",
			json:   `[{"v":1},{"v":5},{"v":3}]`,
			filter: `.[] | select(.v < 4)`,
			want:   "{\n  \"v\": 1\n}\n{\n  \"v\": 3\n}\n",
		},
		{
			name:   "select greater equal",
			json:   `[{"v":1},{"v":2},{"v":3}]`,
			filter: `.[] | select(.v >= 2)`,
			want:   "{\n  \"v\": 2\n}\n{\n  \"v\": 3\n}\n",
		},
		{
			name:   "select less equal",
			json:   `[{"v":1},{"v":2},{"v":3}]`,
			filter: `.[] | select(.v <= 2)`,
			want:   "{\n  \"v\": 1\n}\n{\n  \"v\": 2\n}\n",
		},
		{
			name:   "select not equal",
			json:   `[{"v":1},{"v":2},{"v":3}]`,
			filter: `.[] | select(.v != 2)`,
			want:   "{\n  \"v\": 1\n}\n{\n  \"v\": 3\n}\n",
		},
		{
			name:   "select string equal",
			json:   `[{"name":"alice"},{"name":"bob"},{"name":"alice"}]`,
			filter: `.[] | select(.name == "alice")`,
			want:   "{\n  \"name\": \"alice\"\n}\n{\n  \"name\": \"alice\"\n}\n",
		},
		{
			name:   "select truthiness true",
			json:   `[{"active":true,"name":"a"},{"active":false,"name":"b"},{"active":true,"name":"c"}]`,
			filter: `.[] | select(.active)`,
			want:   "{\n  \"active\": true,\n  \"name\": \"a\"\n}\n{\n  \"active\": true,\n  \"name\": \"c\"\n}\n",
		},
		{
			name:   "select truthiness null excluded",
			json:   `[{"v":1},{"v":null},{"v":3}]`,
			filter: `.[] | select(.v)`,
			want:   "{\n  \"v\": 1\n}\n{\n  \"v\": 3\n}\n",
		},
		{
			name:   "select none match",
			json:   `[{"v":1},{"v":2}]`,
			filter: `.[] | select(.v > 10)`,
			want:   "",
		},
		{
			name:   "select all match",
			json:   `[{"v":5},{"v":10}]`,
			filter: `.[] | select(.v > 0)`,
			want:   "{\n  \"v\": 5\n}\n{\n  \"v\": 10\n}\n",
		},
		{
			name:   "select with nested key",
			json:   `[{"user":{"role":"admin"}},{"user":{"role":"guest"}}]`,
			filter: `.[] | select(.user.role == "admin")`,
			want:   "{\n  \"user\": {\n    \"role\": \"admin\"\n  }\n}\n",
		},
		{
			name:   "select null comparison",
			json:   `[{"v":null},{"v":1}]`,
			filter: `.[] | select(.v == null)`,
			want:   "{\n  \"v\": null\n}\n",
		},
		{
			name:   "select then access key",
			json:   `[{"name":"alice","age":30},{"name":"bob","age":25}]`,
			filter: `.[] | select(.age > 28) | .name`,
			want:   "\"alice\"\n",
		},
		{
			name:   "select with multibyte string",
			json:   `[{"名前":"太郎"},{"名前":"花子"}]`,
			filter: `.[] | select(.名前 == "太郎")`,
			want:   "{\n  \"名前\": \"太郎\"\n}\n",
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

func TestSelectErrors(t *testing.T) {
	tests := []struct {
		name   string
		filter string
	}{
		{name: "empty select", filter: "select()"},
		{name: "invalid value in select", filter: `select(.x == abc)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseFilter(tt.filter)
			if err == nil {
				t.Errorf("expected error for filter %q", tt.filter)
			}
		})
	}
}

func TestParseFilterTier3(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		wantErr bool
		stages  int
	}{
		{name: "keys function", filter: "keys", stages: 1},
		{name: "values function", filter: "values", stages: 1},
		{name: "pipe to keys", filter: ". | keys", stages: 2},
		{name: "pipe to values", filter: ".data | values", stages: 2},
		{name: "select basic", filter: `select(.age > 30)`, stages: 1},
		{name: "select with pipe", filter: `.[] | select(.v == 1)`, stages: 2},
		{name: "empty select error", filter: "select()", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stages, err := parseFilter(tt.filter)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for filter %q", tt.filter)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(stages) != tt.stages {
				t.Errorf("got %d stages, want %d", len(stages), tt.stages)
			}
		})
	}
}

func mustParseFilter(t *testing.T, filter string) [][]token {
	t.Helper()
	stages, err := parseFilter(filter)
	if err != nil {
		t.Fatalf("parseFilter(%q): %v", filter, err)
	}
	return stages
}
