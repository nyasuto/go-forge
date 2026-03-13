package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFields(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		want    []fieldRange
		wantErr bool
	}{
		{name: "single field", spec: "2", want: []fieldRange{{2, 2}}},
		{name: "multiple fields", spec: "1,3", want: []fieldRange{{1, 1}, {3, 3}}},
		{name: "range", spec: "1-3", want: []fieldRange{{1, 3}}},
		{name: "open end range", spec: "2-", want: []fieldRange{{2, -1}}},
		{name: "open start range", spec: "-3", want: []fieldRange{{1, 3}}},
		{name: "mixed", spec: "1,3-5,7", want: []fieldRange{{1, 1}, {3, 5}, {7, 7}}},
		{name: "invalid field zero", spec: "0", wantErr: true},
		{name: "invalid field negative", spec: "-0", wantErr: true},
		{name: "invalid decreasing range", spec: "5-3", wantErr: true},
		{name: "invalid empty", spec: ",", wantErr: true},
		{name: "invalid non-numeric", spec: "abc", wantErr: true},
		{name: "invalid dash-dash", spec: "-", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFields(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseFields(%q) expected error, got nil", tt.spec)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFields(%q) unexpected error: %v", tt.spec, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseFields(%q) got %d ranges, want %d", tt.spec, len(got), len(tt.want))
			}
			for i, r := range got {
				if r != tt.want[i] {
					t.Errorf("parseFields(%q)[%d] = %+v, want %+v", tt.spec, i, r, tt.want[i])
				}
			}
		})
	}
}

func TestSelectFields(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		ranges []fieldRange
		want   []string
	}{
		{
			name:   "single field",
			fields: []string{"a", "b", "c"},
			ranges: []fieldRange{{2, 2}},
			want:   []string{"b"},
		},
		{
			name:   "multiple fields",
			fields: []string{"a", "b", "c", "d"},
			ranges: []fieldRange{{1, 1}, {3, 3}},
			want:   []string{"a", "c"},
		},
		{
			name:   "range",
			fields: []string{"a", "b", "c", "d"},
			ranges: []fieldRange{{2, 4}},
			want:   []string{"b", "c", "d"},
		},
		{
			name:   "open end range",
			fields: []string{"a", "b", "c", "d"},
			ranges: []fieldRange{{3, -1}},
			want:   []string{"c", "d"},
		},
		{
			name:   "field out of range",
			fields: []string{"a", "b"},
			ranges: []fieldRange{{5, 5}},
			want:   nil,
		},
		{
			name:   "partial out of range",
			fields: []string{"a", "b", "c"},
			ranges: []fieldRange{{2, 5}},
			want:   []string{"b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectFields(tt.fields, tt.ranges)
			if len(got) != len(tt.want) {
				t.Fatalf("selectFields() got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("selectFields()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     cutOptions
		ranges   []fieldRange
		wantOut  string
		wantCode int
	}{
		{
			name:    "tab delimiter default field 2",
			input:   "a\tb\tc\nd\te\tf\n",
			opts:    cutOptions{delimiter: "\t"},
			ranges:  []fieldRange{{2, 2}},
			wantOut: "b\ne\n",
		},
		{
			name:    "comma delimiter field 1 and 3",
			input:   "one,two,three\nfour,five,six\n",
			opts:    cutOptions{delimiter: ","},
			ranges:  []fieldRange{{1, 1}, {3, 3}},
			wantOut: "one,three\nfour,six\n",
		},
		{
			name:    "field range 2-3",
			input:   "a:b:c:d\ne:f:g:h\n",
			opts:    cutOptions{delimiter: ":"},
			ranges:  []fieldRange{{2, 3}},
			wantOut: "b:c\nf:g\n",
		},
		{
			name:    "open end range 3-",
			input:   "1,2,3,4,5\n",
			opts:    cutOptions{delimiter: ","},
			ranges:  []fieldRange{{3, -1}},
			wantOut: "3,4,5\n",
		},
		{
			name:    "field beyond columns outputs empty",
			input:   "a,b\n",
			opts:    cutOptions{delimiter: ","},
			ranges:  []fieldRange{{5, 5}},
			wantOut: "\n",
		},
		{
			name:    "empty input",
			input:   "",
			opts:    cutOptions{delimiter: "\t"},
			ranges:  []fieldRange{{1, 1}},
			wantOut: "",
		},
		{
			name:    "multibyte delimiter and content",
			input:   "東京\t大阪\t名古屋\n",
			opts:    cutOptions{delimiter: "\t"},
			ranges:  []fieldRange{{1, 1}, {3, 3}},
			wantOut: "東京\t名古屋\n",
		},
		{
			name:    "single column input",
			input:   "hello\nworld\n",
			opts:    cutOptions{delimiter: "\t"},
			ranges:  []fieldRange{{1, 1}},
			wantOut: "hello\nworld\n",
		},
		{
			name:    "no delimiter in line outputs whole line as field 1",
			input:   "nodelimiter\n",
			opts:    cutOptions{delimiter: ","},
			ranges:  []fieldRange{{1, 1}},
			wantOut: "nodelimiter\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			stdin := strings.NewReader(tt.input)
			code := run(nil, stdin, &stdout, &stderr, tt.opts, tt.ranges)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, tt.wantCode, stderr.String())
			}
			if stdout.String() != tt.wantOut {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantOut)
			}
		})
	}
}

func TestRunWithFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "test1.csv")
	os.WriteFile(file1, []byte("a,b,c\nd,e,f\n"), 0644)

	file2 := filepath.Join(tmpDir, "test2.csv")
	os.WriteFile(file2, []byte("x,y,z\n"), 0644)

	t.Run("single file", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		opts := cutOptions{delimiter: ","}
		ranges := []fieldRange{{2, 2}}
		code := run([]string{file1}, nil, &stdout, &stderr, opts, ranges)
		if code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
		want := "b\ne\n"
		if stdout.String() != want {
			t.Errorf("stdout = %q, want %q", stdout.String(), want)
		}
	})

	t.Run("multiple files", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		opts := cutOptions{delimiter: ","}
		ranges := []fieldRange{{1, 1}, {3, 3}}
		code := run([]string{file1, file2}, nil, &stdout, &stderr, opts, ranges)
		if code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
		want := "a,c\nd,f\nx,z\n"
		if stdout.String() != want {
			t.Errorf("stdout = %q, want %q", stdout.String(), want)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		opts := cutOptions{delimiter: ","}
		ranges := []fieldRange{{1, 1}}
		code := run([]string{filepath.Join(tmpDir, "no-such-file")}, nil, &stdout, &stderr, opts, ranges)
		if code != 1 {
			t.Errorf("exit code = %d, want 1", code)
		}
		if !strings.Contains(stderr.String(), "gf-cut:") {
			t.Errorf("stderr should contain error message, got %q", stderr.String())
		}
	})

	t.Run("stdin via hyphen", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader("p|q|r\n")
		opts := cutOptions{delimiter: "|"}
		ranges := []fieldRange{{2, 2}}
		code := run([]string{"-"}, stdin, &stdout, &stderr, opts, ranges)
		if code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
		want := "q\n"
		if stdout.String() != want {
			t.Errorf("stdout = %q, want %q", stdout.String(), want)
		}
	})

	t.Run("mixed file and stdin", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader("1,2,3\n")
		opts := cutOptions{delimiter: ","}
		ranges := []fieldRange{{2, 2}}
		code := run([]string{file1, "-"}, stdin, &stdout, &stderr, opts, ranges)
		if code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
		want := "b\ne\n2\n"
		if stdout.String() != want {
			t.Errorf("stdout = %q, want %q", stdout.String(), want)
		}
	})
}

