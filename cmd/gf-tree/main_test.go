package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary builds the gf-tree binary for integration tests
func buildBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "gf-tree")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build: %s\n%s", err, out)
	}
	return binary
}

// createTestTree creates a temporary directory structure for testing
func createTestTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// Create directories
	os.MkdirAll(filepath.Join(root, "dir1", "subdir1"), 0755)
	os.MkdirAll(filepath.Join(root, "dir2"), 0755)

	// Create files
	os.WriteFile(filepath.Join(root, "file1.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(root, "file2.txt"), []byte("world"), 0644)
	os.WriteFile(filepath.Join(root, "dir1", "a.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(root, "dir1", "subdir1", "deep.txt"), []byte("deep"), 0644)
	os.WriteFile(filepath.Join(root, "dir2", "b.txt"), []byte("data"), 0644)

	return root
}

// --- Unit tests for walkDir ---

func TestWalkDir(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		wantDirs int
		wantFiles int
		wantLines []string // substrings that must appear in output
	}{
		{
			name: "simple directory with files",
			setup: func(t *testing.T) string {
				root := t.TempDir()
				os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0644)
				os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0644)
				return root
			},
			wantDirs:  0,
			wantFiles: 2,
			wantLines: []string{"├── a.txt", "└── b.txt"},
		},
		{
			name: "nested directories",
			setup: func(t *testing.T) string {
				root := t.TempDir()
				os.MkdirAll(filepath.Join(root, "sub", "deep"), 0755)
				os.WriteFile(filepath.Join(root, "sub", "deep", "file.txt"), []byte("x"), 0644)
				return root
			},
			wantDirs:  2,
			wantFiles: 1,
			wantLines: []string{"└── sub", "    └── deep", "        └── file.txt"},
		},
		{
			name: "empty directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantDirs:  0,
			wantFiles: 0,
		},
		{
			name: "directory with mixed content",
			setup: func(t *testing.T) string {
				root := t.TempDir()
				os.MkdirAll(filepath.Join(root, "alpha"), 0755)
				os.WriteFile(filepath.Join(root, "beta.txt"), []byte("b"), 0644)
				os.WriteFile(filepath.Join(root, "alpha", "inside.txt"), []byte("i"), 0644)
				return root
			},
			wantDirs:  1,
			wantFiles: 2,
			wantLines: []string{"├── alpha", "│   └── inside.txt", "└── beta.txt"},
		},
		{
			name: "multibyte filenames",
			setup: func(t *testing.T) string {
				root := t.TempDir()
				os.WriteFile(filepath.Join(root, "日本語.txt"), []byte("jp"), 0644)
				os.WriteFile(filepath.Join(root, "🎄.md"), []byte("tree"), 0644)
				return root
			},
			wantDirs:  0,
			wantFiles: 2,
			wantLines: []string{"日本語.txt", "🎄.md"},
		},
		{
			name: "alphabetical sorting",
			setup: func(t *testing.T) string {
				root := t.TempDir()
				os.WriteFile(filepath.Join(root, "charlie.txt"), []byte("c"), 0644)
				os.WriteFile(filepath.Join(root, "alpha.txt"), []byte("a"), 0644)
				os.WriteFile(filepath.Join(root, "bravo.txt"), []byte("b"), 0644)
				return root
			},
			wantDirs:  0,
			wantFiles: 3,
			wantLines: []string{"├── alpha.txt", "├── bravo.txt", "└── charlie.txt"},
		},
		{
			name: "deep nesting with connectors",
			setup: func(t *testing.T) string {
				root := t.TempDir()
				os.MkdirAll(filepath.Join(root, "a"), 0755)
				os.MkdirAll(filepath.Join(root, "b"), 0755)
				os.WriteFile(filepath.Join(root, "a", "x.txt"), []byte("x"), 0644)
				os.WriteFile(filepath.Join(root, "b", "y.txt"), []byte("y"), 0644)
				return root
			},
			wantDirs:  2,
			wantFiles: 2,
			wantLines: []string{
				"├── a",
				"│   └── x.txt",
				"└── b",
				"    └── y.txt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)

			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			stats, err := walkDir(dir, "", 1, treeOptions{})

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if stats.dirs != tt.wantDirs {
				t.Errorf("dirs = %d, want %d", stats.dirs, tt.wantDirs)
			}
			if stats.files != tt.wantFiles {
				t.Errorf("files = %d, want %d", stats.files, tt.wantFiles)
			}

			for _, line := range tt.wantLines {
				if !strings.Contains(output, line) {
					t.Errorf("output missing expected line %q\nfull output:\n%s", line, output)
				}
			}
		})
	}
}

