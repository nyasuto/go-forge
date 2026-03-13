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

type uniqOptions struct {
	count      bool
	duplicates bool
	ignoreCase bool
	global     bool
}

func main() {
	showVersion := flag.Bool("version", false, "show version")
	optC := flag.Bool("c", false, "prefix lines by the number of occurrences")
	optD := flag.Bool("d", false, "only print duplicate lines")
	optI := flag.Bool("i", false, "ignore differences in case when comparing")
	optGlobal := flag.Bool("global", false, "remove non-adjacent duplicates too")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-uniq [OPTIONS] [FILE...]\n")
		fmt.Fprintf(os.Stderr, "Filter adjacent matching lines.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("gf-uniq version %s\n", version)
		os.Exit(0)
	}

	opts := uniqOptions{
		count:      *optC,
		duplicates: *optD,
		ignoreCase: *optI,
		global:     *optGlobal,
	}

	args := flag.Args()
	exitCode := run(args, os.Stdin, os.Stdout, os.Stderr, opts)
	os.Exit(exitCode)
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, opts uniqOptions) int {
	w := bufio.NewWriter(stdout)
	defer w.Flush()

	if len(args) == 0 {
		processReader(stdin, w, opts)
		return 0
	}

	hasError := false
	for _, arg := range args {
		if arg == "-" {
			processReader(stdin, w, opts)
			continue
		}
		f, err := os.Open(arg)
		if err != nil {
			fmt.Fprintf(stderr, "gf-uniq: %s: %v\n", arg, unwrapPathError(err))
			hasError = true
			continue
		}
		processReader(f, w, opts)
		f.Close()
	}

	if hasError {
		return 1
	}
	return 0
}

func compareLine(a, b string, ignoreCase bool) bool {
	if ignoreCase {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func processReader(r io.Reader, w *bufio.Writer, opts uniqOptions) {
	if opts.global {
		processReaderGlobal(r, w, opts)
		return
	}

	scanner := bufio.NewScanner(r)
	prev := ""
	first := true
	count := 0

	flush := func() {
		if first {
			return
		}
		if opts.duplicates && count < 2 {
			return
		}
		if opts.count {
			fmt.Fprintf(w, "%7d %s\n", count, prev)
		} else {
			fmt.Fprintln(w, prev)
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if first || !compareLine(line, prev, opts.ignoreCase) {
			flush()
			prev = line
			count = 1
			first = false
		} else {
			count++
		}
	}
	flush()
}

func processReaderGlobal(r io.Reader, w *bufio.Writer, opts uniqOptions) {
	scanner := bufio.NewScanner(r)

	type entry struct {
		line  string
		count int
		order int
	}

	seen := make(map[string]*entry)
	var keys []string
	idx := 0

	for scanner.Scan() {
		line := scanner.Text()
		key := line
		if opts.ignoreCase {
			key = strings.ToLower(line)
		}

		if e, ok := seen[key]; ok {
			e.count++
		} else {
			seen[key] = &entry{line: line, count: 1, order: idx}
			keys = append(keys, key)
			idx++
		}
	}

	for _, key := range keys {
		e := seen[key]
		if opts.duplicates && e.count < 2 {
			continue
		}
		if opts.count {
			fmt.Fprintf(w, "%7d %s\n", e.count, e.line)
		} else {
			fmt.Fprintln(w, e.line)
		}
	}
}

func unwrapPathError(err error) error {
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err
	}
	return err
}
