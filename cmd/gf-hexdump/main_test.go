package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatLine(t *testing.T) {
	tests := []struct {
		name   string
		offset int
		data   []byte
		want   string
	}{
		{
			name:   "full 16 bytes",
			offset: 0,
			data:   []byte("Hello, World!..!"),
			want:   "00000000  48 65 6c 6c 6f 2c 20 57  6f 72 6c 64 21 2e 2e 21  |Hello, World!..!|\n",
		},
		{
			name:   "partial line",
			offset: 0x10,
			data:   []byte("Hi"),
			want:   "00000010  48 69                                             |Hi|\n",
		},
		{
			name:   "non-printable bytes",
			offset: 0,
			data:   []byte{0x00, 0x01, 0x7f, 0xff, 0x41},
			want:   "00000000  00 01 7f ff 41                                    |....A|\n",
		},
		{
			name:   "offset at boundary",
			offset: 0x100,
			data:   []byte{0x42},
			want:   "00000100  42                                                |B|\n",
		},
		{
			name:   "8 bytes boundary gap",
			offset: 0,
			data:   []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a},
			want:   "00000000  01 02 03 04 05 06 07 08  09 0a                    |..........|\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatLine(&buf, tt.offset, tt.data, false)
			got := buf.String()
			if got != tt.want {
				t.Errorf("formatLine() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestHexdump(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantCode int
		contains []string
	}{
		{
			name:     "simple ASCII",
			input:    []byte("Hello"),
			wantCode: 0,
			contains: []string{"00000000", "48 65 6c 6c 6f", "|Hello|"},
		},
		{
			name:     "empty input",
			input:    []byte{},
			wantCode: 0,
			contains: nil,
		},
		{
			name:     "exactly 16 bytes",
			input:    []byte("0123456789ABCDEF"),
			wantCode: 0,
			contains: []string{"00000000", "30 31 32 33", "|0123456789ABCDEF|"},
		},
		{
			name:     "more than 16 bytes",
			input:    []byte("This is a longer string that spans multiple lines!"),
			wantCode: 0,
			contains: []string{"00000000", "00000010", "00000020", "00000030"},
		},
		{
			name:     "binary data with nulls",
			input:    []byte{0x00, 0x00, 0x00, 0x00},
			wantCode: 0,
			contains: []string{"00 00 00 00", "|....|"},
		},
		{
			name:     "all byte values 0x20-0x2f",
			input:    []byte{0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f},
			wantCode: 0,
			contains: []string{`| !"#$%&'()*+,-./|`},
		},
		{
			name:     "multibyte UTF-8",
			input:    []byte("日本語"),
			wantCode: 0,
			contains: []string{"e6 97 a5 e6 9c ac e8 aa  9e", "|.........|"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := hexdump(bytes.NewReader(tt.input), &stdout, &stderr, hexdumpOptions{limit: -1})
			if code != tt.wantCode {
				t.Errorf("hexdump() code = %d, want %d", code, tt.wantCode)
			}
			for _, s := range tt.contains {
				if !strings.Contains(stdout.String(), s) {
					t.Errorf("output missing %q\ngot:\n%s", s, stdout.String())
				}
			}
		})
	}
}

func TestRun(t *testing.T) {
	// Create temp files
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "test1.bin")
	os.WriteFile(file1, []byte("ABCD"), 0644)

	file2 := filepath.Join(tmpDir, "test2.bin")
	os.WriteFile(file2, []byte{0xff, 0xfe}, 0644)

	tests := []struct {
		name     string
		args     []string
		stdin    string
		wantCode int
		contains []string
		errMsg   string
	}{
		{
			name:     "stdin no args",
			args:     []string{},
			stdin:    "test",
			wantCode: 0,
			contains: []string{"74 65 73 74", "|test|"},
		},
		{
			name:     "stdin with dash",
			args:     []string{"-"},
			stdin:    "test",
			wantCode: 0,
			contains: []string{"|test|"},
		},
		{
			name:     "file argument",
			args:     []string{file1},
			wantCode: 0,
			contains: []string{"41 42 43 44", "|ABCD|"},
		},
		{
			name:     "multiple files",
			args:     []string{file1, file2},
			wantCode: 0,
			contains: []string{"41 42 43 44", "ff fe"},
		},
		{
			name:     "nonexistent file",
			args:     []string{"/nonexistent/file"},
			wantCode: 1,
			errMsg:   "gf-hexdump:",
		},
		{
			name:     "version flag",
			args:     []string{"--version"},
			wantCode: 0,
			contains: []string{"gf-hexdump version 0.1.0"},
		},
		{
			name:     "unknown flag",
			args:     []string{"--unknown"},
			wantCode: 2,
		},
		{
			name:     "nonexistent file with valid file",
			args:     []string{file1, "/nonexistent", file2},
			wantCode: 1,
			contains: []string{"41 42 43 44", "ff fe"},
			errMsg:   "gf-hexdump:",
		},
		{
			name:     "large input crossing multiple lines",
			args:     []string{},
			stdin:    strings.Repeat("A", 48),
			wantCode: 0,
			contains: []string{"00000000", "00000010", "00000020"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			stdin := strings.NewReader(tt.stdin)
			code := run(tt.args, stdin, &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("run() code = %d, want %d\nstderr: %s", code, tt.wantCode, stderr.String())
			}
			for _, s := range tt.contains {
				if !strings.Contains(stdout.String(), s) {
					t.Errorf("stdout missing %q\ngot:\n%s", s, stdout.String())
				}
			}
			if tt.errMsg != "" {
				if !strings.Contains(stderr.String(), tt.errMsg) {
					t.Errorf("stderr missing %q\ngot: %s", tt.errMsg, stderr.String())
				}
			}
		})
	}
}

func TestHexdumpSkip(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		opts     hexdumpOptions
		wantCode int
		contains []string
		notContains []string
	}{
		{
			name:     "skip 4 bytes",
			input:    []byte("ABCDhello"),
			opts:     hexdumpOptions{skip: 4, limit: -1},
			wantCode: 0,
			contains: []string{"00000004", "68 65 6c 6c 6f", "|hello|"},
			notContains: []string{"41 42 43 44"},
		},
		{
			name:     "skip past end",
			input:    []byte("short"),
			opts:     hexdumpOptions{skip: 100, limit: -1},
			wantCode: 0,
		},
		{
			name:     "skip 0 is noop",
			input:    []byte("AB"),
			opts:     hexdumpOptions{skip: 0, limit: -1},
			wantCode: 0,
			contains: []string{"00000000", "41 42"},
		},
		{
			name:     "skip 16 to second line",
			input:    []byte("0123456789ABCDEFsecond"),
			opts:     hexdumpOptions{skip: 16, limit: -1},
			wantCode: 0,
			contains: []string{"00000010", "|second|"},
			notContains: []string{"00000000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := hexdump(bytes.NewReader(tt.input), &stdout, &stderr, tt.opts)
			if code != tt.wantCode {
				t.Errorf("hexdump() code = %d, want %d", code, tt.wantCode)
			}
			for _, s := range tt.contains {
				if !strings.Contains(stdout.String(), s) {
					t.Errorf("output missing %q\ngot:\n%s", s, stdout.String())
				}
			}
			for _, s := range tt.notContains {
				if strings.Contains(stdout.String(), s) {
					t.Errorf("output should not contain %q\ngot:\n%s", s, stdout.String())
				}
			}
		})
	}
}

