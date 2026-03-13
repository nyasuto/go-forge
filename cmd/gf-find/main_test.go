package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
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

// Unit tests for parseSizeExpr
func TestParseSizeExpr(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantOp  string
		wantN   int64
		wantErr bool
	}{
		{"bytes", "100c", "=", 100, false},
		{"kilobytes", "10k", "=", 10240, false},
		{"megabytes", "2M", "=", 2 * 1024 * 1024, false},
		{"gigabytes", "1G", "=", 1024 * 1024 * 1024, false},
		{"blocks default", "10", "=", 5120, false},
		{"greater than", "+100c", "+", 100, false},
		{"less than", "-50k", "-", 51200, false},
		{"empty", "", "", 0, true},
		{"only sign", "+", "", 0, true},
		{"only unit", "+c", "", 0, true},
		{"invalid unit", "10x", "", 0, true},
		{"not a number", "+abck", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, n, err := parseSizeExpr(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSizeExpr(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if op != tt.wantOp {
					t.Errorf("op = %q, want %q", op, tt.wantOp)
				}
				if n != tt.wantN {
					t.Errorf("n = %d, want %d", n, tt.wantN)
				}
			}
		})
	}
}

// Unit tests for parseMtimeExpr
func TestParseMtimeExpr(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantOp  string
		wantN   int
		wantErr bool
	}{
		{"exact days", "7", "=", 7, false},
		{"more than", "+30", "+", 30, false},
		{"less than", "-1", "-", 1, false},
		{"zero days", "0", "=", 0, false},
		{"empty", "", "", 0, true},
		{"only sign", "+", "", 0, true},
		{"not a number", "+abc", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, n, err := parseMtimeExpr(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMtimeExpr(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if op != tt.wantOp {
					t.Errorf("op = %q, want %q", op, tt.wantOp)
				}
				if n != tt.wantN {
					t.Errorf("n = %d, want %d", n, tt.wantN)
				}
			}
		})
	}
}

