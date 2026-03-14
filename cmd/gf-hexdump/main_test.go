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
			formatLine(&buf, tt.offset, tt.data)
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
			code := hexdump(bytes.NewReader(tt.input), &stdout, &stderr)
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

func TestHexdumpExactly16Bytes(t *testing.T) {
	// Verify that exactly 16 bytes produces one line with correct format
	input := []byte("0123456789abcdef")
	var stdout, stderr bytes.Buffer
	code := hexdump(bytes.NewReader(input), &stdout, &stderr)
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
	code := hexdump(bytes.NewReader(input), &stdout, &stderr)
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
