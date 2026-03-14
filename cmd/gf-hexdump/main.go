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

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gf-hexdump", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "show version")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintf(stdout, "gf-hexdump version %s\n", version)
		return 0
	}

	files := fs.Args()

	if len(files) == 0 || (len(files) == 1 && files[0] == "-") {
		return hexdump(stdin, stdout, stderr)
	}

	exitCode := 0
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(stderr, "gf-hexdump: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		if code := hexdump(f, stdout, stderr); code != 0 && exitCode == 0 {
			exitCode = code
		}
		f.Close()
	}
	return exitCode
}

func hexdump(r io.Reader, stdout, stderr io.Writer) int {
	offset := 0
	buf := make([]byte, 16)

	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			formatLine(stdout, offset, buf[:n])
			offset += n
		}
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			fmt.Fprintf(stderr, "gf-hexdump: read error: %v\n", err)
			return 1
		}
	}
	return 0
}

func formatLine(w io.Writer, offset int, data []byte) {
	// Offset
	fmt.Fprintf(w, "%08x  ", offset)

	// Hex bytes
	for i := 0; i < 16; i++ {
		if i == 8 {
			fmt.Fprint(w, " ")
		}
		if i < len(data) {
			fmt.Fprintf(w, "%02x ", data[i])
		} else {
			fmt.Fprint(w, "   ")
		}
	}

	// ASCII
	fmt.Fprint(w, " |")
	for i := 0; i < len(data); i++ {
		if data[i] >= 0x20 && data[i] <= 0x7e {
			fmt.Fprintf(w, "%c", data[i])
		} else {
			fmt.Fprint(w, ".")
		}
	}
	fmt.Fprintln(w, "|")
}
