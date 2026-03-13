package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

const version = "0.1.0"

type counts struct {
	lines int
	words int
	bytes int
	chars int
}

func main() {
	showVersion := flag.Bool("version", false, "バージョンを表示")
	countLines := flag.Bool("l", false, "行数のみ表示")
	countWords := flag.Bool("w", false, "単語数のみ表示")
	countBytes := flag.Bool("c", false, "バイト数のみ表示")
	countChars := flag.Bool("m", false, "文字数のみ表示")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-wc [OPTIONS] [FILE]...\n\n行数・単語数・バイト数をカウントする。\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("gf-wc version " + version)
		os.Exit(0)
	}

	// フラグ指定なし→全表示
	showAll := !*countLines && !*countWords && !*countBytes && !*countChars

	args := flag.Args()
	exitCode := 0

	if len(args) == 0 {
		c, err := wc(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gf-wc: %v\n", err)
			os.Exit(1)
		}
		printCounts(c, "", showAll, *countLines, *countWords, *countBytes, *countChars)
		return
	}

	var total counts
	for _, arg := range args {
		var r io.Reader
		var name string

		if arg == "-" {
			r = os.Stdin
			name = ""
		} else {
			f, err := os.Open(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "gf-wc: %v\n", err)
				exitCode = 1
				continue
			}
			r = f
			name = arg
			defer f.Close()
		}

		c, err := wc(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gf-wc: %v\n", err)
			exitCode = 1
			continue
		}
		printCounts(c, name, showAll, *countLines, *countWords, *countBytes, *countChars)
		total.lines += c.lines
		total.words += c.words
		total.bytes += c.bytes
		total.chars += c.chars
	}

	if len(args) > 1 {
		printCounts(total, "total", showAll, *countLines, *countWords, *countBytes, *countChars)
	}

	os.Exit(exitCode)
}

func wc(r io.Reader) (counts, error) {
	var c counts
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		c.lines++
		c.bytes += len(line) + 1 // +1 for newline
		c.chars += utf8.RuneCount(line) + 1 // +1 for newline
		c.words += countWords(line)
	}
	if err := scanner.Err(); err != nil {
		return c, err
	}
	return c, nil
}

func countWords(line []byte) int {
	count := 0
	inWord := false
	for _, b := range line {
		if b == ' ' || b == '\t' || b == '\r' || b == '\v' || b == '\f' {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
	return count
}

func printCounts(c counts, name string, showAll, showLines, showWords, showBytes, showChars bool) {
	parts := []string{}
	if showAll || showLines {
		parts = append(parts, fmt.Sprintf("%8d", c.lines))
	}
	if showAll || showWords {
		parts = append(parts, fmt.Sprintf("%8d", c.words))
	}
	if showAll || showBytes {
		parts = append(parts, fmt.Sprintf("%8d", c.bytes))
	}
	if showChars {
		parts = append(parts, fmt.Sprintf("%8d", c.chars))
	}

	for i, p := range parts {
		if i > 0 {
			fmt.Print("")
		}
		fmt.Print(p)
	}
	if name != "" {
		fmt.Printf(" %s", name)
	}
	fmt.Println()
}