// --- Unit tests for printTree ---

func TestPrintTree(t *testing.T) {
	t.Run("prints root directory name", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "test.txt"), []byte("t"), 0644)

		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		_, err := printTree(dir, "", treeOptions{})

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.HasPrefix(output, dir) {
			t.Errorf("output should start with root dir %q, got:\n%s", dir, output)
		}
	})

	t.Run("error on non-existent path", func(t *testing.T) {
		_, err := printTree("/nonexistent/path/xyz", "", treeOptions{})
		if err == nil {
			t.Error("expected error for non-existent path")
		}
	})

	t.Run("error on file path", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "file.txt")
		os.WriteFile(f, []byte("x"), 0644)

		_, err := printTree(f, "", treeOptions{})
		if err == nil {
			t.Error("expected error for file path")
		}
	})
}

// --- Unit tests for -L depth limit ---

func TestWalkDirWithDepthLimit(t *testing.T) {
	tests := []struct {
		name      string
		maxDepth  int
		wantDirs  int
		wantFiles int
		wantLines []string
		notWant   []string
	}{
		{
			name:      "depth 1 shows only top-level",
			maxDepth:  1,
			wantDirs:  2,
			wantFiles: 2,
			wantLines: []string{"├── dir1", "├── dir2", "├── file1.txt", "└── file2.txt"},
			notWant:   []string{"a.go", "subdir1", "deep.txt", "b.txt"},
		},
		{
			name:      "depth 2 shows two levels",
			maxDepth:  2,
			wantDirs:  3,
			wantFiles: 4,
			wantLines: []string{"├── dir1", "│   ├── a.go", "│   └── subdir1", "├── dir2", "│   └── b.txt"},
			notWant:   []string{"deep.txt"},
		},
		{
			name:      "depth 0 means unlimited",
			maxDepth:  0,
			wantDirs:  3,
			wantFiles: 5,
			wantLines: []string{"deep.txt"},
		},
		{
			name:      "depth exceeding tree depth shows all",
			maxDepth:  100,
			wantDirs:  3,
			wantFiles: 5,
			wantLines: []string{"deep.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := createTestTree(t)

			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			opts := treeOptions{maxDepth: tt.maxDepth}
			stats, err := walkDir(root, "", 1, opts)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if stats.dirs != tt.wantDirs {
				t.Errorf("dirs = %d, want %d", stats.dirs, tt.wantDirs)
			}
			if stats.files != tt.wantFiles {
				t.Errorf("files = %d, want %d", stats.files, tt.wantFiles)
			}

			for _, line := range tt.wantLines {
				if !strings.Contains(output, line) {
					t.Errorf("output missing expected %q\nfull output:\n%s", line, output)
				}
			}
			for _, line := range tt.notWant {
				if strings.Contains(output, line) {
					t.Errorf("output should NOT contain %q\nfull output:\n%s", line, output)
				}
			}
		})
	}
}

// --- Unit tests for -I exclude pattern ---

func TestWalkDirWithExclude(t *testing.T) {
	tests := []struct {
		name      string
		exclude   string
		wantDirs  int
		wantFiles int
		wantLines []string
		notWant   []string
	}{
		{
			name:      "exclude by extension",
			exclude:   "*.txt",
			wantDirs:  3,
			wantFiles: 1,
			wantLines: []string{"a.go"},
			notWant:   []string{"file1.txt", "file2.txt", "deep.txt", "b.txt"},
		},
		{
			name:      "exclude directory by name",
			exclude:   "dir1",
			wantDirs:  1,
			wantFiles: 3,
			wantLines: []string{"dir2", "file1.txt", "file2.txt", "b.txt"},
			notWant:   []string{"dir1", "a.go", "subdir1"},
		},
		{
			name:      "exclude with glob pattern",
			exclude:   "file*",
			wantDirs:  3,
			wantFiles: 3,
			wantLines: []string{"dir1", "dir2", "a.go", "deep.txt", "b.txt"},
			notWant:   []string{"file1.txt", "file2.txt"},
		},
		{
			name:      "no match excludes nothing",
			exclude:   "*.xyz",
			wantDirs:  3,
			wantFiles: 5,
		},
		{
			name:      "exclude applied at all levels",
			exclude:   "*.go",
			wantDirs:  3,
			wantFiles: 4,
			notWant:   []string{"a.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := createTestTree(t)

			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			opts := treeOptions{exclude: tt.exclude}
			stats, err := walkDir(root, "", 1, opts)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if stats.dirs != tt.wantDirs {
				t.Errorf("dirs = %d, want %d", stats.dirs, tt.wantDirs)
			}
			if stats.files != tt.wantFiles {
				t.Errorf("files = %d, want %d", stats.files, tt.wantFiles)
			}

			for _, line := range tt.wantLines {
				if !strings.Contains(output, line) {
					t.Errorf("output missing expected %q\nfull output:\n%s", line, output)
				}
			}
			for _, line := range tt.notWant {
				if strings.Contains(output, line) {
					t.Errorf("output should NOT contain %q\nfull output:\n%s", line, output)
				}
			}
		})
	}
}

