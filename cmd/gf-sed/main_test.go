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

// ===== Tier 2 Tests =====

func TestParseExpressionTier2(t *testing.T) {
	tests := []struct {
		name       string
		expr       string
		wantPat    string
		wantRepl   string
		wantGlobal bool
		wantAddr   addressType
		wantErr    bool
	}{
		{
			name:       "g flag",
			expr:       "s/foo/bar/g",
			wantPat:    "foo",
			wantRepl:   "bar",
			wantGlobal: true,
		},
		{
			name:       "no g flag",
			expr:       "s/foo/bar/",
			wantPat:    "foo",
			wantRepl:   "bar",
			wantGlobal: false,
		},
		{
			name:    "unknown flag",
			expr:    "s/foo/bar/x",
			wantErr: true,
		},
		{
			name:       "line number address",
			expr:       "3s/foo/bar/",
			wantPat:    "foo",
			wantRepl:   "bar",
			wantAddr:   addrLine,
			wantGlobal: false,
		},
		{
			name:       "last line address",
			expr:       "$s/foo/bar/",
			wantPat:    "foo",
			wantRepl:   "bar",
			wantAddr:   addrLast,
			wantGlobal: false,
		},
		{
			name:       "pattern address",
			expr:       "/error/s/foo/bar/",
			wantPat:    "foo",
			wantRepl:   "bar",
			wantAddr:   addrPattern,
			wantGlobal: false,
		},
		{
			name:       "line address with g flag",
			expr:       "2s/a/b/g",
			wantPat:    "a",
			wantRepl:   "b",
			wantAddr:   addrLine,
			wantGlobal: true,
		},
		{
			name:    "unterminated address regex",
			expr:    "/errors/foo/bar/",
			wantErr: true,
		},
		{
			name:    "invalid address regex",
			expr:    "/[bad/s/foo/bar/",
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
			if cmd.global != tt.wantGlobal {
				t.Errorf("global = %v, want %v", cmd.global, tt.wantGlobal)
			}
			if tt.wantAddr != addrNone {
				if cmd.addr == nil {
					t.Errorf("addr is nil, want type %d", tt.wantAddr)
				} else if cmd.addr.typ != tt.wantAddr {
					t.Errorf("addr.typ = %d, want %d", cmd.addr.typ, tt.wantAddr)
				}
			}
		})
	}
}

func TestGlobalFlag(t *testing.T) {
	tests := []struct {
		name string
		expr string
		line string
		want string
	}{
		{
			name: "g replaces all occurrences",
			expr: "s/foo/bar/g",
			line: "foo foo foo",
			want: "bar bar bar",
		},
		{
			name: "g with regex replaces all",
			expr: "s/[0-9]+/NUM/g",
			line: "a1 b2 c3",
			want: "aNUM bNUM cNUM",
		},
		{
			name: "g with no match",
			expr: "s/xyz/abc/g",
			line: "hello world",
			want: "hello world",
		},
		{
			name: "g with overlapping-like matches",
			expr: "s/a/X/g",
			line: "abracadabra",
			want: "XbrXcXdXbrX",
		},
		{
			name: "g with empty match pattern",
			expr: "s/o/0/g",
			line: "foo boo",
			want: "f00 b00",
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

func TestAddressRun(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		input   string
		wantOut string
	}{
		{
			name:    "line number address",
			expr:    "2s/foo/bar/",
			input:   "foo\nfoo\nfoo\n",
			wantOut: "foo\nbar\nfoo\n",
		},
		{
			name:    "line 1 address",
			expr:    "1s/hello/goodbye/",
			input:   "hello\nhello\nhello\n",
			wantOut: "goodbye\nhello\nhello\n",
		},
		{
			name:    "last line address",
			expr:    "$s/end/END/",
			input:   "start\nmiddle\nend\n",
			wantOut: "start\nmiddle\nEND\n",
		},
		{
			name:    "last line address single line",
			expr:    "$s/only/ONLY/",
			input:   "only\n",
			wantOut: "ONLY\n",
		},
		{
			name:    "pattern address",
			expr:    "/error/s/old/new/",
			input:   "error: old msg\ninfo: old msg\nerror: old log\n",
			wantOut: "error: new msg\ninfo: old msg\nerror: new log\n",
		},
		{
			name:    "pattern address with g flag",
			expr:    "/warn/s/x/Y/g",
			input:   "warn: x x x\ninfo: x x x\n",
			wantOut: "warn: Y Y Y\ninfo: x x x\n",
		},
		{
			name:    "line address with g flag",
			expr:    "1s/a/X/g",
			input:   "aaa\naaa\n",
			wantOut: "XXX\naaa\n",
		},
		{
			name:    "address beyond line count",
			expr:    "99s/foo/bar/",
			input:   "foo\nfoo\n",
			wantOut: "foo\nfoo\n",
		},
		{
			name:    "pattern address no match lines",
			expr:    "/DEBUG/s/x/y/",
			input:   "INFO: x\nWARN: x\n",
			wantOut: "INFO: x\nWARN: x\n",
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
			code := run(nil, stdin, &stdout, &stderr, cmd)
			if code != 0 {
				t.Errorf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
			}
			if stdout.String() != tt.wantOut {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantOut)
			}
		})
	}
}

