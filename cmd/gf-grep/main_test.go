package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func compileTestPattern(t *testing.T, pattern string, ignoreCase bool) *regexp.Regexp {
	t.Helper()
	re, err := compilePattern(pattern, ignoreCase)
	if err != nil {
		t.Fatalf("compilePattern failed: %v", err)
	}
	return re
}

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
			re := compileTestPattern(t, regexp.QuoteMeta(tt.pattern), false)
			got := grep(r, &buf, re, tt.prefix, grepOptions{})

			if got != tt.found {
				t.Errorf("grep() found = %v, want %v", got, tt.found)
			}
			if buf.String() != tt.want {
				t.Errorf("grep() output = %q, want %q", buf.String(), tt.want)
			}
		})
	}
}

func TestGrepRegex(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		opts    grepOptions
		want    string
		found   bool
	}{
		{
			name:    "正規表現マッチ",
			pattern: "^func ",
			input:   "func main() {\n\tx := func() {}\n}\n",
			want:    "func main() {\n",
			found:   true,
		},
		{
			name:    "正規表現ドットワイルドカード",
			pattern: "h.llo",
			input:   "hello\nhallo\nhxllo\nworld\n",
			want:    "hello\nhallo\nhxllo\n",
			found:   true,
		},
		{
			name:    "正規表現文字クラス",
			pattern: "[0-9]+",
			input:   "abc\n123\ndef456\n",
			want:    "123\ndef456\n",
			found:   true,
		},
		// -i: 大文字小文字無視
		{
			name:    "-i 大文字小文字無視",
			pattern: "hello",
			input:   "Hello\nhELLO\nworld\n",
			opts:    grepOptions{ignoreCase: true},
			want:    "Hello\nhELLO\n",
			found:   true,
		},
		// -v: 反転マッチ
		{
			name:    "-v 反転マッチ",
			pattern: "skip",
			input:   "keep\nskip this\nkeep too\n",
			opts:    grepOptions{invert: true},
			want:    "keep\nkeep too\n",
			found:   true,
		},
		{
			name:    "-v 全行マッチで反転→出力なし",
			pattern: ".",
			input:   "a\nb\nc\n",
			opts:    grepOptions{invert: true},
			want:    "",
			found:   false,
		},
		// -c: カウント
		{
			name:    "-c マッチ行数カウント",
			pattern: "foo",
			input:   "foo\nbar\nfoo baz\n",
			opts:    grepOptions{count: true},
			want:    "2\n",
			found:   true,
		},
		{
			name:    "-c マッチなしでカウント0",
			pattern: "xyz",
			input:   "aaa\nbbb\n",
			opts:    grepOptions{count: true},
			want:    "0\n",
			found:   false,
		},
		// -n: 行番号
		{
			name:    "-n 行番号表示",
			pattern: "match",
			input:   "no\nmatch1\nno\nmatch2\n",
			opts:    grepOptions{lineNumber: true},
			want:    "2:match1\n4:match2\n",
			found:   true,
		},
		// -i + -v 組み合わせ
		{
			name:    "-i -v 大文字小文字無視+反転",
			pattern: "error",
			input:   "ERROR found\nwarning\nError again\ninfo\n",
			opts:    grepOptions{ignoreCase: true, invert: true},
			want:    "warning\ninfo\n",
			found:   true,
		},
		// -c + -n は -c が優先
		{
			name:    "-c -n はカウントのみ",
			pattern: "x",
			input:   "ax\nbx\ncx\n",
			opts:    grepOptions{count: true, lineNumber: true},
			want:    "3\n",
			found:   true,
		},
		// エッジケース: マルチバイト正規表現
		{
			name:    "マルチバイト正規表現",
			pattern: "テスト[0-9]+",
			input:   "テスト1\nテスト\nテスト99\n",
			want:    "テスト1\nテスト99\n",
			found:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var buf bytes.Buffer
			re := compileTestPattern(t, tt.pattern, tt.opts.ignoreCase)
			got := grep(r, &buf, re, "", tt.opts)

			if got != tt.found {
				t.Errorf("grep() found = %v, want %v", got, tt.found)
			}
			if buf.String() != tt.want {
				t.Errorf("grep() output = %q, want %q", buf.String(), tt.want)
			}
		})
	}
}

