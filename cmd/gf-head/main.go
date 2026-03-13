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
		fmt.Fprintf(os.Stderr, "Usage: gf-head [OPTIONS] [FILE]...\n\nファイルの先頭部分を表示する。\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("gf-head version " + version)
		os.Exit(0)
	}

	if *numLines < 0 {
		fmt.Fprintf(os.Stderr, "gf-head: invalid number of lines: '%d'\n", *numLines)
		os.Exit(2)
	}

	args := flag.Args()
	exitCode := 0

	if len(args) == 0 {
		if err := head(os.Stdin, os.Stdout, *numLines); err != nil {
			fmt.Fprintf(os.Stderr, "gf-head: %v\n", err)
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
				fmt.Fprintf(os.Stderr, "gf-head: %v\n", err)
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

		if err := head(r, os.Stdout, *numLines); err != nil {
			fmt.Fprintf(os.Stderr, "gf-head: %v\n", err)
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

func head(r io.Reader, w io.Writer, n int) error {
	if n == 0 {
		return nil
	}

	bw := bufio.NewWriter(w)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	count := 0
	for scanner.Scan() {
		fmt.Fprintln(bw, scanner.Text())
		count++
		if count >= n {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return bw.Flush()
}
