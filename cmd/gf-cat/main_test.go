package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// cat関数の単体テスト（Tier 1: 基本機能）
func TestCat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// 正常系
		{
			name:  "単一行",
			input: "hello world\n",
			want:  "hello world\n",
		},
		{
			name:  "複数行",
			input: "line1\nline2\nline3\n",
			want:  "line1\nline2\nline3\n",
		},
		{
			name:  "改行なし末尾",
			input: "no newline",
			want:  "no newline",
		},
		// エッジケース
		{
			name:  "空入力",
			input: "",
			want:  "",
		},
		{
			name:  "マルチバイト文字",
			input: "日本語テスト\nこんにちは世界\n",
			want:  "日本語テスト\nこんにちは世界\n",
		},
		{
			name:  "大きな入力",
			input: strings.Repeat("abcdefghij\n", 10000),
			want:  strings.Repeat("abcdefghij\n", 10000),
		},
		{
			name:  "空行のみ",
			input: "\n\n\n",
			want:  "\n\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var buf bytes.Buffer
			opts := &options{}
			err := cat(r, &buf, opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("cat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("cat() = %q, want %q", got, tt.want)
			}
		})
	}
}

// cat関数の単体テスト（Tier 2: -n 行番号表示）
func TestCatNumberLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// 正常系
		{
			name:  "単一行に行番号",
			input: "hello\n",
			want:  "     1\thello\n",
		},
		{
			name:  "複数行に行番号",
			input: "line1\nline2\nline3\n",
			want:  "     1\tline1\n     2\tline2\n     3\tline3\n",
		},
		{
			name:  "空行にも行番号",
			input: "a\n\nb\n",
			want:  "     1\ta\n     2\t\n     3\tb\n",
		},
		// エッジケース
		{
			name:  "空入力",
			input: "",
			want:  "",
		},
		{
			name:  "マルチバイト文字に行番号",
			input: "日本語\n世界\n",
			want:  "     1\t日本語\n     2\t世界\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var buf bytes.Buffer
			opts := &options{number: true}
			err := cat(r, &buf, opts)
			if err != nil {
				t.Errorf("cat() error = %v", err)
				return
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("cat() = %q, want %q", got, tt.want)
			}
		})
	}
}

// cat関数の単体テスト（Tier 2: -s 連続空行圧縮）
func TestCatSqueezeBlank(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// 正常系
		{
			name:  "連続空行を圧縮",
			input: "a\n\n\n\nb\n",
			want:  "a\n\nb\n",
		},
		{
			name:  "空行なしはそのまま",
			input: "a\nb\nc\n",
			want:  "a\nb\nc\n",
		},
		{
			name:  "先頭の連続空行を圧縮",
			input: "\n\n\na\n",
			want:  "\na\n",
		},
		// エッジケース
		{
			name:  "空入力",
			input: "",
			want:  "",
		},
		{
			name:  "全て空行",
			input: "\n\n\n\n",
			want:  "\n",
		},
		{
			name:  "空行1行はそのまま",
			input: "a\n\nb\n",
			want:  "a\n\nb\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var buf bytes.Buffer
			opts := &options{squeeze: true}
			err := cat(r, &buf, opts)
			if err != nil {
				t.Errorf("cat() error = %v", err)
				return
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("cat() = %q, want %q", got, tt.want)
			}
		})
	}
}

// cat関数の単体テスト（Tier 2: -n と -s の組み合わせ）
func TestCatNumberAndSqueeze(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "行番号+圧縮の組み合わせ",
			input: "a\n\n\n\nb\n",
			want:  "     1\ta\n     2\t\n     3\tb\n",
		},
		{
			name:  "圧縮後の行番号は連番",
			input: "x\n\n\ny\n\n\nz\n",
			want:  "     1\tx\n     2\t\n     3\ty\n     4\t\n     5\tz\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var buf bytes.Buffer
			opts := &options{number: true, squeeze: true}
			err := cat(r, &buf, opts)
			if err != nil {
				t.Errorf("cat() error = %v", err)
				return
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("cat() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ファイル読み取りの統合テスト
func TestCatFile(t *testing.T) {
	// テスト用の一時ファイル作成
	dir := t.TempDir()

	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "file2.txt")
	fileBlank := filepath.Join(dir, "blank.txt")
	os.WriteFile(file1, []byte("content of file1\n"), 0644)
	os.WriteFile(file2, []byte("content of file2\n"), 0644)
	os.WriteFile(fileBlank, []byte("a\n\n\n\nb\n"), 0644)

	// バイナリをビルド
	binary := filepath.Join(dir, "gf-cat")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	tests := []struct {
		name     string
		args     []string
		stdin    string
		want     string
		wantExit int
	}{
		// 正常系（Tier 1）
		{
			name: "単一ファイル",
			args: []string{file1},
			want: "content of file1\n",
		},
		{
			name: "複数ファイル連結",
			args: []string{file1, file2},
			want: "content of file1\ncontent of file2\n",
		},
		{
			name:  "stdinから読み取り（引数なし）",
			args:  []string{},
			stdin: "stdin input\n",
			want:  "stdin input\n",
		},
		{
			name:  "ハイフンでstdin",
			args:  []string{"-"},
			stdin: "stdin via dash\n",
			want:  "stdin via dash\n",
		},
		{
			name:  "ファイルとstdinの混合",
			args:  []string{file1, "-"},
			stdin: "from stdin\n",
			want:  "content of file1\nfrom stdin\n",
		},
		// 異常系
		{
			name:     "存在しないファイル",
			args:     []string{filepath.Join(dir, "nonexistent.txt")},
			wantExit: 1,
		},
		{
			name:     "存在しないファイルと存在するファイル",
			args:     []string{filepath.Join(dir, "nonexistent.txt"), file1},
			want:     "content of file1\n",
			wantExit: 1,
		},
		// バージョン表示
		{
			name: "バージョン表示",
			args: []string{"-version"},
			want: "gf-cat version 0.1.0\n",
		},
		// Tier 2: -n 行番号表示
		{
			name: "-nで行番号表示",
			args: []string{"-n", file1},
			want: "     1\tcontent of file1\n",
		},
		{
			name: "-nで複数ファイル連番",
			args: []string{"-n", file1, file2},
			want: "     1\tcontent of file1\n     2\tcontent of file2\n",
		},
		// Tier 2: -s 連続空行圧縮
		{
			name: "-sで連続空行圧縮",
			args: []string{"-s", fileBlank},
			want: "a\n\nb\n",
		},
		// Tier 2: -n と -s の組み合わせ
		{
			name: "-n -sの組み合わせ",
			args: []string{"-n", "-s", fileBlank},
			want: "     1\ta\n     2\t\n     3\tb\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binary, tt.args...)
			if tt.stdin != "" {
				cmd.Stdin = strings.NewReader(tt.stdin)
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if exitCode != tt.wantExit {
				t.Errorf("exit code = %d, want %d\nstderr: %s", exitCode, tt.wantExit, stderr.String())
			}

			if tt.want != "" {
				if got := stdout.String(); got != tt.want {
					t.Errorf("stdout = %q, want %q", got, tt.want)
				}
			}
		})
	}
}
