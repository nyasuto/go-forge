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
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-cat [OPTIONS] [FILE]...\n\nファイルを連結して標準出力に表示する。\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("gf-cat version " + version)
		os.Exit(0)
	}

	args := flag.Args()
	exitCode := 0

	if len(args) == 0 {
		if err := cat(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "gf-cat: %v\n", err)
			os.Exit(1)
		}
		return
	}

	for _, arg := range args {
		if arg == "-" {
			if err := cat(os.Stdin, os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "gf-cat: %v\n", err)
				exitCode = 1
			}
			continue
		}

		f, err := os.Open(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gf-cat: %v\n", err)
			exitCode = 1
			continue
		}
		if err := cat(f, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "gf-cat: %v\n", err)
			exitCode = 1
		}
		f.Close()
	}

	os.Exit(exitCode)
}

func cat(r io.Reader, w io.Writer) error {
	bw := bufio.NewWriter(w)
	_, err := io.Copy(bw, r)
	if err != nil {
		return err
	}
	return bw.Flush()
}
