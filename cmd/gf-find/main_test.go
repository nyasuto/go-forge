package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gf-find")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

// createTestTree creates a temporary directory tree for testing.
// Returns the root path.
func createTestTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// Create directory structure:
	// root/
	//   file1.txt
	//   file2.go
	//   sub/
	//     file3.txt
	//     file4.go
	//     deep/
	//       file5.json
	//       ファイル.txt
	dirs := []string{
		filepath.Join(root, "sub"),
		filepath.Join(root, "sub", "deep"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	files := []string{
		filepath.Join(root, "file1.txt"),
		filepath.Join(root, "file2.go"),
		filepath.Join(root, "sub", "file3.txt"),
		filepath.Join(root, "sub", "file4.go"),
		filepath.Join(root, "sub", "deep", "file5.json"),
		filepath.Join(root, "sub", "deep", "ファイル.txt"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func runFind(t *testing.T, bin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var stdout, stderr strings.Builder
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
	return stdout.String(), stderr.String(), exitCode
}

func sortedLines(s string) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	sort.Strings(lines)
	return lines
}

// Unit tests for matchName
func TestMatchName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		pattern string
		want    bool
	}{
		{"empty pattern matches anything", "file.txt", "", true},
		{"exact match", "file.txt", "file.txt", true},
		{"glob star", "file.txt", "*.txt", true},
		{"glob no match", "file.go", "*.txt", false},
		{"question mark", "file1.txt", "file?.txt", true},
		{"multibyte name", "ファイル.txt", "*.txt", true},
		{"invalid pattern", "file.txt", "[", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchName(tt.input, tt.pattern)
			if got != tt.want {
				t.Errorf("matchName(%q, %q) = %v, want %v", tt.input, tt.pattern, got, tt.want)
			}
		})
	}
}

// Integration tests
func TestFindIntegration(t *testing.T) {
	bin := buildBinary(t)
	root := createTestTree(t)

	tests := []struct {
		name       string
		args       []string
		wantFiles  []string
		wantExit   int
		wantStderr string
	}{
		{
			name: "all files in tree (no -name)",
			args: []string{root},
			wantFiles: []string{
				root,
				filepath.Join(root, "file1.txt"),
				filepath.Join(root, "file2.go"),
				filepath.Join(root, "sub"),
				filepath.Join(root, "sub", "deep"),
				filepath.Join(root, "sub", "deep", "file5.json"),
				filepath.Join(root, "sub", "deep", "ファイル.txt"),
				filepath.Join(root, "sub", "file3.txt"),
				filepath.Join(root, "sub", "file4.go"),
			},
			wantExit: 0,
		},
		{
			name: "find txt files",
			args: []string{"-name", "*.txt", root},
			wantFiles: []string{
				filepath.Join(root, "file1.txt"),
				filepath.Join(root, "sub", "deep", "ファイル.txt"),
				filepath.Join(root, "sub", "file3.txt"),
			},
			wantExit: 0,
		},
		{
			name: "find go files",
			args: []string{"-name", "*.go", root},
			wantFiles: []string{
				filepath.Join(root, "file2.go"),
				filepath.Join(root, "sub", "file4.go"),
			},
			wantExit: 0,
		},
		{
			name: "find specific file",
			args: []string{"-name", "file5.json", root},
			wantFiles: []string{
				filepath.Join(root, "sub", "deep", "file5.json"),
			},
			wantExit: 0,
		},
		{
			name: "no matches",
			args: []string{"-name", "*.rs", root},
			wantFiles: nil,
			wantExit:  0,
		},
		{
			name: "multibyte filename match",
			args: []string{"-name", "ファイル*", root},
			wantFiles: []string{
				filepath.Join(root, "sub", "deep", "ファイル.txt"),
			},
			wantExit: 0,
		},
		{
			name:       "nonexistent path",
			args:       []string{filepath.Join(root, "nonexistent")},
			wantFiles:  nil,
			wantExit:   1,
			wantStderr: "gf-find:",
		},
		{
			name: "multiple paths",
			args: []string{"-name", "*.go", root, filepath.Join(root, "sub")},
			wantFiles: []string{
				filepath.Join(root, "file2.go"),
				filepath.Join(root, "sub", "file4.go"),
				filepath.Join(root, "sub", "file4.go"),
			},
			wantExit: 0,
		},
		{
			name:     "version flag",
			args:     []string{"-version"},
			wantExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitCode := runFind(t, bin, tt.args...)

			if exitCode != tt.wantExit {
				t.Errorf("exit code = %d, want %d (stderr: %s)", exitCode, tt.wantExit, stderr)
			}

			if tt.name == "version flag" {
				if !strings.Contains(stdout, "gf-find version") {
					t.Errorf("expected version output, got %q", stdout)
				}
				return
			}

			if tt.wantStderr != "" && !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("stderr = %q, want containing %q", stderr, tt.wantStderr)
			}

			gotFiles := sortedLines(stdout)
			wantFiles := make([]string, len(tt.wantFiles))
			copy(wantFiles, tt.wantFiles)
			sort.Strings(wantFiles)

			if len(gotFiles) != len(wantFiles) {
				t.Errorf("got %d files, want %d\ngot:  %v\nwant: %v", len(gotFiles), len(wantFiles), gotFiles, wantFiles)
				return
			}
			for i := range gotFiles {
				if gotFiles[i] != wantFiles[i] {
					t.Errorf("file[%d] = %q, want %q", i, gotFiles[i], wantFiles[i])
				}
			}
		})
	}
}

// Edge case: empty directory
func TestFindEmptyDir(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()

	stdout, _, exitCode := runFind(t, bin, root)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	// Should list just the root directory itself
	got := strings.TrimSpace(stdout)
	if got != root {
		t.Errorf("got %q, want %q", got, root)
	}
}

// Edge case: single file as argument
func TestFindSingleFile(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()
	f := filepath.Join(root, "test.txt")
	os.WriteFile(f, []byte("hello"), 0644)

	// Without -name: should print the file
	stdout, _, exitCode := runFind(t, bin, f)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stdout) != f {
		t.Errorf("got %q, want %q", strings.TrimSpace(stdout), f)
	}

	// With matching -name
	stdout, _, exitCode = runFind(t, bin, "-name", "*.txt", f)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stdout) != f {
		t.Errorf("got %q, want %q", strings.TrimSpace(stdout), f)
	}

	// With non-matching -name
	stdout, _, exitCode = runFind(t, bin, "-name", "*.go", f)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty output, got %q", stdout)
	}
}
