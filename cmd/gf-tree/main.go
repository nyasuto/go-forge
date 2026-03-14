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
}

func main() {
	showVersion := flag.Bool("version", false, "バージョンを表示")
	maxDepth := flag.Int("L", 0, "ツリーの深さ制限")
	exclude := flag.String("I", "", "除外パターン（glob）")

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
		fmt.Println(root)
	}

	return walkDir(root, prefix, 1, opts)
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

		fmt.Printf("%s%s%s\n", prefix, connector, entry.Name())

		if entry.IsDir() {
			stats.dirs++
			if opts.maxDepth > 0 && depth >= opts.maxDepth {
				continue
			}
			childStats, err := walkDir(filepath.Join(dir, entry.Name()), prefix+childPrefix, depth+1, opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gf-tree: %s: %s\n", filepath.Join(dir, entry.Name()), err)
				continue
			}
			stats.dirs += childStats.dirs
			stats.files += childStats.files
		} else {
			stats.files++
		}
	}

	return stats, nil
}
