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
	numLines := flag.Int("n", 10, "表示する行数")
	numBytes := flag.Int("c", 0, "表示するバイト数")
	streaming := flag.Bool("F", false, "ストリーミングモード（N行ごとにクリア＆再表示）")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-head [OPTIONS] [FILE]...\n\nファイルの先頭部分を表示する。\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("gf-head version " + version)
		os.Exit(0)
	}

	if *streaming {
		if *numBytes > 0 {
			fmt.Fprintf(os.Stderr, "gf-head: -F cannot be used with -c\n")
			os.Exit(2)
		}
		if len(flag.Args()) > 0 {
			fmt.Fprintf(os.Stderr, "gf-head: -F reads from stdin only\n")
			os.Exit(2)
		}
		if err := headStreaming(os.Stdin, os.Stdout, *numLines); err != nil {
			fmt.Fprintf(os.Stderr, "gf-head: %v\n", err)
			os.Exit(1)
		}
		return
	}

	byteMode := *numBytes > 0
	if !byteMode && *numLines < 0 {
		fmt.Fprintf(os.Stderr, "gf-head: invalid number of lines: '%d'\n", *numLines)
		os.Exit(2)
	}
	if *numBytes < 0 {
		fmt.Fprintf(os.Stderr, "gf-head: invalid number of bytes: '%d'\n", *numBytes)
		os.Exit(2)
	}

	args := flag.Args()
	exitCode := 0

	process := func(r io.Reader, w io.Writer) error {
		if byteMode {
			return headBytes(r, w, *numBytes)
		}
		return head(r, w, *numLines)
	}

	if len(args) == 0 {
		if err := process(os.Stdin, os.Stdout); err != nil {
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

		if err := process(r, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "gf-head: %v\n", err)
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

func headStreaming(r io.Reader, w io.Writer, n int) error {
	if n <= 0 {
		return nil
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	batch := make([]string, 0, n)
	first := true

	for scanner.Scan() {
		batch = append(batch, scanner.Text())
		if len(batch) >= n {
			if !first {
				// ANSI: clear screen + cursor home
				fmt.Fprint(w, "\033[2J\033[H")
			}
			first = false
			for _, line := range batch {
				fmt.Fprintln(w, line)
			}
			batch = batch[:0]
		}
	}

	// 残りの行があれば表示
	if len(batch) > 0 {
		if !first {
			fmt.Fprint(w, "\033[2J\033[H")
		}
		fmt.Fprint(w, strings.Join(batch, "\n")+"\n")
	}

	return scanner.Err()
}

func headBytes(r io.Reader, w io.Writer, n int) error {
	if n == 0 {
		return nil
	}
	_, err := io.CopyN(w, r, int64(n))
	if err == io.EOF {
		return nil
	}
	return err
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
