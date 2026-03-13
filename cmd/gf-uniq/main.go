package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

const version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "show version")
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

	args := flag.Args()
	exitCode := run(args, os.Stdin, os.Stdout, os.Stderr)
	os.Exit(exitCode)
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	w := bufio.NewWriter(stdout)
	defer w.Flush()

	if len(args) == 0 {
		processReader(stdin, w)
		return 0
	}

	hasError := false
	for _, arg := range args {
		if arg == "-" {
			processReader(stdin, w)
			continue
		}
		f, err := os.Open(arg)
		if err != nil {
			fmt.Fprintf(stderr, "gf-uniq: %s: %v\n", arg, unwrapPathError(err))
			hasError = true
			continue
		}
		processReader(f, w)
		f.Close()
	}

	if hasError {
		return 1
	}
	return 0
}

func processReader(r io.Reader, w *bufio.Writer) {
	scanner := bufio.NewScanner(r)
	prev := ""
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first || line != prev {
			fmt.Fprintln(w, line)
			prev = line
			first = false
		}
	}
}

func unwrapPathError(err error) error {
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err
	}
	return err
}