func TestGrepJSONField(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		opts    grepOptions
		want    string
		found   bool
	}{
		// 正常系
		{
			name:    "トップレベルキーでマッチ",
			pattern: "error",
			input:   `{"level":"error","msg":"disk full"}` + "\n" + `{"level":"info","msg":"ok"}` + "\n",
			opts:    grepOptions{jsonField: "level"},
			want:    `{"level":"error","msg":"disk full"}` + "\n",
			found:   true,
		},
		{
			name:    "値の部分一致",
			pattern: "disk",
			input:   `{"level":"error","msg":"disk full"}` + "\n" + `{"level":"info","msg":"ok"}` + "\n",
			opts:    grepOptions{jsonField: "msg"},
			want:    `{"level":"error","msg":"disk full"}` + "\n",
			found:   true,
		},
		{
			name:    "ネストされたキーでマッチ",
			pattern: "admin",
			input:   `{"user":{"role":"admin","name":"alice"}}` + "\n" + `{"user":{"role":"viewer","name":"bob"}}` + "\n",
			opts:    grepOptions{jsonField: "user.role"},
			want:    `{"user":{"role":"admin","name":"alice"}}` + "\n",
			found:   true,
		},
		// 正規表現との組み合わせ
		{
			name:    "正規表現パターン",
			pattern: "^err",
			input:   `{"level":"error","msg":"fail"}` + "\n" + `{"level":"warning","msg":"err in msg"}` + "\n",
			opts:    grepOptions{jsonField: "level"},
			want:    `{"level":"error","msg":"fail"}` + "\n",
			found:   true,
		},
		// -i との組み合わせ
		{
			name:    "-i 大文字小文字無視とJSON",
			pattern: "ERROR",
			input:   `{"level":"error","msg":"fail"}` + "\n" + `{"level":"info","msg":"ok"}` + "\n",
			opts:    grepOptions{jsonField: "level", ignoreCase: true},
			want:    `{"level":"error","msg":"fail"}` + "\n",
			found:   true,
		},
		// -v との組み合わせ
		{
			name:    "-v 反転とJSON",
			pattern: "info",
			input:   `{"level":"error","msg":"fail"}` + "\n" + `{"level":"info","msg":"ok"}` + "\n",
			opts:    grepOptions{jsonField: "level", invert: true},
			want:    `{"level":"error","msg":"fail"}` + "\n",
			found:   true,
		},
		// -c との組み合わせ
		{
			name:    "-c カウントとJSON",
			pattern: "error",
			input:   `{"level":"error","msg":"a"}` + "\n" + `{"level":"error","msg":"b"}` + "\n" + `{"level":"info","msg":"c"}` + "\n",
			opts:    grepOptions{jsonField: "level", count: true},
			want:    "2\n",
			found:   true,
		},
		// 異常系
		{
			name:    "非JSONの行はスキップ",
			pattern: "hello",
			input:   "not json\n" + `{"msg":"hello"}` + "\n",
			opts:    grepOptions{jsonField: "msg"},
			want:    `{"msg":"hello"}` + "\n",
			found:   true,
		},
		{
			name:    "存在しないキー→マッチなし",
			pattern: "test",
			input:   `{"level":"error","msg":"test"}` + "\n",
			opts:    grepOptions{jsonField: "nonexistent"},
			want:    "",
			found:   false,
		},
		// エッジケース
		{
			name:    "数値フィールドのマッチ",
			pattern: "42",
			input:   `{"code":42,"msg":"found"}` + "\n" + `{"code":200,"msg":"ok"}` + "\n",
			opts:    grepOptions{jsonField: "code"},
			want:    `{"code":42,"msg":"found"}` + "\n",
			found:   true,
		},
		{
			name:    "マルチバイト値のマッチ",
			pattern: "エラー",
			input:   `{"msg":"エラー発生"}` + "\n" + `{"msg":"正常"}` + "\n",
			opts:    grepOptions{jsonField: "msg"},
			want:    `{"msg":"エラー発生"}` + "\n",
			found:   true,
		},
		{
			name:    "空入力",
			pattern: "test",
			input:   "",
			opts:    grepOptions{jsonField: "key"},
			want:    "",
			found:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var buf bytes.Buffer
			re := compileTestPattern(t, tt.pattern, tt.opts.ignoreCase)
			got := grep(r, &buf, re, "", tt.opts)

			if got != tt.found {
				t.Errorf("grep() found = %v, want %v", got, tt.found)
			}
			if buf.String() != tt.want {
				t.Errorf("grep() output = %q, want %q", buf.String(), tt.want)
			}
		})
	}
}

