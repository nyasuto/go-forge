package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("gf-tee", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "バージョンを表示")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "gf-tee version "+version)
		return 0
	}

	filePaths := fs.Args()

	files, exitCode := openFiles(filePaths, stderr)
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

	if _, err := io.Copy(mw, stdin); err != nil {
		fmt.Fprintf(stderr, "gf-tee: %v\n", err)
		return 1
	}

	return 0
}

func openFiles(paths []string, stderr io.Writer) ([]*os.File, int) {
	files := make([]*os.File, 0, len(paths))
	for _, path := range paths {
		f, err := os.Create(path)
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