func TestHexdumpLimit(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		opts     hexdumpOptions
		wantCode int
		contains []string
		notContains []string
	}{
		{
			name:     "limit 5 bytes",
			input:    []byte("Hello, World!"),
			opts:     hexdumpOptions{limit: 5},
			wantCode: 0,
			contains: []string{"48 65 6c 6c 6f", "|Hello|"},
			notContains: []string{"2c"},
		},
		{
			name:     "limit 0 bytes",
			input:    []byte("Hello"),
			opts:     hexdumpOptions{limit: 0},
			wantCode: 0,
		},
		{
			name:     "limit larger than input",
			input:    []byte("Hi"),
			opts:     hexdumpOptions{limit: 100},
			wantCode: 0,
			contains: []string{"48 69", "|Hi|"},
		},
		{
			name:     "limit exactly 16",
			input:    []byte("0123456789ABCDEFGHIJ"),
			opts:     hexdumpOptions{limit: 16},
			wantCode: 0,
			contains: []string{"|0123456789ABCDEF|"},
			notContains: []string{"47 48"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := hexdump(bytes.NewReader(tt.input), &stdout, &stderr, tt.opts)
			if code != tt.wantCode {
				t.Errorf("hexdump() code = %d, want %d", code, tt.wantCode)
			}
			for _, s := range tt.contains {
				if !strings.Contains(stdout.String(), s) {
					t.Errorf("output missing %q\ngot:\n%s", s, stdout.String())
				}
			}
			for _, s := range tt.notContains {
				if strings.Contains(stdout.String(), s) {
					t.Errorf("output should not contain %q\ngot:\n%s", s, stdout.String())
				}
			}
		})
	}
}

