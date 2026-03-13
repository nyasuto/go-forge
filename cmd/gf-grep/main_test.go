package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrep(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		prefix  string
		want    string
		found   bool
	}{
		// 正常系
		{
			name:    "単一マッチ",
			pattern: "hello",
			input:   "hello world\ngoodbye world\n",
			want:    "hello world\n",
			found:   true,
		},
		{
			name:    "複数マッチ",
			pattern: "world",
			input:   "hello world\ngoodbye world\nfoo\n",
			want:    "hello world\ngoodbye world\n",
			found:   true,
		},
		{
			name:    "プレフィックス付き",
			pattern: "foo",
			input:   "foo bar\nbaz\nfoo qux\n",
			prefix:  "test.txt:",
			want:    "test.txt:foo bar\ntest.txt:foo qux\n",
			found:   true,
		},
		// 異常系
		{
			name:    "マッチなし",
			pattern: "xyz",
			input:   "hello\nworld\n",
			want:    "",
			found:   false,
		},
		{
			name:    "空パターンは全行マッチ",
			pattern: "",
			input:   "aaa\nbbb\n",
			want:    "aaa\nbbb\n",
			found:   true,
		},
		// エッジケース
		{
			name:    "空入力",
			pattern: "test",
			input:   "",
			want:    "",
			found:   false,
		},
		{
			name:    "マルチバイト文字",
			pattern: "日本語",
			input:   "これは日本語のテストです\nEnglish only\n日本語マッチ\n",
			want:    "これは日本語のテストです\n日本語マッチ\n",
			found:   true,
		},
		{
			name:    "大文字小文字は区別する",
			pattern: "Hello",
			input:   "hello\nHello\nHELLO\n",
			want:    "Hello\n",
			found:   true,
		},
		{
			name:    "行の一部にマッチ",
			pattern: "err",
			input:   "error occurred\nno errors\nclean\n",
			want:    "error occurred\nno errors\n",
			found:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var buf bytes.Buffer
			got := grep(r, &buf, tt.pattern, tt.prefix)

			if got != tt.found {
				t.Errorf("grep() found = %v, want %v", got, tt.found)
			}
			if buf.String() != tt.want {
				t.Errorf("grep() output = %q, want %q", buf.String(), tt.want)
			}
		})
	}
}

// buildBinary builds the gf-grep binary for integration tests.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gf-grep")
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

	t.Run("stdin入力でマッチ", func(t *testing.T) {
		cmd := exec.Command(bin, "hello")
		cmd.Stdin = strings.NewReader("hello world\ngoodbye\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "hello world\n" {
			t.Errorf("got %q, want %q", string(out), "hello world\n")
		}
	})

	t.Run("stdin入力でマッチなし→exit 1", func(t *testing.T) {
		cmd := exec.Command(bin, "xyz")
		cmd.Stdin = strings.NewReader("hello\nworld\n")
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("expected exit code 1")
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok || exitErr.ExitCode() != 1 {
			t.Errorf("expected exit code 1, got %v", err)
		}
		if string(out) != "" {
			t.Errorf("expected no output, got %q", string(out))
		}
	})

	t.Run("ファイル入力", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "test.txt")
		os.WriteFile(tmp, []byte("alpha\nbeta\ngamma\nbeta2\n"), 0644)

		cmd := exec.Command(bin, "beta", tmp)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "beta\nbeta2\n" {
			t.Errorf("got %q, want %q", string(out), "beta\nbeta2\n")
		}
	})

	t.Run("複数ファイルでファイル名表示", func(t *testing.T) {
		dir := t.TempDir()
		f1 := filepath.Join(dir, "a.txt")
		f2 := filepath.Join(dir, "b.txt")
		os.WriteFile(f1, []byte("foo\nbar\n"), 0644)
		os.WriteFile(f2, []byte("baz\nfoo\n"), 0644)

		cmd := exec.Command(bin, "foo", f1, f2)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := f1 + ":foo\n" + f2 + ":foo\n"
		if string(out) != want {
			t.Errorf("got %q, want %q", string(out), want)
		}
	})

	t.Run("存在しないファイル→stderr出力", func(t *testing.T) {
		cmd := exec.Command(bin, "test", "/nonexistent/file.txt")
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(string(out), "gf-grep:") {
			t.Errorf("expected error message, got %q", string(out))
		}
	})

	t.Run("パターンなし→exit 2", func(t *testing.T) {
		cmd := exec.Command(bin)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("expected exit code 2")
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok || exitErr.ExitCode() != 2 {
			t.Errorf("expected exit code 2, got %v", err)
		}
		if !strings.Contains(string(out), "パターンが指定されていません") {
			t.Errorf("expected usage error, got %q", string(out))
		}
	})

	t.Run("--version表示", func(t *testing.T) {
		cmd := exec.Command(bin, "--version")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(string(out), "gf-grep version "+version) {
			t.Errorf("got %q", string(out))
		}
	})

	t.Run("ハイフンでstdin読み取り", func(t *testing.T) {
		cmd := exec.Command(bin, "match", "-")
		cmd.Stdin = strings.NewReader("match this\nnope\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "match this\n" {
			t.Errorf("got %q, want %q", string(out), "match this\n")
		}
	})
}
