package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

const version = "0.1.0"

type options struct {
	number    bool
	squeeze   bool
	colorMode string // "auto", "always", "never"
	lang      string // detected language extension
	lineNum   int
	lastBlank bool
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func main() {
	showVersion := flag.Bool("version", false, "バージョンを表示")
	numberLines := flag.Bool("n", false, "行番号を表示")
	squeezeBlank := flag.Bool("s", false, "連続する空行を1行に圧縮")
	colorFlag := flag.String("color", "auto", "シンタックスハイライト (auto/always/never)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-cat [OPTIONS] [FILE]...\n\nファイルを連結して標準出力に表示する。\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("gf-cat version " + version)
		os.Exit(0)
	}

	opts := &options{
		number:    *numberLines,
		squeeze:   *squeezeBlank,
		colorMode: *colorFlag,
	}

	args := flag.Args()
	exitCode := 0

	if len(args) == 0 {
		if err := cat(os.Stdin, os.Stdout, opts); err != nil {
			fmt.Fprintf(os.Stderr, "gf-cat: %v\n", err)
			os.Exit(1)
		}
		return
	}

	for _, arg := range args {
		if arg == "-" {
			opts.lang = ""
			if err := cat(os.Stdin, os.Stdout, opts); err != nil {
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
		opts.lang = detectLanguage(arg)
		if err := cat(f, os.Stdout, opts); err != nil {
			fmt.Fprintf(os.Stderr, "gf-cat: %v\n", err)
			exitCode = 1
		}
		f.Close()
	}

	os.Exit(exitCode)
}

func shouldHighlight(opts *options) bool {
	switch opts.colorMode {
	case "always":
		return opts.lang != ""
	case "never":
		return false
	default: // "auto"
		return opts.lang != "" && isTerminal(os.Stdout)
	}
}

func cat(r io.Reader, w io.Writer, opts *options) error {
	highlight := shouldHighlight(opts)

	if !opts.number && !opts.squeeze && !highlight {
		bw := bufio.NewWriter(w)
		_, err := io.Copy(bw, r)
		if err != nil {
			return err
		}
		return bw.Flush()
	}

	bw := bufio.NewWriter(w)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		isBlank := line == ""

		if opts.squeeze && isBlank && opts.lastBlank {
			continue
		}
		opts.lastBlank = isBlank

		if highlight {
			line = highlightLine(line, opts.lang)
		}

		opts.lineNum++
		if opts.number {
			fmt.Fprintf(bw, "%6d\t%s\n", opts.lineNum, line)
		} else {
			fmt.Fprintf(bw, "%s\n", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return bw.Flush()
}
