package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const version = "0.1.0"

type treeOptions struct {
	maxDepth int    // -L: max depth (0 = unlimited)
	exclude  string // -I: exclude pattern (glob)
	showSize bool   // -s: show file size
	du       bool   // --du: show directory size (aggregate)
}

func main() {
	showVersion := flag.Bool("version", false, "バージョンを表示")
	maxDepth := flag.Int("L", 0, "ツリーの深さ制限")
	exclude := flag.String("I", "", "除外パターン（glob）")
	showSize := flag.Bool("s", false, "ファイルサイズを表示")
	du := flag.Bool("du", false, "ディレクトリサイズを集計表示")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-tree [OPTIONS] [DIRECTORY]...\n\nディレクトリツリーを再帰的に表示する。\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("gf-tree version " + version)
		os.Exit(0)
	}

	if *maxDepth < 0 {
		fmt.Fprintf(os.Stderr, "gf-tree: invalid depth: %d\n", *maxDepth)
		os.Exit(2)
	}

	opts := treeOptions{
		maxDepth: *maxDepth,
		exclude:  *exclude,
		showSize: *showSize,
		du:       *du,
	}

	dirs := flag.Args()
	if len(dirs) == 0 {
		dirs = []string{"."}
	}

	exitCode := 0
	for i, dir := range dirs {
		if i > 0 {
			fmt.Println()
		}
		stats, err := printTree(dir, "", opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gf-tree: %s: %s\n", dir, err)
			exitCode = 1
			continue
		}
		fmt.Printf("\n%d directories, %d files\n", stats.dirs, stats.files)
	}
	os.Exit(exitCode)
}

type treeStats struct {
	dirs  int
	files int
	size  int64 // total size in bytes
}

func printTree(root string, prefix string, opts treeOptions) (treeStats, error) {
	info, err := os.Stat(root)
	if err != nil {
		return treeStats{}, err
	}
	if !info.IsDir() {
		return treeStats{}, fmt.Errorf("not a directory")
	}

	// Print root directory name when called at top level
	if prefix == "" {
		if opts.du {
			// Calculate total size first
			totalSize := calcDirSize(root, opts)
			fmt.Printf("[%s]  %s\n", formatSize(totalSize), root)
		} else {
			fmt.Println(root)
		}
	}

	return walkDir(root, prefix, 1, opts)
}

// formatSize formats bytes into human-readable size
func formatSize(bytes int64) string {
	const (
		kB = 1024
		mB = 1024 * kB
		gB = 1024 * mB
	)
	switch {
	case bytes >= gB:
		return fmt.Sprintf("%4.1fG", float64(bytes)/float64(gB))
	case bytes >= mB:
		return fmt.Sprintf("%4.1fM", float64(bytes)/float64(mB))
	case bytes >= kB:
		return fmt.Sprintf("%4.1fK", float64(bytes)/float64(kB))
	default:
		return fmt.Sprintf("%5d", bytes)
	}
}

// calcDirSize calculates total size of a directory recursively
func calcDirSize(dir string, opts treeOptions) int64 {
	var total int64
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		if isExcluded(entry.Name(), opts.exclude) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			total += calcDirSize(path, opts)
		} else {
			info, err := entry.Info()
			if err == nil {
				total += info.Size()
			}
		}
	}
	return total
}

func isExcluded(name string, pattern string) bool {
	if pattern == "" {
		return false
	}
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return false
	}
	return matched
}

func walkDir(dir string, prefix string, depth int, opts treeOptions) (treeStats, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return treeStats{}, err
	}

	// Sort entries alphabetically
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// Filter excluded entries
	if opts.exclude != "" {
		filtered := entries[:0]
		for _, entry := range entries {
			if !isExcluded(entry.Name(), opts.exclude) {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}

	stats := treeStats{}

	for i, entry := range entries {
		isLast := i == len(entries)-1

		connector := "├── "
		childPrefix := "│   "
		if isLast {
			connector = "└── "
			childPrefix = "    "
		}

		entryPath := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			stats.dirs++
			if opts.du {
				dirSize := calcDirSize(entryPath, opts)
				stats.size += dirSize
				fmt.Printf("%s%s[%s]  %s\n", prefix, connector, formatSize(dirSize), entry.Name())
			} else {
				fmt.Printf("%s%s%s\n", prefix, connector, entry.Name())
			}
			if opts.maxDepth > 0 && depth >= opts.maxDepth {
				continue
			}
			childStats, err := walkDir(entryPath, prefix+childPrefix, depth+1, opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gf-tree: %s: %s\n", entryPath, err)
				continue
			}
			stats.dirs += childStats.dirs
			stats.files += childStats.files
			if !opts.du {
				stats.size += childStats.size
			}
		} else {
			stats.files++
			info, err := entry.Info()
			var fileSize int64
			if err == nil {
				fileSize = info.Size()
			}
			stats.size += fileSize
			if opts.showSize || opts.du {
				fmt.Printf("%s%s[%s]  %s\n", prefix, connector, formatSize(fileSize), entry.Name())
			} else {
				fmt.Printf("%s%s%s\n", prefix, connector, entry.Name())
			}
		}
	}

	return stats, nil
}
