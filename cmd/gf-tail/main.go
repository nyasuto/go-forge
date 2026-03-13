package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

const version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "バージョンを表示")
	numLines := flag.Int("n", 10, "表示する行数")
	follow := flag.Bool("f", false, "ファイル追記を監視して表示し続ける")
	pattern := flag.String("p", "", "マッチ行をハイライト表示するパターン（正規表現）")
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

	var re *regexp.Regexp
	if *pattern != "" {
		var err error
		re, err = regexp.Compile(*pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gf-tail: invalid pattern: %v\n", err)
			os.Exit(2)
		}
	}

	args := flag.Args()

	// -f はファイル引数が必要（stdinやハイフンは不可）
	if *follow {
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "gf-tail: -f requires a file argument\n")
			os.Exit(2)
		}
		for _, a := range args {
			if a == "-" {
				fmt.Fprintf(os.Stderr, "gf-tail: -f cannot be used with stdin\n")
				os.Exit(2)
			}
		}
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "gf-tail: -f supports only one file\n")
			os.Exit(2)
		}
	}

	exitCode := 0

	if len(args) == 0 {
		if err := tail(os.Stdin, os.Stdout, *numLines, re); err != nil {
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

		if err := tail(r, os.Stdout, *numLines, re); err != nil {
			fmt.Fprintf(os.Stderr, "gf-tail: %v\n", err)
			exitCode = 1
		}
	}

	if *follow && exitCode == 0 {
		followFile(args[0], os.Stdout, re)
		// followFile runs forever (or until error)
	}

	os.Exit(exitCode)
}

// highlightLine applies ANSI highlight to matching portions of a line.
// If re is nil or doesn't match, the line is returned unchanged.
func highlightLine(line string, re *regexp.Regexp) string {
	if re == nil {
		return line
	}
	locs := re.FindAllStringIndex(line, -1)
	if locs == nil {
		return line
	}
	const (
		red   = "\033[1;31m"
		reset = "\033[0m"
	)
	var b strings.Builder
	last := 0
	for _, loc := range locs {
		b.WriteString(line[last:loc[0]])
		b.WriteString(red)
		b.WriteString(line[loc[0]:loc[1]])
		b.WriteString(reset)
		last = loc[1]
	}
	b.WriteString(line[last:])
	return b.String()
}

func tail(r io.Reader, w io.Writer, n int, re *regexp.Regexp) error {
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
		fmt.Fprintln(bw, highlightLine(ring[idx], re))
	}

	return bw.Flush()
}

// followFile polls the file for new data and writes it to w.
// It runs until an error occurs or the process is interrupted.
func followFile(path string, w io.Writer, re *regexp.Regexp) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gf-tail: %v\n", err)
		return
	}
	defer f.Close()

	// Seek to end of file
	offset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gf-tail: %v\n", err)
		return
	}

	bw := bufio.NewWriter(w)

	if re == nil {
		// No pattern: stream raw bytes as before
		buf := make([]byte, 4096)
		for {
			time.Sleep(100 * time.Millisecond)

			info, err := os.Stat(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gf-tail: %v\n", err)
				return
			}

			newSize := info.Size()
			if newSize == offset {
				continue
			}

			if newSize < offset {
				offset = 0
				f.Seek(0, io.SeekStart)
			}

			for offset < newSize {
				n, err := f.Read(buf)
				if n > 0 {
					bw.Write(buf[:n])
					bw.Flush()
					offset += int64(n)
				}
				if err != nil {
					break
				}
			}
		}
	} else {
		// With pattern: read line by line so we can highlight
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for {
			time.Sleep(100 * time.Millisecond)

			info, err := os.Stat(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gf-tail: %v\n", err)
				return
			}

			newSize := info.Size()
			if newSize == offset {
				continue
			}

			if newSize < offset {
				offset = 0
				f.Seek(0, io.SeekStart)
				scanner = bufio.NewScanner(f)
				scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
			}

			for scanner.Scan() {
				line := scanner.Text()
				fmt.Fprintln(bw, highlightLine(line, re))
				bw.Flush()
			}

			// Update offset to current position
			pos, err := f.Seek(0, io.SeekCurrent)
			if err == nil {
				offset = pos
			}
		}
	}
}
