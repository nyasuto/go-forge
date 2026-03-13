package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
)

const version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "show version")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-sort [OPTIONS] [FILE...]\n")
		fmt.Fprintf(os.Stderr, "Sort lines of text.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("gf-sort version %s\n", version)
		os.Exit(0)
	}

	args := flag.Args()

	lines, err := readLines(args)
	if err != nil {
		os.Exit(1)
	}

	sort.Strings(lines)

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
}

func readLines(args []string) ([]string, error) {
	if len(args) == 0 {
		return readLinesFrom(os.Stdin)
	}

	var allLines []string
	hasError := false

	for _, arg := range args {
		if arg == "-" {
			lines, err := readLinesFrom(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gf-sort: %v\n", err)
				hasError = true
				continue
			}
			allLines = append(allLines, lines...)
			continue
		}

		f, err := os.Open(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gf-sort: %s: %v\n", arg, unwrapPathError(err))
			hasError = true
			continue
		}
		lines, err := readLinesFrom(f)
		f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gf-sort: %s: %v\n", arg, err)
			hasError = true
			continue
		}
		allLines = append(allLines, lines...)
	}

	if hasError {
		return allLines, fmt.Errorf("read error")
	}
	return allLines, nil
}

func readLinesFrom(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return lines, err
	}
	return lines, nil
}

func unwrapPathError(err error) error {
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err
	}
	return err
}
