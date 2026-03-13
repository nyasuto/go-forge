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
	showVersion := flag.Bool("version", false, "バージョンを表示")
	numLines := flag.Int("n", 10, "表示する行数")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-tail [OPTIONS] [FILE]...\n\nファイルの末尾部分を表示する。\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("gf-tail version " + version)
		os.Exit(0)
	}

	if *numLines < 0 {
		fmt.Fprintf(os.Stderr, "gf-tail: invalid number of lines: '%d'\n", *numLines)
		os.Exit(2)
	}

	args := flag.Args()
	exitCode := 0

	if len(args) == 0 {
		if err := tail(os.Stdin, os.Stdout, *numLines); err != nil {
			fmt.Fprintf(os.Stderr, "gf-tail: %v\n", err)
			os.Exit(1)
		}
		return
	}

	for i, arg := range args {
		var r io.Reader
		if arg == "-" {
			r = os.Stdin
		} else {
			f, err := os.Open(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gf-tail: %v\n", err)
				exitCode = 1
				continue
			}
			defer f.Close()
			r = f
		}

		if len(args) > 1 {
			if i > 0 {
				fmt.Fprintln(os.Stdout)
			}
			fmt.Fprintf(os.Stdout, "==> %s <==\n", arg)
		}

		if err := tail(r, os.Stdout, *numLines); err != nil {
			fmt.Fprintf(os.Stderr, "gf-tail: %v\n", err)
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

func tail(r io.Reader, w io.Writer, n int) error {
	if n == 0 {
		return nil
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// リングバッファで末尾n行を保持
	ring := make([]string, n)
	count := 0

	for scanner.Scan() {
		ring[count%n] = scanner.Text()
		count++
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	bw := bufio.NewWriter(w)

	total := count
	if total > n {
		total = n
	}

	start := 0
	if count > n {
		start = count % n
	}

	for i := 0; i < total; i++ {
		idx := (start + i) % n
		fmt.Fprintln(bw, ring[idx])
	}

	return bw.Flush()
}
