package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

const version = "0.1.0"

type sortOptions struct {
	numeric   bool
	reverse   bool
	unique    bool
	keyField  int    // 1-based, 0 means no key
	delimiter string // field delimiter (empty means whitespace)
}

func main() {
	showVersion := flag.Bool("version", false, "show version")
	numeric := flag.Bool("n", false, "numeric sort")
	reverse := flag.Bool("r", false, "reverse sort order")
	unique := flag.Bool("u", false, "output only unique lines")
	keyField := flag.Int("k", 0, "sort by field number (1-based)")
	delimiter := flag.String("t", "", "field delimiter (default: whitespace)")
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

	if *keyField < 0 {
		fmt.Fprintf(os.Stderr, "gf-sort: invalid key field: %d\n", *keyField)
		os.Exit(2)
	}

	opts := sortOptions{
		numeric:   *numeric,
		reverse:   *reverse,
		unique:    *unique,
		keyField:  *keyField,
		delimiter: *delimiter,
	}

	args := flag.Args()

	lines, err := readLines(args)
	if err != nil {
		os.Exit(1)
	}

	sortLines(lines, opts)

	if opts.unique {
		lines = dedup(lines)
	}

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
}

// extractKey returns the sort key for a line based on options.
func extractKey(line string, keyField int, delimiter string) string {
	if keyField <= 0 {
		return line
	}
	var fields []string
	if delimiter == "" {
		fields = strings.Fields(line)
	} else {
		fields = strings.Split(line, delimiter)
	}
	if keyField > len(fields) {
		return ""
	}
	return fields[keyField-1]
}

// parseNumber extracts a leading numeric value from a string for numeric sort.
func parseNumber(s string) float64 {
	s = strings.TrimSpace(s)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func sortLines(lines []string, opts sortOptions) {
	sort.SliceStable(lines, func(i, j int) bool {
		ki := extractKey(lines[i], opts.keyField, opts.delimiter)
		kj := extractKey(lines[j], opts.keyField, opts.delimiter)

		var less bool
		if opts.numeric {
			ni := parseNumber(ki)
			nj := parseNumber(kj)
			less = ni < nj
		} else {
			less = ki < kj
		}

		if opts.reverse {
			return !less
		}
		return less
	})
}

// dedup removes consecutive duplicate lines (assumes sorted input).
func dedup(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	result := []string{lines[0]}
	for i := 1; i < len(lines); i++ {
		if lines[i] != lines[i-1] {
			result = append(result, lines[i])
		}
	}
	return result
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
