package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseExpression(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		wantPat   string
		wantRepl  string
		wantErr   bool
	}{
		{
			name:     "basic substitution",
			expr:     "s/foo/bar/",
			wantPat:  "foo",
			wantRepl: "bar",
		},
		{
			name:     "without trailing delimiter",
			expr:     "s/foo/bar",
			wantPat:  "foo",
			wantRepl: "bar",
		},
		{
			name:     "regex pattern",
			expr:     "s/[0-9]+/NUM/",
			wantPat:  "[0-9]+",
			wantRepl: "NUM",
		},
		{
			name:     "empty replacement",
			expr:     "s/foo//",
			wantPat:  "foo",
			wantRepl: "",
		},
		{
			name:     "custom delimiter",
			expr:     "s|foo|bar|",
			wantPat:  "foo",
			wantRepl: "bar",
		},
		{
			name:     "escaped delimiter in pattern",
			expr:     `s/foo\/bar/baz/`,
			wantPat:  "foo/bar",
			wantRepl: "baz",
		},
		{
			name:     "replacement with backslash",
			expr:     `s/foo/bar\\baz/`,
			wantPat:  "foo",
			wantRepl: `bar\\baz`,
		},
		{
			name:    "unknown command",
			expr:    "d/foo/bar/",
			wantErr: true,
		},
		{
			name:    "invalid regex",
			expr:    "s/[invalid/bar/",
			wantErr: true,
		},
		{
			name:    "no pattern or replacement",
			expr:    "s/",
			wantErr: true,
		},
		{
			name:    "empty expression after s",
			expr:    "s",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := parseExpression(tt.expr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if cmd.pattern.String() != tt.wantPat {
				t.Errorf("pattern = %q, want %q", cmd.pattern.String(), tt.wantPat)
			}
			if cmd.replace != tt.wantRepl {
				t.Errorf("replace = %q, want %q", cmd.replace, tt.wantRepl)
			}
		})
	}
}

func TestApplySubstitution(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		expr    string
		want    string
	}{
		{
			name: "basic replacement",
			line: "hello world",
			expr: "s/world/earth/",
			want: "hello earth",
		},
		{
			name: "first match only",
			line: "foo foo foo",
			expr: "s/foo/bar/",
			want: "bar foo foo",
		},
		{
			name: "no match",
			line: "hello world",
			expr: "s/xyz/abc/",
			want: "hello world",
		},
		{
			name: "regex replacement",
			line: "abc 123 def",
			expr: "s/[0-9]+/NUM/",
			want: "abc NUM def",
		},
		{
			name: "delete pattern",
			line: "hello world",
			expr: "s/world//",
			want: "hello ",
		},
		{
			name: "replace at beginning",
			line: "hello world",
			expr: "s/^hello/goodbye/",
			want: "goodbye world",
		},
		{
			name: "replace at end",
			line: "hello world",
			expr: "s/world$/earth/",
			want: "hello earth",
		},
		{
			name: "capture group",
			line: "John Smith",
			expr: "s/([A-Z]\\w+) ([A-Z]\\w+)/$2, $1/",
			want: "Smith, John",
		},
		{
			name: "multibyte characters",
			line: "こんにちは世界",
			expr: "s/世界/地球/",
			want: "こんにちは地球",
		},
		{
			name: "empty line",
			line: "",
			expr: "s/foo/bar/",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := parseExpression(tt.expr)
			if err != nil {
				t.Fatalf("failed to parse expression: %v", err)
			}
			got := applySubstitution(tt.line, cmd)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitByDelim(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		delim byte
		want  []string
	}{
		{
			name:  "basic split",
			s:     "foo/bar/",
			delim: '/',
			want:  []string{"foo", "bar", ""},
		},
		{
			name:  "escaped delimiter",
			s:     `foo\/bar/baz/`,
			delim: '/',
			want:  []string{"foo/bar", "baz", ""},
		},
		{
			name:  "no trailing delimiter",
			s:     "foo/bar",
			delim: '/',
			want:  []string{"foo", "bar"},
		},
		{
			name:  "pipe delimiter",
			s:     "foo|bar|",
			delim: '|',
			want:  []string{"foo", "bar", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitByDelim(tt.s, tt.delim)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v (len %d), want %v (len %d)", got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("part[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRun(t *testing.T) {
	// Create temp dir for test files
	tmpDir := t.TempDir()

	writeFile := func(name, content string) string {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	tests := []struct {
		name     string
		expr     string
		input    string   // stdin content
		files    []string // file paths
		wantOut  string
		wantErr  string
		wantCode int
	}{
		// Normal cases
		{
			name:     "stdin basic substitution",
			expr:     "s/foo/bar/",
			input:    "foo baz\nfoo qux\n",
			wantOut:  "bar baz\nbar qux\n",
			wantCode: 0,
		},
		{
			name:     "file input",
			expr:     "s/hello/goodbye/",
			files:    []string{writeFile("hello.txt", "hello world\nhello again\n")},
			wantOut:  "goodbye world\ngoodbye again\n",
			wantCode: 0,
		},
		{
			name:     "multiple files",
			expr:     "s/old/new/",
			files:    []string{writeFile("a.txt", "old value\n"), writeFile("b.txt", "old stuff\n")},
			wantOut:  "new value\nnew stuff\n",
			wantCode: 0,
		},
		{
			name:     "stdin with hyphen",
			expr:     "s/a/b/",
			input:    "abc\n",
			files:    []string{"-"},
			wantOut:  "bbc\n",
			wantCode: 0,
		},
		{
			name:     "no match passes through",
			expr:     "s/xyz/abc/",
			input:    "hello world\n",
			wantOut:  "hello world\n",
			wantCode: 0,
		},
		// Edge cases
		{
			name:     "empty input",
			expr:     "s/foo/bar/",
			input:    "",
			wantOut:  "",
			wantCode: 0,
		},
		{
			name:     "multibyte replacement",
			expr:     "s/hello/こんにちは/",
			input:    "hello world\n",
			wantOut:  "こんにちは world\n",
			wantCode: 0,
		},
		{
			name:     "first match only per line",
			expr:     "s/a/X/",
			input:    "abracadabra\n",
			wantOut:  "Xbracadabra\n",
			wantCode: 0,
		},
		// Error cases
		{
			name:     "nonexistent file",
			expr:     "s/foo/bar/",
			files:    []string{filepath.Join(tmpDir, "nonexistent.txt")},
			wantErr:  "no such file or directory",
			wantCode: 1,
		},
		{
			name:     "mixed existing and nonexistent files",
			expr:     "s/old/new/",
			files:    []string{writeFile("exists.txt", "old line\n"), filepath.Join(tmpDir, "nope.txt")},
			wantOut:  "new line\n",
			wantErr:  "no such file or directory",
			wantCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := parseExpression(tt.expr)
			if err != nil {
				t.Fatalf("failed to parse expression: %v", err)
			}

			stdin := strings.NewReader(tt.input)
			var stdout, stderr bytes.Buffer

			code := run(tt.files, stdin, &stdout, &stderr, cmd)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, tt.wantCode, stderr.String())
			}
			if stdout.String() != tt.wantOut {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantOut)
			}
			if tt.wantErr != "" && !strings.Contains(stderr.String(), tt.wantErr) {
				t.Errorf("stderr = %q, want containing %q", stderr.String(), tt.wantErr)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	if version != "0.1.0" {
		t.Errorf("version = %q, want %q", version, "0.1.0")
	}
}