func TestHexdumpSkipAndLimit(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		opts     hexdumpOptions
		wantCode int
		contains []string
		notContains []string
	}{
		{
			name:     "skip 5 limit 5",
			input:    []byte("Hello, World!"),
			opts:     hexdumpOptions{skip: 5, limit: 5},
			wantCode: 0,
			contains: []string{"00000005", "2c 20 57 6f 72", "|, Wor|"},
			notContains: []string{"48 65 6c"},
		},
		{
			name:     "skip and limit to single byte",
			input:    []byte("ABCDE"),
			opts:     hexdumpOptions{skip: 2, limit: 1},
			wantCode: 0,
			contains: []string{"00000002", "43", "|C|"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := hexdump(bytes.NewReader(tt.input), &stdout, &stderr, tt.opts)
			if code != tt.wantCode {
				t.Errorf("hexdump() code = %d, want %d", code, tt.wantCode)
			}
			for _, s := range tt.contains {
				if !strings.Contains(stdout.String(), s) {
					t.Errorf("output missing %q\ngot:\n%s", s, stdout.String())
				}
			}
			for _, s := range tt.notContains {
				if strings.Contains(stdout.String(), s) {
					t.Errorf("output should not contain %q\ngot:\n%s", s, stdout.String())
				}
			}
		})
	}
}

func TestRunWithSkipAndLimit(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test.bin")
	os.WriteFile(file1, []byte("Hello, World! This is a test."), 0644)

	tests := []struct {
		name     string
		args     []string
		stdin    string
		wantCode int
		contains []string
		notContains []string
		errMsg   string
	}{
		{
			name:     "skip flag with file",
			args:     []string{"-s", "7", file1},
			wantCode: 0,
			contains: []string{"00000007", "57 6f 72 6c 64"},
		},
		{
			name:     "limit flag with file",
			args:     []string{"-n", "5", file1},
			wantCode: 0,
			contains: []string{"48 65 6c 6c 6f", "|Hello|"},
		},
		{
			name:     "skip and limit with file",
			args:     []string{"-s", "7", "-n", "5", file1},
			wantCode: 0,
			contains: []string{"00000007", "|World|"},
		},
		{
			name:     "skip with stdin",
			args:     []string{"-s", "3"},
			stdin:    "ABCDhello",
			wantCode: 0,
			contains: []string{"00000003", "|Dhello|"},
		},
		{
			name:     "limit with stdin",
			args:     []string{"-n", "3"},
			stdin:    "ABCDhello",
			wantCode: 0,
			contains: []string{"|ABC|"},
			notContains: []string{"44"},
		},
		{
			name:     "negative skip",
			args:     []string{"-s", "-1"},
			wantCode: 2,
			errMsg:   "invalid skip value",
		},
		{
			name:     "negative limit",
			args:     []string{"-n", "-2"},
			wantCode: 2,
			errMsg:   "invalid length value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			stdin := strings.NewReader(tt.stdin)
			code := run(tt.args, stdin, &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("run() code = %d, want %d\nstderr: %s\nstdout: %s", code, tt.wantCode, stderr.String(), stdout.String())
			}
			for _, s := range tt.contains {
				if !strings.Contains(stdout.String(), s) {
					t.Errorf("stdout missing %q\ngot:\n%s", s, stdout.String())
				}
			}
			for _, s := range tt.notContains {
				if strings.Contains(stdout.String(), s) {
					t.Errorf("stdout should not contain %q\ngot:\n%s", s, stdout.String())
				}
			}
			if tt.errMsg != "" {
				if !strings.Contains(stderr.String(), tt.errMsg) {
					t.Errorf("stderr missing %q\ngot: %s", tt.errMsg, stderr.String())
				}
			}
		})
	}
}

func TestHexdumpExactly16Bytes(t *testing.T) {
	// Verify that exactly 16 bytes produces one line with correct format
	input := []byte("0123456789abcdef")
	var stdout, stderr bytes.Buffer
	code := hexdump(bytes.NewReader(input), &stdout, &stderr, hexdumpOptions{limit: -1})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d:\n%s", len(lines), stdout.String())
	}
}

func TestHexdump17Bytes(t *testing.T) {
	// 17 bytes should produce exactly 2 lines
	input := bytes.Repeat([]byte{0x41}, 17)
	var stdout, stderr bytes.Buffer
	code := hexdump(bytes.NewReader(input), &stdout, &stderr, hexdumpOptions{limit: -1})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d:\n%s", len(lines), stdout.String())
	}
	// Second line should have offset 0x10
	if !strings.HasPrefix(lines[1], "00000010") {
		t.Errorf("second line should start with 00000010, got: %s", lines[1])
	}
}

