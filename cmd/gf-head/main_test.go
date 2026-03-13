package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHead(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		n       int
		want    string
		wantErr bool
	}{
		// 正常系
		{
			name:  "default 10 lines from 15 line input",
			input: "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13\n14\n15\n",
			n:     10,
			want:  "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n",
		},
		{
			name:  "fewer lines than n",
			input: "a\nb\nc\n",
			n:     10,
			want:  "a\nb\nc\n",
		},
		{
			name:  "custom n=3",
			input: "one\ntwo\nthree\nfour\nfive\n",
			n:     3,
			want:  "one\ntwo\nthree\n",
		},
		{
			name:  "n=1",
			input: "first\nsecond\nthird\n",
			n:     1,
			want:  "first\n",
		},
		// エッジケース
		{
			name:  "empty input",
			input: "",
			n:     10,
			want:  "",
		},
		{
			name:  "n=0 outputs nothing",
			input: "hello\nworld\n",
			n:     0,
			want:  "",
		},
		{
			name:  "multibyte characters",
			input: "こんにちは\n世界\nGoForge\nテスト\n日本語\n",
			n:     3,
			want:  "こんにちは\n世界\nGoForge\n",
		},
		{
			name:  "lines without trailing newline",
			input: "a\nb\nc",
			n:     2,
			want:  "a\nb\n",
		},
		{
			name:  "blank lines",
			input: "\n\n\nfoo\n",
			n:     2,
			want:  "\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var buf bytes.Buffer
			err := head(r, &buf, tt.n)
			if (err != nil) != tt.wantErr {
				t.Fatalf("head() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("head() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Integration Tests ---

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gf-head")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func TestIntegration(t *testing.T) {
	bin := buildBinary(t)

	t.Run("stdin default 10 lines", func(t *testing.T) {
		var lines []string
		for i := 1; i <= 20; i++ {
			lines = append(lines, fmt.Sprintf("line%d", i))
		}
		input := strings.Join(lines, "\n") + "\n"

		cmd := exec.Command(bin)
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var expected []string
		for i := 1; i <= 10; i++ {
			expected = append(expected, fmt.Sprintf("line%d", i))
		}
		want := strings.Join(expected, "\n") + "\n"
		if string(out) != want {
			t.Errorf("got %q, want %q", string(out), want)
		}
	})

	t.Run("file argument", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "test.txt")
		content := "a\nb\nc\nd\ne\n"
		os.WriteFile(tmpFile, []byte(content), 0644)

		cmd := exec.Command(bin, "-n", "3", tmpFile)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "a\nb\nc\n"
		if string(out) != want {
			t.Errorf("got %q, want %q", string(out), want)
		}
	})

	t.Run("nonexistent file exits 1", func(t *testing.T) {
		cmd := exec.Command(bin, "/nonexistent/file")
		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("expected ExitError, got %T", err)
		}
		if exitErr.ExitCode() != 1 {
			t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
		}
	})

	t.Run("version flag", func(t *testing.T) {
		cmd := exec.Command(bin, "-version")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(string(out), "gf-head version 0.1.0") {
			t.Errorf("version output = %q", string(out))
		}
	})

	t.Run("stdin with hyphen", func(t *testing.T) {
		cmd := exec.Command(bin, "-n", "2", "-")
		cmd.Stdin = strings.NewReader("x\ny\nz\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "x\ny\n"
		if string(out) != want {
			t.Errorf("got %q, want %q", string(out), want)
		}
	})

	t.Run("multiple files with headers", func(t *testing.T) {
		dir := t.TempDir()
		f1 := filepath.Join(dir, "a.txt")
		f2 := filepath.Join(dir, "b.txt")
		os.WriteFile(f1, []byte("alpha\nbeta\n"), 0644)
		os.WriteFile(f2, []byte("gamma\ndelta\n"), 0644)

		cmd := exec.Command(bin, "-n", "1", f1, f2)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := fmt.Sprintf("==> %s <==\nalpha\n\n==> %s <==\ngamma\n", f1, f2)
		if string(out) != want {
			t.Errorf("got %q, want %q", string(out), want)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "empty.txt")
		os.WriteFile(tmpFile, []byte(""), 0644)

		cmd := exec.Command(bin, tmpFile)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "" {
			t.Errorf("expected empty output, got %q", string(out))
		}
	})

	t.Run("pipe: echo | gf-head", func(t *testing.T) {
		cmd := exec.Command(bin, "-n", "1")
		cmd.Stdin = strings.NewReader("piped\ndata\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "piped\n" {
			t.Errorf("got %q, want %q", string(out), "piped\n")
		}
	})
}