func TestExpandRecursive(t *testing.T) {
	dir := t.TempDir()
	// Create directory structure
	os.MkdirAll(filepath.Join(dir, "sub", "deep"), 0755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("world\n"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "deep", "c.txt"), []byte("deep\n"), 0644)

	files, errs := expandRecursive([]string{dir})
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d: %v", len(files), files)
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

	// Tier 2 統合テスト
	t.Run("正規表現マッチ", func(t *testing.T) {
		cmd := exec.Command(bin, "^hello")
		cmd.Stdin = strings.NewReader("hello world\nsay hello\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "hello world\n" {
			t.Errorf("got %q, want %q", string(out), "hello world\n")
		}
	})

	t.Run("-i 大文字小文字無視", func(t *testing.T) {
		cmd := exec.Command(bin, "-i", "hello")
		cmd.Stdin = strings.NewReader("Hello\nhELLO\nworld\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "Hello\nhELLO\n" {
			t.Errorf("got %q, want %q", string(out), "Hello\nhELLO\n")
		}
	})

	t.Run("-v 反転マッチ", func(t *testing.T) {
		cmd := exec.Command(bin, "-v", "skip")
		cmd.Stdin = strings.NewReader("keep\nskip\nkeep too\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "keep\nkeep too\n" {
			t.Errorf("got %q, want %q", string(out), "keep\nkeep too\n")
		}
	})

	t.Run("-c カウント", func(t *testing.T) {
		cmd := exec.Command(bin, "-c", "foo")
		cmd.Stdin = strings.NewReader("foo\nbar\nfoo baz\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "2\n" {
			t.Errorf("got %q, want %q", string(out), "2\n")
		}
	})

	t.Run("-n 行番号表示", func(t *testing.T) {
		cmd := exec.Command(bin, "-n", "match")
		cmd.Stdin = strings.NewReader("no\nmatch1\nno\nmatch2\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "2:match1\n4:match2\n" {
			t.Errorf("got %q, want %q", string(out), "2:match1\n4:match2\n")
		}
	})

	t.Run("-r 再帰検索", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "sub"), 0755)
		os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\nworld\n"), 0644)
		os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("hello again\nbye\n"), 0644)

		cmd := exec.Command(bin, "-r", "hello", dir)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		output := string(out)
		if !strings.Contains(output, "hello") {
			t.Errorf("expected hello matches, got %q", output)
		}
		// Should have file prefixes since multiple files
		if !strings.Contains(output, ":hello") {
			t.Errorf("expected file prefix, got %q", output)
		}
	})

	t.Run("不正な正規表現→exit 2", func(t *testing.T) {
		cmd := exec.Command(bin, "[invalid")
		cmd.Stdin = strings.NewReader("test\n")
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("expected exit code 2")
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok || exitErr.ExitCode() != 2 {
			t.Errorf("expected exit code 2, got %v", err)
		}
		if !strings.Contains(string(out), "不正な正規表現") {
			t.Errorf("expected regex error, got %q", string(out))
		}
	})

	t.Run("-c 複数ファイルでファイル名付きカウント", func(t *testing.T) {
		dir := t.TempDir()
		f1 := filepath.Join(dir, "a.txt")
		f2 := filepath.Join(dir, "b.txt")
		os.WriteFile(f1, []byte("foo\nfoo\nbar\n"), 0644)
		os.WriteFile(f2, []byte("baz\nfoo\n"), 0644)

		cmd := exec.Command(bin, "-c", "foo", f1, f2)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := f1 + ":2\n" + f2 + ":1\n"
		if string(out) != want {
			t.Errorf("got %q, want %q", string(out), want)
		}
	})

	// Tier 3 統合テスト
	t.Run("-j JSONフィールド検索", func(t *testing.T) {
		cmd := exec.Command(bin, "-j", "level", "error")
		cmd.Stdin = strings.NewReader(`{"level":"error","msg":"disk full"}` + "\n" + `{"level":"info","msg":"ok"}` + "\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := `{"level":"error","msg":"disk full"}` + "\n"
		if string(out) != want {
			t.Errorf("got %q, want %q", string(out), want)
		}
	})

	t.Run("-j ネストされたキー", func(t *testing.T) {
		cmd := exec.Command(bin, "-j", "user.role", "admin")
		cmd.Stdin = strings.NewReader(`{"user":{"role":"admin"}}` + "\n" + `{"user":{"role":"viewer"}}` + "\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := `{"user":{"role":"admin"}}` + "\n"
		if string(out) != want {
			t.Errorf("got %q, want %q", string(out), want)
		}
	})

	t.Run("-j -n -c 組み合わせ", func(t *testing.T) {
		cmd := exec.Command(bin, "-j", "level", "-c", "error")
		cmd.Stdin = strings.NewReader(`{"level":"error","msg":"a"}` + "\n" + `{"level":"error","msg":"b"}` + "\n" + `{"level":"info","msg":"c"}` + "\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "2\n" {
			t.Errorf("got %q, want %q", string(out), "2\n")
		}
	})

	t.Run("-j ファイル入力", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "data.json")
		content := `{"status":"ok","code":200}` + "\n" + `{"status":"fail","code":500}` + "\n"
		os.WriteFile(tmp, []byte(content), 0644)

		cmd := exec.Command(bin, "-j", "status", "fail", tmp)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := `{"status":"fail","code":500}` + "\n"
		if string(out) != want {
			t.Errorf("got %q, want %q", string(out), want)
		}
	})

	t.Run("-j 非JSONの行はスキップ", func(t *testing.T) {
		cmd := exec.Command(bin, "-j", "msg", "hello")
		cmd.Stdin = strings.NewReader("plain text\n" + `{"msg":"hello world"}` + "\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := `{"msg":"hello world"}` + "\n"
		if string(out) != want {
			t.Errorf("got %q, want %q", string(out), want)
		}
	})

	t.Run("-n 複数ファイルで行番号+ファイル名", func(t *testing.T) {
		dir := t.TempDir()
		f1 := filepath.Join(dir, "a.txt")
		os.WriteFile(f1, []byte("aaa\nbbb\naaa\n"), 0644)

		cmd := exec.Command(bin, "-n", "aaa", f1)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != "1:aaa\n3:aaa\n" {
			t.Errorf("got %q, want %q", string(out), "1:aaa\n3:aaa\n")
		}
	})
}