func TestByteColor(t *testing.T) {
	tests := []struct {
		name string
		b    byte
		want string
	}{
		{"null byte", 0x00, "\033[2m"},
		{"printable A", 0x41, "\033[32m"},
		{"printable space", 0x20, "\033[32m"},
		{"printable tilde", 0x7e, "\033[32m"},
		{"control tab", 0x09, "\033[31m"},
		{"control newline", 0x0a, "\033[31m"},
		{"control 0x01", 0x01, "\033[31m"},
		{"control 0x1f", 0x1f, "\033[31m"},
		{"control DEL", 0x7f, "\033[31m"},
		{"high byte 0x80", 0x80, "\033[34m"},
		{"high byte 0xff", 0xff, "\033[34m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := byteColor(tt.b)
			if got != tt.want {
				t.Errorf("byteColor(0x%02x) = %q, want %q", tt.b, got, tt.want)
			}
		})
	}
}

func TestFormatLineColor(t *testing.T) {
	tests := []struct {
		name     string
		offset   int
		data     []byte
		contains []string
	}{
		{
			name:   "null bytes get dim color",
			offset: 0,
			data:   []byte{0x00},
			contains: []string{
				"\033[2m00\033[0m",  // hex colored
				"\033[2m.\033[0m",   // ASCII colored
			},
		},
		{
			name:   "printable bytes get green",
			offset: 0,
			data:   []byte("A"),
			contains: []string{
				"\033[32m41\033[0m",  // hex colored
				"\033[32mA\033[0m",   // ASCII colored
			},
		},
		{
			name:   "control bytes get red",
			offset: 0,
			data:   []byte{0x01},
			contains: []string{
				"\033[31m01\033[0m",  // hex colored
				"\033[31m.\033[0m",   // ASCII colored
			},
		},
		{
			name:   "high bytes get blue",
			offset: 0,
			data:   []byte{0xff},
			contains: []string{
				"\033[34mff\033[0m",  // hex colored
				"\033[34m.\033[0m",   // ASCII colored
			},
		},
		{
			name:   "mixed bytes",
			offset: 0,
			data:   []byte{0x00, 0x41, 0x01, 0x80},
			contains: []string{
				"\033[2m00\033[0m",   // null
				"\033[32m41\033[0m",  // printable
				"\033[31m01\033[0m",  // control
				"\033[34m80\033[0m",  // high
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatLine(&buf, tt.offset, tt.data, true)
			got := buf.String()
			for _, s := range tt.contains {
				if !strings.Contains(got, s) {
					t.Errorf("color output missing %q\ngot:\n%q", s, got)
				}
			}
		})
	}
}

func TestColorNoColor(t *testing.T) {
	// Verify color=false produces no escape sequences
	data := []byte{0x00, 0x41, 0x01, 0xff}
	var buf bytes.Buffer
	formatLine(&buf, 0, data, false)
	got := buf.String()
	if strings.Contains(got, "\033[") {
		t.Errorf("no-color output contains escape sequences:\n%q", got)
	}
}

func TestRunColorFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		stdin    string
		wantCode int
		hasColor bool
		errMsg   string
	}{
		{
			name:     "color always",
			args:     []string{"--color", "always"},
			stdin:    "AB",
			wantCode: 0,
			hasColor: true,
		},
		{
			name:     "color never",
			args:     []string{"--color", "never"},
			stdin:    "AB",
			wantCode: 0,
			hasColor: false,
		},
		{
			name:     "color auto with non-terminal",
			args:     []string{"--color", "auto"},
			stdin:    "AB",
			wantCode: 0,
			hasColor: false, // bytes.Buffer is not a terminal
		},
		{
			name:     "invalid color mode",
			args:     []string{"--color", "invalid"},
			stdin:    "",
			wantCode: 2,
			errMsg:   "invalid color mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			stdin := strings.NewReader(tt.stdin)
			code := run(tt.args, stdin, &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("run() code = %d, want %d\nstderr: %s", code, tt.wantCode, stderr.String())
			}
			if tt.wantCode == 0 {
				hasEsc := strings.Contains(stdout.String(), "\033[")
				if tt.hasColor && !hasEsc {
					t.Errorf("expected color output but got none:\n%q", stdout.String())
				}
				if !tt.hasColor && hasEsc {
					t.Errorf("expected no color but got escape sequences:\n%q", stdout.String())
				}
			}
			if tt.errMsg != "" && !strings.Contains(stderr.String(), tt.errMsg) {
				t.Errorf("stderr missing %q\ngot: %s", tt.errMsg, stderr.String())
			}
		})
	}
}
