package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

const version = "0.1.0"

// nowFunc is overridden in tests for deterministic timestamps.
var nowFunc = time.Now

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("gf-tee", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "バージョンを表示")
	appendMode := fs.Bool("a", false, "ファイルに追記（appendモード）")
	timestamp := fs.Bool("ts", false, "各行にタイムスタンプを付与")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "gf-tee version "+version)
		return 0
	}

	filePaths := fs.Args()

	files, exitCode := openFiles(filePaths, *appendMode, stderr)
	defer closeFiles(files)
	if exitCode != 0 {
		return exitCode
	}

	writers := make([]io.Writer, 0, len(files)+1)
	writers = append(writers, stdout)
	for _, f := range files {
		writers = append(writers, f)
	}

	mw := io.MultiWriter(writers...)

	if *timestamp {
		return copyWithTimestamp(mw, stdin, stderr)
	}

	if _, err := io.Copy(mw, stdin); err != nil {
		fmt.Fprintf(stderr, "gf-tee: %v\n", err)
		return 1
	}

	return 0
}

func copyWithTimestamp(w io.Writer, r io.Reader, stderr io.Writer) int {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		ts := nowFunc().Format("2006-01-02T15:04:05.000Z07:00")
		line := fmt.Sprintf("[%s] %s\n", ts, scanner.Text())
		if _, err := io.WriteString(w, line); err != nil {
			fmt.Fprintf(stderr, "gf-tee: %v\n", err)
			return 1
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "gf-tee: %v\n", err)
		return 1
	}
	return 0
}

func openFiles(paths []string, appendMode bool, stderr io.Writer) ([]*os.File, int) {
	files := make([]*os.File, 0, len(paths))
	for _, path := range paths {
		var f *os.File
		var err error
		if appendMode {
			f, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			f, err = os.Create(path)
		}
		if err != nil {
			fmt.Fprintf(stderr, "gf-tee: %v\n", err)
			closeFiles(files)
			return nil, 1
		}
		files = append(files, f)
	}
	return files, 0
}

func closeFiles(files []*os.File) {
	for _, f := range files {
		f.Close()
	}
}