func TestInPlace(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile := func(name, content string) string {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	readFile := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	t.Run("basic in-place edit", func(t *testing.T) {
		path := writeFile("inplace1.txt", "hello world\nhello again\n")
		cmd, _ := parseExpression("s/hello/goodbye/")
		var stderr bytes.Buffer
		code := runInPlace([]string{path}, &stderr, cmd)
		if code != 0 {
			t.Errorf("exit code = %d, want 0 (stderr: %s)", code, stderr.String())
		}
		got := readFile(path)
		want := "goodbye world\ngoodbye again\n"
		if got != want {
			t.Errorf("file content = %q, want %q", got, want)
		}
	})

	t.Run("in-place with g flag", func(t *testing.T) {
		path := writeFile("inplace2.txt", "aaa bbb aaa\nccc aaa ddd\n")
		cmd, _ := parseExpression("s/aaa/XXX/g")
		var stderr bytes.Buffer
		code := runInPlace([]string{path}, &stderr, cmd)
		if code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
		got := readFile(path)
		want := "XXX bbb XXX\nccc XXX ddd\n"
		if got != want {
			t.Errorf("file content = %q, want %q", got, want)
		}
	})

	t.Run("in-place multiple files", func(t *testing.T) {
		p1 := writeFile("ip_a.txt", "old value\n")
		p2 := writeFile("ip_b.txt", "old stuff\n")
		cmd, _ := parseExpression("s/old/new/")
		var stderr bytes.Buffer
		code := runInPlace([]string{p1, p2}, &stderr, cmd)
		if code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
		if got := readFile(p1); got != "new value\n" {
			t.Errorf("file a = %q, want %q", got, "new value\n")
		}
		if got := readFile(p2); got != "new stuff\n" {
			t.Errorf("file b = %q, want %q", got, "new stuff\n")
		}
	})

	t.Run("in-place preserves file permissions", func(t *testing.T) {
		path := writeFile("ip_perm.txt", "foo\n")
		os.Chmod(path, 0755)
		cmd, _ := parseExpression("s/foo/bar/")
		var stderr bytes.Buffer
		runInPlace([]string{path}, &stderr, cmd)
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0755 {
			t.Errorf("permissions = %o, want 0755", info.Mode().Perm())
		}
	})

	t.Run("in-place nonexistent file", func(t *testing.T) {
		cmd, _ := parseExpression("s/foo/bar/")
		var stderr bytes.Buffer
		code := runInPlace([]string{filepath.Join(tmpDir, "nope.txt")}, &stderr, cmd)
		if code != 1 {
			t.Errorf("exit code = %d, want 1", code)
		}
		if !strings.Contains(stderr.String(), "no such file") {
			t.Errorf("stderr = %q, want containing 'no such file'", stderr.String())
		}
	})

	t.Run("in-place stdin rejected", func(t *testing.T) {
		cmd, _ := parseExpression("s/foo/bar/")
		var stderr bytes.Buffer
		code := runInPlace([]string{"-"}, &stderr, cmd)
		if code != 1 {
			t.Errorf("exit code = %d, want 1", code)
		}
		if !strings.Contains(stderr.String(), "stdin") {
			t.Errorf("stderr = %q, want containing 'stdin'", stderr.String())
		}
	})

	t.Run("in-place with address", func(t *testing.T) {
		path := writeFile("ip_addr.txt", "line1\nline2\nline3\n")
		cmd, _ := parseExpression("2s/line/LINE/")
		var stderr bytes.Buffer
		code := runInPlace([]string{path}, &stderr, cmd)
		if code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
		got := readFile(path)
		want := "line1\nLINE2\nline3\n"
		if got != want {
			t.Errorf("file content = %q, want %q", got, want)
		}
	})
}

func TestGlobalFlagRun(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		input   string
		wantOut string
	}{
		{
			name:    "global replace multiple lines",
			expr:    "s/a/X/g",
			input:   "aaa\nbaba\n",
			wantOut: "XXX\nbXbX\n",
		},
		{
			name:    "global with capture groups",
			expr:    `s/(\w+)/[$1]/g`,
			input:   "hello world\n",
			wantOut: "[hello] [world]\n",
		},
		{
			name:    "global with multibyte",
			expr:    "s/あ/い/g",
			input:   "ああああ\n",
			wantOut: "いいいい\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := parseExpression(tt.expr)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			stdin := strings.NewReader(tt.input)
			var stdout, stderr bytes.Buffer
			code := run(nil, stdin, &stdout, &stderr, cmd)
			if code != 0 {
				t.Errorf("exit code = %d, want 0", code)
			}
			if stdout.String() != tt.wantOut {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantOut)
			}
		})
	}
}
