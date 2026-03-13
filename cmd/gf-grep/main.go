package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "バージョンを表示")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-grep [OPTIONS] PATTERN [FILE]...\n\nパターンにマッチする行を表示する。\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("gf-grep version " + version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "gf-grep: パターンが指定されていません\n")
		os.Exit(2)
	}

	pattern := args[0]
	files := args[1:]

	exitCode := 1 // 1 = no match found

	if len(files) == 0 {
		if grep(os.Stdin, os.Stdout, pattern, "") {
			exitCode = 0
		}
		os.Exit(exitCode)
	}

	showFilename := len(files) > 1

	for _, name := range files {
		if name == "-" {
			if grep(os.Stdin, os.Stdout, pattern, stdinPrefix(showFilename)) {
				exitCode = 0
			}
			continue
		}

		f, err := os.Open(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gf-grep: %v\n", err)
			if exitCode != 0 {
				exitCode = 2
			}
			continue
		}

		prefix := ""
		if showFilename {
			prefix = name + ":"
		}
		if grep(f, os.Stdout, pattern, prefix) {
			exitCode = 0
		}
		f.Close()
	}

	os.Exit(exitCode)
}

func stdinPrefix(show bool) string {
	if show {
		return "(standard input):"
	}
	return ""
}

// grep searches for pattern in r and writes matching lines to w.
// Returns true if at least one match was found.
func grep(r io.Reader, w io.Writer, pattern, prefix string) bool {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, pattern) {
			found = true
			if prefix != "" {
				fmt.Fprintf(bw, "%s%s\n", prefix, line)
			} else {
				fmt.Fprintf(bw, "%s\n", line)
			}
		}
	}
	return found
}