// Unit tests for matchType
func TestMatchType(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "test.txt")
	os.WriteFile(f, []byte("x"), 0644)
	d := filepath.Join(root, "subdir")
	os.Mkdir(d, 0755)

	fInfo, _ := os.Stat(f)
	dInfo, _ := os.Stat(d)

	tests := []struct {
		name       string
		fi         os.FileInfo
		typeFilter string
		want       bool
	}{
		{"empty filter matches file", fInfo, "", true},
		{"empty filter matches dir", dInfo, "", true},
		{"f matches file", fInfo, "f", true},
		{"f rejects dir", dInfo, "f", false},
		{"d matches dir", dInfo, "d", true},
		{"d rejects file", fInfo, "d", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchType(tt.fi, tt.typeFilter)
			if got != tt.want {
				t.Errorf("matchType(%q, %q) = %v, want %v", tt.fi.Name(), tt.typeFilter, got, tt.want)
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
			name:     "no matches",
			args:     []string{"-name", "*.rs", root},
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

// Integration tests for -type filter
func TestFindTypeFilter(t *testing.T) {
	bin := buildBinary(t)
	root := createTestTree(t)

	t.Run("type f - files only", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "f", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		want := []string{
			filepath.Join(root, "file1.txt"),
			filepath.Join(root, "file2.go"),
			filepath.Join(root, "sub", "deep", "file5.json"),
			filepath.Join(root, "sub", "deep", "ファイル.txt"),
			filepath.Join(root, "sub", "file3.txt"),
			filepath.Join(root, "sub", "file4.go"),
		}
		sort.Strings(want)
		if len(got) != len(want) {
			t.Errorf("got %d files, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
			return
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("file[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("type d - directories only", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "d", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		want := []string{
			root,
			filepath.Join(root, "sub"),
			filepath.Join(root, "sub", "deep"),
		}
		sort.Strings(want)
		if len(got) != len(want) {
			t.Errorf("got %d dirs, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
			return
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("dir[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("type f with name filter", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "f", "-name", "*.txt", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		want := []string{
			filepath.Join(root, "file1.txt"),
			filepath.Join(root, "sub", "deep", "ファイル.txt"),
			filepath.Join(root, "sub", "file3.txt"),
		}
		sort.Strings(want)
		if len(got) != len(want) {
			t.Errorf("got %d files, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
		}
	})

	t.Run("invalid type value", func(t *testing.T) {
		_, stderr, exitCode := runFind(t, bin, "-type", "x", root)
		if exitCode != 2 {
			t.Errorf("exit code = %d, want 2", exitCode)
		}
		if !strings.Contains(stderr, "gf-find:") {
			t.Errorf("expected error message in stderr, got %q", stderr)
		}
	})
}

// Integration tests for -size filter
func TestFindSizeFilter(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()

	// Create files of different sizes
	small := filepath.Join(root, "small.txt")
	os.WriteFile(small, []byte("hi"), 0644) // 2 bytes

	medium := filepath.Join(root, "medium.txt")
	os.WriteFile(medium, make([]byte, 1024), 0644) // 1024 bytes

	large := filepath.Join(root, "large.txt")
	os.WriteFile(large, make([]byte, 10240), 0644) // 10240 bytes

	t.Run("size greater than 500c", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "f", "-size", "+500c", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		want := []string{large, medium}
		sort.Strings(want)
		if len(got) != len(want) {
			t.Errorf("got %d files, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
			return
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("file[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("size less than 100c", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "f", "-size", "-100c", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		if len(got) != 1 || got[0] != small {
			t.Errorf("got %v, want [%s]", got, small)
		}
	})

	t.Run("size exact 1k", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "f", "-size", "1k", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		if len(got) != 1 || got[0] != medium {
			t.Errorf("got %v, want [%s]", got, medium)
		}
	})

	t.Run("size greater than 5k", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "f", "-size", "+5k", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		if len(got) != 1 || got[0] != large {
			t.Errorf("got %v, want [%s]", got, large)
		}
	})

	t.Run("invalid size expression", func(t *testing.T) {
		_, stderr, exitCode := runFind(t, bin, "-size", "+abck", root)
		if exitCode != 2 {
			t.Errorf("exit code = %d, want 2", exitCode)
		}
		if !strings.Contains(stderr, "gf-find:") {
			t.Errorf("expected error in stderr, got %q", stderr)
		}
	})
}

// Integration tests for -mtime filter
func TestFindMtimeFilter(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()

	// Create files with different modification times
	recent := filepath.Join(root, "recent.txt")
	os.WriteFile(recent, []byte("new"), 0644) // just created = 0 days old

	old := filepath.Join(root, "old.txt")
	os.WriteFile(old, []byte("old"), 0644)
	// Set old.txt to 10 days ago
	tenDaysAgo := time.Now().Add(-10 * 24 * time.Hour)
	os.Chtimes(old, tenDaysAgo, tenDaysAgo)

	veryOld := filepath.Join(root, "veryold.txt")
	os.WriteFile(veryOld, []byte("very old"), 0644)
	// Set veryold.txt to 60 days ago
	sixtyDaysAgo := time.Now().Add(-60 * 24 * time.Hour)
	os.Chtimes(veryOld, sixtyDaysAgo, sixtyDaysAgo)

	t.Run("mtime -5 (modified less than 5 days ago)", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "f", "-mtime", "-5", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		if len(got) != 1 || got[0] != recent {
			t.Errorf("got %v, want [%s]", got, recent)
		}
	})

	t.Run("mtime +7 (modified more than 7 days ago)", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "f", "-mtime", "+7", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		want := []string{old, veryOld}
		sort.Strings(want)
		if len(got) != len(want) {
			t.Errorf("got %d files, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
			return
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("file[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("mtime +30 (modified more than 30 days ago)", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "f", "-mtime", "+30", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		if len(got) != 1 || got[0] != veryOld {
			t.Errorf("got %v, want [%s]", got, veryOld)
		}
	})

	t.Run("mtime 0 (modified today)", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "f", "-mtime", "0", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		if len(got) != 1 || got[0] != recent {
			t.Errorf("got %v, want [%s]", got, recent)
		}
	})

	t.Run("invalid mtime expression", func(t *testing.T) {
		_, stderr, exitCode := runFind(t, bin, "-mtime", "abc", root)
		if exitCode != 2 {
			t.Errorf("exit code = %d, want 2", exitCode)
		}
		if !strings.Contains(stderr, "gf-find:") {
			t.Errorf("expected error in stderr, got %q", stderr)
		}
	})
}

// Edge case: combined filters
func TestFindCombinedFilters(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()

	// Create structure with varied file types and sizes
	sub := filepath.Join(root, "src")
	os.MkdirAll(sub, 0755)

	small := filepath.Join(root, "small.go")
	os.WriteFile(small, []byte("package main"), 0644) // 12 bytes

	big := filepath.Join(root, "big.txt")
	os.WriteFile(big, make([]byte, 2048), 0644)

	subFile := filepath.Join(sub, "code.go")
	os.WriteFile(subFile, make([]byte, 500), 0644)

	t.Run("type f + name *.go + size +100c", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "f", "-name", "*.go", "-size", "+100c", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		if len(got) != 1 || got[0] != subFile {
			t.Errorf("got %v, want [%s]", got, subFile)
		}
	})

	t.Run("type d only", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-type", "d", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		want := []string{root, sub}
		sort.Strings(want)
		if len(got) != len(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

// Edge case: empty result with filters
func TestFindNoMatchWithFilter(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0644)

	// type d but only files exist (plus root dir)
	stdout, _, exitCode := runFind(t, bin, "-type", "f", "-name", "*.go", root)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	got := strings.TrimSpace(stdout)
	if got != "" {
		t.Errorf("expected empty output, got %q", got)
	}
}

// Unit tests for matchPath
func TestMatchPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pattern string
		want    bool
	}{
		{"empty pattern matches anything", "/foo/bar.txt", "", true},
		{"exact path match", "foo/bar.txt", "foo/bar.txt", true},
		{"glob star in filename", "foo/bar.txt", "foo/*.txt", true},
		{"glob star no match", "foo/bar.go", "foo/*.txt", false},
		{"glob question mark", "foo/bar1.txt", "foo/bar?.txt", true},
		{"nested path with glob", "src/sub/file.go", "src/*/*.go", true},
		{"no match different dir", "other/file.go", "src/*.go", false},
		{"multibyte path", "src/ファイル.txt", "src/*.txt", true},
		{"invalid pattern", "foo/bar.txt", "[", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPath(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("matchPath(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

// Unit tests for executeCmd
func TestExecuteCmd(t *testing.T) {
	// Test that confirmation prompt is shown and "n" skips execution
	oldPromptReader := promptReader
	defer func() { promptReader = oldPromptReader }()

	t.Run("decline execution", func(t *testing.T) {
		promptReader = bufio.NewReader(strings.NewReader("n\n"))
		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		executeCmd("/tmp/test.txt", "echo {}")

		w.Close()
		os.Stderr = oldStderr
		var buf [1024]byte
		n, _ := r.Read(buf[:])
		stderr := string(buf[:n])
		if !strings.Contains(stderr, "< echo /tmp/test.txt >?") {
			t.Errorf("expected prompt in stderr, got %q", stderr)
		}
	})

	t.Run("accept execution", func(t *testing.T) {
		promptReader = bufio.NewReader(strings.NewReader("y\n"))
		// Capture stdout and stderr
		oldStdout := os.Stdout
		oldStderr := os.Stderr
		rOut, wOut, _ := os.Pipe()
		rErr, wErr, _ := os.Pipe()
		os.Stdout = wOut
		os.Stderr = wErr

		executeCmd("hello_world", "echo {}")

		wOut.Close()
		wErr.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr

		var outBuf [1024]byte
		n, _ := rOut.Read(outBuf[:])
		stdout := string(outBuf[:n])
		rErr.Read(outBuf[:]) // drain stderr

		if !strings.Contains(stdout, "hello_world") {
			t.Errorf("expected echo output, got %q", stdout)
		}
	})

	t.Run("yes also accepted", func(t *testing.T) {
		promptReader = bufio.NewReader(strings.NewReader("yes\n"))
		oldStdout := os.Stdout
		oldStderr := os.Stderr
		_, wOut, _ := os.Pipe()
		_, wErr, _ := os.Pipe()
		os.Stdout = wOut
		os.Stderr = wErr

		executeCmd("test_file", "echo {}")

		wOut.Close()
		wErr.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})
}

// Integration tests for -path filter
func TestFindPathFilter(t *testing.T) {
	bin := buildBinary(t)
	root := createTestTree(t)

	t.Run("path glob matching subdirectory", func(t *testing.T) {
		// Match files under sub/deep/
		stdout, _, exitCode := runFind(t, bin, "-path", filepath.Join(root, "sub", "deep", "*.json"), root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		want := []string{filepath.Join(root, "sub", "deep", "file5.json")}
		if len(got) != len(want) {
			t.Errorf("got %d files, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
			return
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("file[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("path with name combined", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-path", filepath.Join(root, "sub", "*"), "-name", "*.go", root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := sortedLines(stdout)
		want := []string{filepath.Join(root, "sub", "file4.go")}
		if len(got) != len(want) {
			t.Errorf("got %d files, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
		}
	})

	t.Run("path no match", func(t *testing.T) {
		stdout, _, exitCode := runFind(t, bin, "-path", filepath.Join(root, "nonexist", "*"), root)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		got := strings.TrimSpace(stdout)
		if got != "" {
			t.Errorf("expected empty output, got %q", got)
		}
	})
}

// Integration tests for -exec option
func TestFindExec(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()

	f1 := filepath.Join(root, "a.txt")
	f2 := filepath.Join(root, "b.txt")
	os.WriteFile(f1, []byte("hello"), 0644)
	os.WriteFile(f2, []byte("world"), 0644)

	t.Run("exec with yes confirmation", func(t *testing.T) {
		cmd := exec.Command(bin, "-type", "f", "-name", "a.txt", "-exec", "echo {}", root)
		cmd.Stdin = strings.NewReader("y\n")
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout.String(), f1) {
			t.Errorf("expected echo output with file path, got stdout=%q", stdout.String())
		}
		if !strings.Contains(stderr.String(), "< echo "+f1+" >?") {
			t.Errorf("expected prompt in stderr, got %q", stderr.String())
		}
	})

	t.Run("exec with no confirmation", func(t *testing.T) {
		cmd := exec.Command(bin, "-type", "f", "-name", "a.txt", "-exec", "echo {}", root)
		cmd.Stdin = strings.NewReader("n\n")
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Should not have executed echo
		if strings.Contains(stdout.String(), f1) {
			t.Errorf("expected no output when declined, got stdout=%q", stdout.String())
		}
	})

	t.Run("exec multiple files with all yes", func(t *testing.T) {
		cmd := exec.Command(bin, "-type", "f", "-exec", "echo {}", root)
		// Provide enough y responses for all files
		cmd.Stdin = strings.NewReader("y\ny\ny\ny\ny\n")
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Both files should appear in output
		out := stdout.String()
		if !strings.Contains(out, "a.txt") || !strings.Contains(out, "b.txt") {
			t.Errorf("expected both files in output, got stdout=%q stderr=%q", out, stderr.String())
		}
	})

	t.Run("exec with invalid command", func(t *testing.T) {
		cmd := exec.Command(bin, "-type", "f", "-name", "a.txt", "-exec", "nonexistent_cmd_12345 {}", root)
		cmd.Stdin = strings.NewReader("y\n")
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		cmd.Run() // may fail, that's ok
		if !strings.Contains(stderr.String(), "gf-find:") {
			t.Errorf("expected error in stderr, got %q", stderr.String())
		}
	})

	t.Run("exec with multibyte filename", func(t *testing.T) {
		mbFile := filepath.Join(root, "日本語.txt")
		os.WriteFile(mbFile, []byte("テスト"), 0644)
		cmd := exec.Command(bin, "-type", "f", "-name", "日本語.txt", "-exec", "echo {}", root)
		cmd.Stdin = strings.NewReader("y\n")
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout.String(), "日本語.txt") {
			t.Errorf("expected multibyte filename in output, got %q", stdout.String())
		}
	})
}

// Edge case: -exec with empty match (no prompt should appear)
func TestFindExecNoMatch(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0644)

	cmd := exec.Command(bin, "-type", "f", "-name", "*.go", "-exec", "echo {}", root)
	cmd.Stdin = strings.NewReader("")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if stdout.String() != "" {
		t.Errorf("expected empty stdout, got %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("expected empty stderr, got %q", stderr.String())
	}
}

// Edge case: -path with multibyte characters
func TestFindPathMultibyte(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()
	subDir := filepath.Join(root, "データ")
	os.MkdirAll(subDir, 0755)
	f := filepath.Join(subDir, "ファイル.txt")
	os.WriteFile(f, []byte("テスト"), 0644)

	stdout, _, exitCode := runFind(t, bin, "-path", filepath.Join(root, "データ", "*.txt"), root)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	got := strings.TrimSpace(stdout)
	if got != f {
		t.Errorf("got %q, want %q", got, f)
	}
}
