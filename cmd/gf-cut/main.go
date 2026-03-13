package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const version = "0.1.0"

type cutOptions struct {
	delimiter string
	fields    string
}

type fieldRange struct {
	start int // 1-based, inclusive
	end   int // 1-based, inclusive; -1 means to end of line
}

func main() {
	showVersion := flag.Bool("version", false, "show version")
	optD := flag.String("d", "\t", "use DELIM instead of TAB for field delimiter")
	optF := flag.String("f", "", "select only these fields (e.g. 1,3 or 1-3 or 2-)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-cut [OPTIONS] [FILE...]\n")
		fmt.Fprintf(os.Stderr, "Remove sections from each line of files.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("gf-cut version %s\n", version)
		os.Exit(0)
	}

	if *optF == "" {
		fmt.Fprintf(os.Stderr, "gf-cut: you must specify a list of fields\n")
		os.Exit(2)
	}

	ranges, err := parseFields(*optF)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gf-cut: %v\n", err)
		os.Exit(2)
	}

	opts := cutOptions{
		delimiter: *optD,
		fields:    *optF,
	}

	args := flag.Args()
	exitCode := run(args, os.Stdin, os.Stdout, os.Stderr, opts, ranges)
	os.Exit(exitCode)
}

func parseFields(spec string) ([]fieldRange, error) {
	var ranges []fieldRange
	parts := strings.Split(spec, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("invalid field specification: empty field")
		}
		if strings.Contains(p, "-") {
			dashIdx := strings.Index(p, "-")
			startStr := p[:dashIdx]
			endStr := p[dashIdx+1:]

			if startStr == "" && endStr == "" {
				return nil, fmt.Errorf("invalid field range: '%s'", p)
			}

			start := 1
			end := -1
			var err error

			if startStr != "" {
				start, err = strconv.Atoi(startStr)
				if err != nil || start < 1 {
					return nil, fmt.Errorf("invalid field number: '%s'", startStr)
				}
			}
			if endStr != "" {
				end, err = strconv.Atoi(endStr)
				if err != nil || end < 1 {
					return nil, fmt.Errorf("invalid field number: '%s'", endStr)
				}
			}

			if end != -1 && start > end {
				return nil, fmt.Errorf("invalid decreasing range: '%s'", p)
			}

			ranges = append(ranges, fieldRange{start: start, end: end})
		} else {
			n, err := strconv.Atoi(p)
			if err != nil || n < 1 {
				return nil, fmt.Errorf("invalid field number: '%s'", p)
			}
			ranges = append(ranges, fieldRange{start: n, end: n})
		}
	}
	return ranges, nil
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, opts cutOptions, ranges []fieldRange) int {
	w := bufio.NewWriter(stdout)
	defer w.Flush()

	if len(args) == 0 {
		processReader(stdin, w, opts, ranges)
		return 0
	}

	hasError := false
	for _, arg := range args {
		if arg == "-" {
			processReader(stdin, w, opts, ranges)
			continue
		}
		f, err := os.Open(arg)
		if err != nil {
			fmt.Fprintf(stderr, "gf-cut: %s: %v\n", arg, unwrapPathError(err))
			hasError = true
			continue
		}
		processReader(f, w, opts, ranges)
		f.Close()
	}

	if hasError {
		return 1
	}
	return 0
}

func processReader(r io.Reader, w *bufio.Writer, opts cutOptions, ranges []fieldRange) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, opts.delimiter)
		selected := selectFields(fields, ranges)
		fmt.Fprintln(w, strings.Join(selected, opts.delimiter))
	}
}

func selectFields(fields []string, ranges []fieldRange) []string {
	var result []string
	for _, r := range ranges {
		start := r.start
		end := r.end

		if end == -1 {
			end = len(fields)
		}

		for i := start; i <= end; i++ {
			if i >= 1 && i <= len(fields) {
				result = append(result, fields[i-1])
			}
		}
	}
	return result
}

func unwrapPathError(err error) error {
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err
	}
	return err
}
