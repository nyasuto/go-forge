package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "バージョンを表示")
	namePattern := flag.String("name", "", "ファイル名パターン（glob形式）")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-find [OPTIONS] [PATH]...\n\n再帰的にファイルを検索する。\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("gf-find version " + version)
		os.Exit(0)
	}

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	exitCode := 0
	for _, root := range paths {
		if err := find(root, *namePattern); err != nil {
			fmt.Fprintf(os.Stderr, "gf-find: %v\n", err)
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

func find(root, namePattern string) error {
	info, err := os.Lstat(root)
	if err != nil {
		return err
	}

	// If root is not a directory, check if it matches and print
	if !info.IsDir() {
		if matchName(filepath.Base(root), namePattern) {
			fmt.Println(root)
		}
		return nil
	}

	return filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "gf-find: %v\n", err)
			return nil
		}
		if matchName(fi.Name(), namePattern) {
			fmt.Println(path)
		}
		return nil
	})
}

func matchName(name, pattern string) bool {
	if pattern == "" {
		return true
	}
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return false
	}
	return matched
}
