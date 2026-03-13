package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// cat関数の単体テスト
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
			err := cat(r, &buf)
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

// ファイル読み取りの統合テスト
func TestCatFile(t *testing.T) {
	// テスト用の一時ファイル作成
	dir := t.TempDir()

	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "file2.txt")
	os.WriteFile(file1, []byte("content of file1\n"), 0644)
	os.WriteFile(file2, []byte("content of file2\n"), 0644)

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
		// 正常系
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
