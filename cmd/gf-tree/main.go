package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "バージョンを表示")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-tree [OPTIONS] [DIRECTORY]...\n\nディレクトリツリーを再帰的に表示する。\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("gf-tree version " + version)
		os.Exit(0)
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
		stats, err := printTree(dir, "")
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

func printTree(root string, prefix string) (treeStats, error) {
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

	return walkDir(root, prefix)
}

func walkDir(dir string, prefix string) (treeStats, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return treeStats{}, err
	}

	// Sort entries: directories first, then files, both alphabetically
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

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
			childStats, err := walkDir(filepath.Join(dir, entry.Name()), prefix+childPrefix)
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
