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

			stats, err := walkDir(dir, "")

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

		_, err := printTree(dir, "")

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
		_, err := printTree("/nonexistent/path/xyz", "")
		if err == nil {
			t.Error("expected error for non-existent path")
		}
	})

	t.Run("error on file path", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "file.txt")
		os.WriteFile(f, []byte("x"), 0644)

		_, err := printTree(f, "")
		if err == nil {
			t.Error("expected error for file path")
		}
	})
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