// --- Unit test for isExcluded ---

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		name    string
		entry   string
		pattern string
		want    bool
	}{
		{"match extension", "test.txt", "*.txt", true},
		{"no match extension", "test.go", "*.txt", false},
		{"match prefix", "node_modules", "node*", true},
		{"empty pattern", "test.txt", "", false},
		{"exact match", "Makefile", "Makefile", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExcluded(tt.entry, tt.pattern)
			if got != tt.want {
				t.Errorf("isExcluded(%q, %q) = %v, want %v", tt.entry, tt.pattern, got, tt.want)
			}
		})
	}
}

// --- Unit tests for formatSize ---

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero bytes", 0, "    0"},
		{"small bytes", 42, "   42"},
		{"1023 bytes", 1023, " 1023"},
		{"1 KB", 1024, " 1.0K"},
		{"1.5 KB", 1536, " 1.5K"},
		{"1 MB", 1024 * 1024, " 1.0M"},
		{"2.5 MB", int64(2.5 * 1024 * 1024), " 2.5M"},
		{"1 GB", 1024 * 1024 * 1024, " 1.0G"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

// --- Unit tests for -s file size display ---

func TestWalkDirWithSize(t *testing.T) {
	tests := []struct {
		name      string
		opts      treeOptions
		wantLines []string
		notWant   []string
	}{
		{
			name: "show file sizes with -s",
			opts: treeOptions{showSize: true},
			wantLines: []string{
				"[    5]  file1.txt", // "hello" = 5 bytes
				"[    5]  file2.txt", // "world" = 5 bytes
				"[   12]  a.go",      // "package main" = 12 bytes
				"[    4]  deep.txt",  // "deep" = 4 bytes
				"[    4]  b.txt",     // "data" = 4 bytes
			},
		},
		{
			name:    "no sizes without -s",
			opts:    treeOptions{},
			notWant: []string{"["},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := createTestTree(t)

			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			_, err := walkDir(root, "", 1, tt.opts)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, line := range tt.wantLines {
				if !strings.Contains(output, line) {
					t.Errorf("output missing expected %q\nfull output:\n%s", line, output)
				}
			}
			for _, line := range tt.notWant {
				if strings.Contains(output, line) {
					t.Errorf("output should NOT contain %q\nfull output:\n%s", line, output)
				}
			}
		})
	}
}

// --- Unit tests for --du directory size ---

func TestWalkDirWithDu(t *testing.T) {
	tests := []struct {
		name      string
		opts      treeOptions
		wantLines []string
	}{
		{
			name: "du shows dir and file sizes",
			opts: treeOptions{du: true},
			wantLines: []string{
				"[   16]  dir1",  // a.go(12) + deep.txt(4) = 16
				"[    4]  dir2",  // b.txt(4) = 4
				"[    5]  file1.txt",
				"[    5]  file2.txt",
			},
		},
		{
			name: "du with depth limit",
			opts: treeOptions{du: true, maxDepth: 1},
			wantLines: []string{
				"[   16]  dir1",
				"[    4]  dir2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := createTestTree(t)

			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			_, err := walkDir(root, "", 1, tt.opts)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, line := range tt.wantLines {
				if !strings.Contains(output, line) {
					t.Errorf("output missing expected %q\nfull output:\n%s", line, output)
				}
			}
		})
	}
}

// --- Unit tests for calcDirSize ---

