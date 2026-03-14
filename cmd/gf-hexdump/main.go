package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorNull    = "\033[2m"      // dim for NULL bytes
	colorPrint   = "\033[32m"     // green for printable
	colorControl = "\033[31m"     // red for control characters
	colorHigh    = "\033[34m"     // blue for high bytes (0x80-0xff)
)

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

type hexdumpOptions struct {
	skip  int64
	limit int64
	color bool
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gf-hexdump", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "show version")
	skip := fs.Int64("s", 0, "skip offset bytes from the beginning")
	limit := fs.Int64("n", -1, "read only N bytes")
	colorMode := fs.String("color", "auto", "color output: auto|always|never")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintf(stdout, "gf-hexdump version %s\n", version)
		return 0
	}

	if *skip < 0 {
		fmt.Fprintf(stderr, "gf-hexdump: invalid skip value: %d\n", *skip)
		return 2
	}

	if *limit < -1 {
		fmt.Fprintf(stderr, "gf-hexdump: invalid length value: %d\n", *limit)
		return 2
	}

	useColor := false
	switch *colorMode {
	case "always":
		useColor = true
	case "never":
		useColor = false
	case "auto":
		if f, ok := stdout.(*os.File); ok {
			info, err := f.Stat()
			if err == nil && (info.Mode()&os.ModeCharDevice) != 0 {
				useColor = true
			}
		}
	default:
		fmt.Fprintf(stderr, "gf-hexdump: invalid color mode: %s\n", *colorMode)
		return 2
	}

	opts := hexdumpOptions{skip: *skip, limit: *limit, color: useColor}
	files := fs.Args()

	if len(files) == 0 || (len(files) == 1 && files[0] == "-") {
		return hexdump(stdin, stdout, stderr, opts)
	}

	exitCode := 0
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(stderr, "gf-hexdump: %s: %v\n", path, err)
			exitCode = 1
			continue
		}
		if code := hexdump(f, stdout, stderr, opts); code != 0 && exitCode == 0 {
			exitCode = code
		}
		f.Close()
	}
	return exitCode
}

func hexdump(r io.Reader, stdout, stderr io.Writer, opts hexdumpOptions) int {
	// Skip bytes if -s specified
	if opts.skip > 0 {
		if seeker, ok := r.(io.Seeker); ok {
			if _, err := seeker.Seek(opts.skip, io.SeekStart); err != nil {
				fmt.Fprintf(stderr, "gf-hexdump: seek error: %v\n", err)
				return 1
			}
		} else {
			if _, err := io.CopyN(io.Discard, r, opts.skip); err != nil {
				if err == io.EOF {
					return 0
				}
				fmt.Fprintf(stderr, "gf-hexdump: skip error: %v\n", err)
				return 1
			}
		}
	}

	// Limit bytes if -n specified
	var reader io.Reader = r
	if opts.limit >= 0 {
		reader = io.LimitReader(r, opts.limit)
	}

	offset := int(opts.skip)
	buf := make([]byte, 16)

	for {
		n, err := io.ReadFull(reader, buf)
		if n > 0 {
			formatLine(stdout, offset, buf[:n], opts.color)
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

func byteColor(b byte) string {
	switch {
	case b == 0x00:
		return colorNull
	case b >= 0x20 && b <= 0x7e:
		return colorPrint
	case b >= 0x80:
		return colorHigh
	default:
		return colorControl
	}
}

func formatLine(w io.Writer, offset int, data []byte, color bool) {
	// Offset
	fmt.Fprintf(w, "%08x  ", offset)

	// Hex bytes
	for i := 0; i < 16; i++ {
		if i == 8 {
			fmt.Fprint(w, " ")
		}
		if i < len(data) {
			if color {
				fmt.Fprintf(w, "%s%02x%s ", byteColor(data[i]), data[i], colorReset)
			} else {
				fmt.Fprintf(w, "%02x ", data[i])
			}
		} else {
			fmt.Fprint(w, "   ")
		}
	}

	// ASCII
	fmt.Fprint(w, " |")
	for i := 0; i < len(data); i++ {
		if data[i] >= 0x20 && data[i] <= 0x7e {
			if color {
				fmt.Fprintf(w, "%s%c%s", colorPrint, data[i], colorReset)
			} else {
				fmt.Fprintf(w, "%c", data[i])
			}
		} else {
			if color {
				fmt.Fprintf(w, "%s.%s", byteColor(data[i]), colorReset)
			} else {
				fmt.Fprint(w, ".")
			}
		}
	}
	fmt.Fprintln(w, "|")
}