func TestSelectChars(t *testing.T) {
	tests := []struct {
		name   string
		runes  []rune
		ranges []fieldRange
		want   string
	}{
		{
			name:   "single char",
			runes:  []rune("abcdef"),
			ranges: []fieldRange{{3, 3}},
			want:   "c",
		},
		{
			name:   "char range",
			runes:  []rune("abcdef"),
			ranges: []fieldRange{{2, 4}},
			want:   "bcd",
		},
		{
			name:   "open end range",
			runes:  []rune("abcdef"),
			ranges: []fieldRange{{4, -1}},
			want:   "def",
		},
		{
			name:   "multiple positions",
			runes:  []rune("abcdef"),
			ranges: []fieldRange{{1, 1}, {3, 3}, {5, 5}},
			want:   "ace",
		},
		{
			name:   "out of range",
			runes:  []rune("ab"),
			ranges: []fieldRange{{5, 5}},
			want:   "",
		},
		{
			name:   "multibyte rune positions",
			runes:  []rune("あいうえお"),
			ranges: []fieldRange{{2, 4}},
			want:   "いうえ",
		},
		{
			name:   "mixed ascii and multibyte",
			runes:  []rune("a漢b字c"),
			ranges: []fieldRange{{1, 3}},
			want:   "a漢b",
		},
		{
			name:   "emoji rune positions",
			runes:  []rune("🍎🍊🍇🍉"),
			ranges: []fieldRange{{2, 3}},
			want:   "🍊🍇",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectChars(tt.runes, tt.ranges)
			if got != tt.want {
				t.Errorf("selectChars() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunCharMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		ranges   []fieldRange
		wantOut  string
		wantCode int
	}{
		{
			name:    "basic char cut",
			input:   "abcdef\nghijkl\n",
			ranges:  []fieldRange{{1, 3}},
			wantOut: "abc\nghi\n",
		},
		{
			name:    "char cut single position",
			input:   "hello world\n",
			ranges:  []fieldRange{{7, 7}},
			wantOut: "w\n",
		},
		{
			name:    "char cut open end",
			input:   "abcdef\n",
			ranges:  []fieldRange{{4, -1}},
			wantOut: "def\n",
		},
		{
			name:    "char cut multibyte",
			input:   "東京都新宿区\n",
			ranges:  []fieldRange{{1, 3}},
			wantOut: "東京都\n",
		},
		{
			name:    "char cut empty input",
			input:   "",
			ranges:  []fieldRange{{1, 5}},
			wantOut: "",
		},
		{
			name:    "char cut beyond line length",
			input:   "ab\n",
			ranges:  []fieldRange{{1, 10}},
			wantOut: "ab\n",
		},
		{
			name:    "char cut multiple ranges",
			input:   "abcdefghij\n",
			ranges:  []fieldRange{{1, 2}, {5, 6}, {9, 10}},
			wantOut: "abefij\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			stdin := strings.NewReader(tt.input)
			opts := cutOptions{mode: modeChars}
			code := run(nil, stdin, &stdout, &stderr, opts, tt.ranges)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			if stdout.String() != tt.wantOut {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantOut)
			}
		})
	}
}

func TestSplitCsvFields(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		delim string
		want  []string
	}{
		{
			name:  "simple no quotes",
			line:  "a,b,c",
			delim: ",",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "quoted field with delimiter inside",
			line:  `"hello,world",b,c`,
			delim: ",",
			want:  []string{`"hello,world"`, "b", "c"},
		},
		{
			name:  "escaped quote inside quoted field",
			line:  `"say ""hi""",b`,
			delim: ",",
			want:  []string{`"say ""hi"""`, "b"},
		},
		{
			name:  "empty quoted field",
			line:  `"",b,c`,
			delim: ",",
			want:  []string{`""`, "b", "c"},
		},
		{
			name:  "mixed quoted and unquoted",
			line:  `a,"b,c",d`,
			delim: ",",
			want:  []string{"a", `"b,c"`, "d"},
		},
		{
			name:  "multibyte content in quotes",
			line:  `"東京,大阪",名古屋`,
			delim: ",",
			want:  []string{`"東京,大阪"`, "名古屋"},
		},
		{
			name:  "newline-like content in quotes",
			line:  `"field1","field with ""quotes""","field3"`,
			delim: ",",
			want:  []string{`"field1"`, `"field with ""quotes"""`, `"field3"`},
		},
		{
			name:  "single field",
			line:  `"only"`,
			delim: ",",
			want:  []string{`"only"`},
		},
		{
			name:  "tab delimiter with csv quotes",
			line:  "\"a\tb\"\tc",
			delim: "\t",
			want:  []string{"\"a\tb\"", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCsvFields(tt.line, tt.delim)
			if len(got) != len(tt.want) {
				t.Fatalf("splitCsvFields(%q) got %v (len %d), want %v (len %d)", tt.line, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitCsvFields(%q)[%d] = %q, want %q", tt.line, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRunCsvMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     cutOptions
		ranges   []fieldRange
		wantOut  string
		wantCode int
	}{
		{
			name:    "csv basic field extraction",
			input:   "a,b,c\nd,e,f\n",
			opts:    cutOptions{delimiter: ",", mode: modeCsv},
			ranges:  []fieldRange{{2, 2}},
			wantOut: "b\ne\n",
		},
		{
			name:    "csv quoted field with delimiter",
			input:   "\"hello,world\",b,c\n",
			opts:    cutOptions{delimiter: ",", mode: modeCsv},
			ranges:  []fieldRange{{1, 1}},
			wantOut: "\"hello,world\"\n",
		},
		{
			name:    "csv select field after quoted",
			input:   "\"a,b\",c,d\n",
			opts:    cutOptions{delimiter: ",", mode: modeCsv},
			ranges:  []fieldRange{{2, 2}},
			wantOut: "c\n",
		},
		{
			name:    "csv multiple quoted fields",
			input:   "\"x,y\",\"a,b\",z\n",
			opts:    cutOptions{delimiter: ",", mode: modeCsv},
			ranges:  []fieldRange{{1, 1}, {3, 3}},
			wantOut: "\"x,y\",z\n",
		},
		{
			name:    "csv escaped quotes",
			input:   "\"say \"\"hi\"\"\",b\n",
			opts:    cutOptions{delimiter: ",", mode: modeCsv},
			ranges:  []fieldRange{{1, 1}},
			wantOut: "\"say \"\"hi\"\"\"\n",
		},
		{
			name:    "csv empty input",
			input:   "",
			opts:    cutOptions{delimiter: ",", mode: modeCsv},
			ranges:  []fieldRange{{1, 1}},
			wantOut: "",
		},
		{
			name:    "csv multibyte in quotes",
			input:   "\"東京,大阪\",名古屋\n",
			opts:    cutOptions{delimiter: ",", mode: modeCsv},
			ranges:  []fieldRange{{1, 1}},
			wantOut: "\"東京,大阪\"\n",
		},
		{
			name:    "csv field range with quotes",
			input:   "a,\"b,c\",d,e\n",
			opts:    cutOptions{delimiter: ",", mode: modeCsv},
			ranges:  []fieldRange{{2, 3}},
			wantOut: "\"b,c\",d\n",
		},
		{
			name:    "csv without quotes behaves like normal",
			input:   "one,two,three\n",
			opts:    cutOptions{delimiter: ",", mode: modeCsv},
			ranges:  []fieldRange{{1, 1}, {3, 3}},
			wantOut: "one,three\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			stdin := strings.NewReader(tt.input)
			code := run(nil, stdin, &stdout, &stderr, tt.opts, tt.ranges)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, tt.wantCode, stderr.String())
			}
			if stdout.String() != tt.wantOut {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantOut)
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	// This is tested indirectly through the main function behavior,
	// but we can at least verify the version constant
	if version != "0.1.0" {
		t.Errorf("version = %q, want %q", version, "0.1.0")
	}
}