func TestCalcDirSize(t *testing.T) {
	root := createTestTree(t)
	opts := treeOptions{}

	// Total: hello(5) + world(5) + package main(12) + deep(4) + data(4) = 30
	total := calcDirSize(root, opts)
	if total != 30 {
		t.Errorf("calcDirSize = %d, want 30", total)
	}

	// dir1 only: a.go(12) + deep.txt(4) = 16
	dir1Size := calcDirSize(filepath.Join(root, "dir1"), opts)
	if dir1Size != 16 {
		t.Errorf("calcDirSize(dir1) = %d, want 16", dir1Size)
	}

	// With exclude
	optsExclude := treeOptions{exclude: "*.txt"}
	sizeNoTxt := calcDirSize(root, optsExclude)
	if sizeNoTxt != 12 { // only a.go
		t.Errorf("calcDirSize with exclude *.txt = %d, want 12", sizeNoTxt)
	}
}

// --- Integration tests ---

func TestIntegration(t *testing.T) {
	binary := buildBinary(t)

	t.Run("basic tree output", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary, root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		// Should contain root path
		if !strings.Contains(output, root) {
			t.Errorf("output should contain root path %q", root)
		}

		// Should contain tree connectors
		if !strings.Contains(output, "├── ") && !strings.Contains(output, "└── ") {
			t.Error("output should contain tree connectors")
		}

		// Should contain summary line
		if !strings.Contains(output, "directories") || !strings.Contains(output, "files") {
			t.Errorf("output should contain summary line, got:\n%s", output)
		}
	})

	t.Run("default to current directory", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		// Should start with "."
		if !strings.HasPrefix(output, ".") {
			t.Errorf("output should start with '.', got:\n%s", output)
		}
	})

	t.Run("multiple directories", func(t *testing.T) {
		root1 := t.TempDir()
		root2 := t.TempDir()
		os.WriteFile(filepath.Join(root1, "a.txt"), []byte("a"), 0644)
		os.WriteFile(filepath.Join(root2, "b.txt"), []byte("b"), 0644)

		cmd := exec.Command(binary, root1, root2)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		// Should contain both root paths
		if !strings.Contains(output, root1) || !strings.Contains(output, root2) {
			t.Errorf("output should contain both root paths, got:\n%s", output)
		}

		// Should have two summary lines
		count := strings.Count(output, "directories,")
		if count != 2 {
			t.Errorf("expected 2 summary lines, got %d\noutput:\n%s", count, output)
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		cmd := exec.Command(binary, "/nonexistent/xyz")
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err == nil {
			t.Error("expected non-zero exit code")
		}

		if !strings.Contains(output, "gf-tree") {
			t.Errorf("stderr should contain tool name, got:\n%s", output)
		}
	})

	t.Run("version flag", func(t *testing.T) {
		cmd := exec.Command(binary, "--version")
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(output, "gf-tree version 0.1.0") {
			t.Errorf("unexpected version output: %s", output)
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		root := t.TempDir()
		cmd := exec.Command(binary, root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		if !strings.Contains(output, "0 directories, 0 files") {
			t.Errorf("expected '0 directories, 0 files', got:\n%s", output)
		}
	})

	t.Run("correct directory and file counts", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary, root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		// createTestTree: dir1, dir1/subdir1, dir2 = 3 dirs
		// file1.txt, file2.txt, dir1/a.go, dir1/subdir1/deep.txt, dir2/b.txt = 5 files
		if !strings.Contains(output, "3 directories, 5 files") {
			t.Errorf("expected '3 directories, 5 files', got:\n%s", output)
		}
	})

	t.Run("-L depth limit", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary, "-L", "1", root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		if strings.Contains(output, "a.go") || strings.Contains(output, "subdir1") {
			t.Errorf("-L 1 should not show nested entries, got:\n%s", output)
		}
		if !strings.Contains(output, "dir1") || !strings.Contains(output, "file1.txt") {
			t.Errorf("-L 1 should show top-level entries, got:\n%s", output)
		}
	})

	t.Run("-L 2 depth limit", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary, "-L", "2", root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		if strings.Contains(output, "deep.txt") {
			t.Errorf("-L 2 should not show level 3 entries, got:\n%s", output)
		}
		if !strings.Contains(output, "a.go") || !strings.Contains(output, "subdir1") {
			t.Errorf("-L 2 should show level 2 entries, got:\n%s", output)
		}
	})

	t.Run("-I exclude pattern", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary, "-I", "*.txt", root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		if strings.Contains(output, ".txt") {
			t.Errorf("-I '*.txt' should exclude .txt files, got:\n%s", output)
		}
		if !strings.Contains(output, "a.go") {
			t.Errorf("-I '*.txt' should keep .go files, got:\n%s", output)
		}
	})

	t.Run("-I exclude directory", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary, "-I", "dir1", root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		if strings.Contains(output, "dir1") {
			t.Errorf("-I dir1 should exclude dir1 and its contents, got:\n%s", output)
		}
		if !strings.Contains(output, "dir2") {
			t.Errorf("-I dir1 should keep dir2, got:\n%s", output)
		}
	})

	t.Run("-L and -I combined", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary, "-L", "2", "-I", "*.txt", root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		if strings.Contains(output, ".txt") {
			t.Errorf("should exclude .txt files, got:\n%s", output)
		}
		if strings.Contains(output, "deep.txt") {
			t.Errorf("should not show level 3 entries, got:\n%s", output)
		}
		if !strings.Contains(output, "a.go") {
			t.Errorf("should show a.go at level 2, got:\n%s", output)
		}
	})

	t.Run("-L negative value exits with code 2", func(t *testing.T) {
		cmd := exec.Command(binary, "-L", "-1", ".")
		out, err := cmd.CombinedOutput()

		if err == nil {
			t.Error("expected non-zero exit code for negative -L")
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 2 {
				t.Errorf("expected exit code 2, got %d", exitErr.ExitCode())
			}
		}
		_ = out
	})

	t.Run("-s shows file sizes", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary, "-s", root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		// Files should have size brackets
		if !strings.Contains(output, "[    5]  file1.txt") {
			t.Errorf("-s should show file sizes, got:\n%s", output)
		}
		// Directories should NOT have size brackets with -s only
		if strings.Contains(output, "]  dir1") {
			t.Errorf("-s should not show directory sizes, got:\n%s", output)
		}
	})

	t.Run("--du shows directory and file sizes", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary, "--du", root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		// Root should have total size
		if !strings.Contains(output, "[   30]") {
			t.Errorf("--du should show root total size [   30], got:\n%s", output)
		}
		// Directories should have aggregated sizes
		if !strings.Contains(output, "[   16]  dir1") {
			t.Errorf("--du should show dir1 size [   16], got:\n%s", output)
		}
		if !strings.Contains(output, "[    4]  dir2") {
			t.Errorf("--du should show dir2 size [    4], got:\n%s", output)
		}
	})

	t.Run("--du with -L depth limit", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary, "--du", "-L", "1", root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		// Should still show correct aggregated sizes even with depth limit
		if !strings.Contains(output, "[   16]  dir1") {
			t.Errorf("--du -L 1 should aggregate dir1 size, got:\n%s", output)
		}
	})

	t.Run("--du with -I exclude", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary, "--du", "-I", "*.txt", root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		// Should exclude .txt files from size calculation
		if !strings.Contains(output, "[   12]  dir1") {
			t.Errorf("--du -I '*.txt' should show dir1 size as 12 (only a.go), got:\n%s", output)
		}
	})

	t.Run("-s with large file", func(t *testing.T) {
		root := t.TempDir()
		// Create a file larger than 1KB
		data := make([]byte, 2048)
		os.WriteFile(filepath.Join(root, "big.bin"), data, 0644)

		cmd := exec.Command(binary, "-s", root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		if !strings.Contains(output, " 2.0K") {
			t.Errorf("-s should show human-readable size for large file, got:\n%s", output)
		}
	})

	t.Run("tree structure ordering", func(t *testing.T) {
		root := createTestTree(t)
		cmd := exec.Command(binary, root)
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, output)
		}

		lines := strings.Split(output, "\n")
		// dir1 should come before dir2, which should come before file1.txt
		dir1Idx, dir2Idx, file1Idx := -1, -1, -1
		for i, line := range lines {
			if strings.Contains(line, "dir1") && !strings.Contains(line, "subdir1") {
				dir1Idx = i
			}
			if strings.Contains(line, "dir2") {
				dir2Idx = i
			}
			if strings.Contains(line, "file1.txt") {
				file1Idx = i
			}
		}

		if dir1Idx == -1 || dir2Idx == -1 || file1Idx == -1 {
			t.Fatalf("could not find expected entries in output:\n%s", output)
		}

		if !(dir1Idx < dir2Idx && dir2Idx < file1Idx) {
			t.Errorf("expected alphabetical order: dir1(%d) < dir2(%d) < file1.txt(%d)", dir1Idx, dir2Idx, file1Idx)
		}
	})
}
