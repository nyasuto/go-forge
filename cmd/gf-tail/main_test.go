package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		n       int
		want    string
		wantErr bool
	}{
		// 正常系
		{
			name:  "last 10 lines from 15 line input",
			input: "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13\n14\n15\n",
			n:     10,
			want:  "6\n7\n8\n9\n10\n11\n12\n13\n14\n15\n",
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
			want:  "three\nfour\nfive\n",
		},
		{
			name:  "n=1",
			input: "first\nsecond\nthird\n",
			n:     1,
			want:  "third\n",
		},
		// 異常系
		{
			name:  "n=0 outputs nothing",
			input: "hello\nworld\n",
			n:     0,
			want:  "",
		},
		{
			name:  "empty input",
			input: "",
			n:     10,
			want:  "",
		},
		// エッジケース
		{
			name:  "multibyte characters",
			input: "こんにちは\n世界\nGoForge\nテスト\n日本語\n",
			n:     3,
			want:  "GoForge\nテスト\n日本語\n",
		},
		{
			name:  "lines without trailing newline",
			input: "a\nb\nc",
			n:     2,
			want:  "b\nc\n",
		},
		{
			name:  "blank lines",
			input: "\n\n\nfoo\n",
			n:     2,
			want:  "\nfoo\n",
		},
		{
			name:  "exact n lines",
			input: "a\nb\nc\n",
			n:     3,
			want:  "a\nb\nc\n",
		},
		{
			name:  "single line",
			input: "only\n",
			n:     5,
			want:  "only\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var buf bytes.Buffer
			err := tail(r, &buf, tt.n)
			if (err != nil) != tt.wantErr {
				t.Fatalf("tail() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("tail() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Integration Tests ---

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gf-tail")
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
		for i := 11; i <= 20; i++ {
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
		want := "c\nd\ne\n"
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
		if !strings.Contains(string(out), "gf-tail version 0.1.0") {
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
		want := "y\nz\n"
		if string(out) != want {
			t.Errorf("got %q, want %q", string(out), want)
		}
	})

	t.Run("multiple files with headers", func(t *testing.T) {
		dir := t.TempDir()
		f1 := filepath.Join(dir, "a.txt")
		f2 := filepath.Join(dir, "b.txt")
		os.WriteFile(f1, []byte("alpha\nbeta\ngamma\n"), 0644)
		os.WriteFile(f2, []byte("delta\nepsilon\nzeta\n"), 0644)

		cmd := exec.Command(bin, "-n", "2", f1, f2)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := fmt.Sprintf("==> %s <==\nbeta\ngamma\n\n==> %s <==\nepsilon\nzeta\n", f1, f2)
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

	t.Run("pipe: echo | gf-tail", func(t *testing.T) {
		cmd := exec.Command(bin, "-n", "1")
		cmd.Stdin = strings.NewReader("piped\ndata\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "data\n" {
			t.Errorf("got %q, want %q", string(out), "data\n")
		}
	})

	// --- Tier 2: -f follow mode tests ---

	t.Run("-f follows appended data", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "follow.txt")
		os.WriteFile(tmpFile, []byte("line1\nline2\nline3\n"), 0644)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, bin, "-f", "-n", "2", tmpFile)
		var outBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &bytes.Buffer{}

		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start: %v", err)
		}

		// Wait for initial output
		time.Sleep(300 * time.Millisecond)

		// Append new data to the file
		f, err := os.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatalf("failed to open for append: %v", err)
		}
		fmt.Fprintln(f, "line4")
		fmt.Fprintln(f, "line5")
		f.Close()

		// Wait for follow to pick up new data
		time.Sleep(500 * time.Millisecond)

		cancel()
		cmd.Wait()

		got := outBuf.String()
		// Should contain initial tail (line2, line3) and appended lines
		if !strings.Contains(got, "line2\n") {
			t.Errorf("expected initial tail to contain 'line2', got %q", got)
		}
		if !strings.Contains(got, "line3\n") {
			t.Errorf("expected initial tail to contain 'line3', got %q", got)
		}
		if !strings.Contains(got, "line4\n") {
			t.Errorf("expected followed output to contain 'line4', got %q", got)
		}
		if !strings.Contains(got, "line5\n") {
			t.Errorf("expected followed output to contain 'line5', got %q", got)
		}
	})

	t.Run("-f without file exits 2", func(t *testing.T) {
		cmd := exec.Command(bin, "-f")
		cmd.Stdin = strings.NewReader("data\n")
		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error for -f without file")
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("expected ExitError, got %T", err)
		}
		if exitErr.ExitCode() != 2 {
			t.Errorf("exit code = %d, want 2", exitErr.ExitCode())
		}
	})

	t.Run("-f with stdin hyphen exits 2", func(t *testing.T) {
		cmd := exec.Command(bin, "-f", "-")
		cmd.Stdin = strings.NewReader("data\n")
		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error for -f with stdin")
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("expected ExitError, got %T", err)
		}
		if exitErr.ExitCode() != 2 {
			t.Errorf("exit code = %d, want 2", exitErr.ExitCode())
		}
	})

	t.Run("-f with multiple files exits 2", func(t *testing.T) {
		dir := t.TempDir()
		f1 := filepath.Join(dir, "a.txt")
		f2 := filepath.Join(dir, "b.txt")
		os.WriteFile(f1, []byte("a\n"), 0644)
		os.WriteFile(f2, []byte("b\n"), 0644)

		cmd := exec.Command(bin, "-f", f1, f2)
		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error for -f with multiple files")
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("expected ExitError, got %T", err)
		}
		if exitErr.ExitCode() != 2 {
			t.Errorf("exit code = %d, want 2", exitErr.ExitCode())
		}
	})

	t.Run("-f detects file truncation", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "trunc.txt")
		os.WriteFile(tmpFile, []byte("old1\nold2\nold3\n"), 0644)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, bin, "-f", "-n", "1", tmpFile)
		var outBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &bytes.Buffer{}

		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start: %v", err)
		}

		time.Sleep(300 * time.Millisecond)

		// Truncate and write new content
		os.WriteFile(tmpFile, []byte("new1\n"), 0644)

		time.Sleep(500 * time.Millisecond)

		cancel()
		cmd.Wait()

		got := outBuf.String()
		if !strings.Contains(got, "new1\n") {
			t.Errorf("expected truncated+rewritten content 'new1', got %q", got)
		}
	})
}
