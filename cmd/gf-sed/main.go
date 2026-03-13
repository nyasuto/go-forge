package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const version = "0.1.0"

// addressType represents how a line address is specified.
type addressType int

const (
	addrNone    addressType = iota
	addrLine                // specific line number
	addrLast                // $ (last line)
	addrPattern             // /regex/ pattern match
)

type address struct {
	typ     addressType
	line    int
	pattern *regexp.Regexp
}

type sedCommand struct {
	pattern *regexp.Regexp
	replace string
	global  bool
	addr    *address
}

func main() {
	showVersion := flag.Bool("version", false, "show version")
	inPlace := flag.Bool("i", false, "edit files in place")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-sed [OPTIONS] EXPRESSION [FILE...]\n")
		fmt.Fprintf(os.Stderr, "Stream editor for filtering and transforming text.\n\n")
		fmt.Fprintf(os.Stderr, "Supported expressions:\n")
		fmt.Fprintf(os.Stderr, "  s/PATTERN/REPLACEMENT/[g]   Replace match of PATTERN with REPLACEMENT\n")
		fmt.Fprintf(os.Stderr, "  Ns/PATTERN/REPLACEMENT/     Apply only to line N\n")
		fmt.Fprintf(os.Stderr, "  $s/PATTERN/REPLACEMENT/     Apply only to last line\n")
		fmt.Fprintf(os.Stderr, "  /PAT/s/PATTERN/REPLACEMENT/ Apply only to lines matching PAT\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("gf-sed version %s\n", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "gf-sed: no expression specified\n")
		os.Exit(2)
	}

	expr := args[0]
	cmd, err := parseExpression(expr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gf-sed: %v\n", err)
		os.Exit(2)
	}

	files := args[1:]

	if *inPlace {
		if len(files) == 0 {
			fmt.Fprintf(os.Stderr, "gf-sed: -i requires file arguments\n")
			os.Exit(2)
		}
		exitCode := runInPlace(files, os.Stderr, cmd)
		os.Exit(exitCode)
	}

	exitCode := run(files, os.Stdin, os.Stdout, os.Stderr, cmd)
	os.Exit(exitCode)
}

func parseExpression(expr string) (*sedCommand, error) {
	addr, rest, err := parseAddress(expr)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(rest, "s") {
		return nil, fmt.Errorf("unknown command: '%s'", expr)
	}
	afterS := rest[1:]
	if len(afterS) == 0 {
		return nil, fmt.Errorf("invalid expression: '%s'", expr)
	}

	delim, delimSize := utf8.DecodeRuneInString(afterS)
	if delim == utf8.RuneError {
		return nil, fmt.Errorf("invalid delimiter in expression: '%s'", expr)
	}
	body := afterS[delimSize:]

	// Split by delimiter, handling escaped delimiters (rune-aware)
	parts := splitByDelim(body, delim)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid substitution expression: '%s'", expr)
	}

	pattern := parts[0]
	replace := parts[1]

	// Parse flags (part after the closing delimiter)
	globalFlag := false
	if len(parts) >= 3 {
		flags := parts[2]
		for _, ch := range flags {
			switch ch {
			case 'g':
				globalFlag = true
			default:
				return nil, fmt.Errorf("unknown flag '%c' in expression: '%s'", ch, expr)
			}
		}
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %v", err)
	}

	return &sedCommand{
		pattern: re,
		replace: replace,
		global:  globalFlag,
		addr:    addr,
	}, nil
}

// parseAddress extracts an optional address prefix from the expression.
// Returns the address (nil if none), the remaining expression string, and any error.
func parseAddress(expr string) (*address, string, error) {
	if len(expr) == 0 {
		return nil, expr, nil
	}

	firstRune, _ := utf8.DecodeRuneInString(expr)

	// /pattern/ address
	if firstRune == '/' {
		end := findClosingSlash(expr, 1)
		if end < 0 {
			return nil, "", fmt.Errorf("unterminated address regex: '%s'", expr)
		}
		pat := expr[1:end]
		re, err := regexp.Compile(pat)
		if err != nil {
			return nil, "", fmt.Errorf("invalid address regex: %v", err)
		}
		return &address{typ: addrPattern, pattern: re}, expr[end+1:], nil
	}

	// $ address
	if firstRune == '$' {
		return &address{typ: addrLast}, expr[1:], nil
	}

	// Line number address (digits are always single-byte ASCII)
	i := 0
	for i < len(expr) && expr[i] >= '0' && expr[i] <= '9' {
		i++
	}
	if i > 0 {
		n, err := strconv.Atoi(expr[:i])
		if err != nil {
			return nil, "", fmt.Errorf("invalid line number: '%s'", expr[:i])
		}
		return &address{typ: addrLine, line: n}, expr[i:], nil
	}

	return nil, expr, nil
}

// findClosingSlash finds the byte index of the closing '/' in an address pattern,
// handling backslash-escaped slashes. Processes rune-by-rune for multibyte safety.
// Returns -1 if not found.
func findClosingSlash(s string, start int) int {
	i := start
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\\' {
			i += size
			if i < len(s) {
				_, size2 := utf8.DecodeRuneInString(s[i:])
				i += size2
			}
			continue
		}
		if r == '/' {
			return i
		}
		i += size
	}
	return -1
}

// splitByDelim splits a string by a delimiter rune, respecting backslash escapes.
// Returns the parts (without the delimiter). The trailing delimiter is optional.
// Processes input rune-by-rune for multibyte safety.
func splitByDelim(s string, delim rune) []string {
	var parts []string
	var current strings.Builder
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		if runes[i] == '\\' && i+1 < len(runes) {
			if runes[i+1] == delim {
				current.WriteRune(delim)
				i += 2
				continue
			}
			current.WriteRune(runes[i])
			current.WriteRune(runes[i+1])
			i += 2
			continue
		}
		if runes[i] == delim {
			parts = append(parts, current.String())
			current.Reset()
			i++
			continue
		}
		current.WriteRune(runes[i])
		i++
	}
	parts = append(parts, current.String())
	return parts
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, cmd *sedCommand) int {
	w := bufio.NewWriter(stdout)
	defer w.Flush()

	if len(args) == 0 {
		processReader(stdin, w, cmd)
		return 0
	}

	hasError := false
	for _, arg := range args {
		if arg == "-" {
			processReader(stdin, w, cmd)
			continue
		}
		f, err := os.Open(arg)
		if err != nil {
			fmt.Fprintf(stderr, "gf-sed: %s: %v\n", arg, unwrapPathError(err))
			hasError = true
			continue
		}
		processReader(f, w, cmd)
		f.Close()
	}

	if hasError {
		return 1
	}
	return 0
}

func runInPlace(files []string, stderr io.Writer, cmd *sedCommand) int {
	hasError := false
	for _, path := range files {
		if path == "-" {
			fmt.Fprintf(stderr, "gf-sed: -i cannot be used with stdin\n")
			hasError = true
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "gf-sed: %s: %v\n", path, unwrapPathError(err))
			hasError = true
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(stderr, "gf-sed: %s: %v\n", path, unwrapPathError(err))
			hasError = true
			continue
		}

		var buf bytes.Buffer
		w := bufio.NewWriter(&buf)
		processReader(bytes.NewReader(data), w, cmd)
		w.Flush()

		if err := os.WriteFile(path, buf.Bytes(), info.Mode()); err != nil {
			fmt.Fprintf(stderr, "gf-sed: %s: %v\n", path, unwrapPathError(err))
			hasError = true
			continue
		}
	}

	if hasError {
		return 1
	}
	return 0
}

func processReader(r io.Reader, w *bufio.Writer, cmd *sedCommand) {
	if cmd.addr != nil && cmd.addr.typ == addrLast {
		processReaderLastLine(r, w, cmd)
		return
	}

	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if matchAddress(cmd.addr, line, lineNum) {
			line = applySubstitution(line, cmd)
		}
		fmt.Fprintln(w, line)
	}
}

// processReaderLastLine handles $ address by reading all lines first.
func processReaderLastLine(r io.Reader, w *bufio.Writer, cmd *sedCommand) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	for i, line := range lines {
		if i == len(lines)-1 {
			line = applySubstitution(line, cmd)
		}
		fmt.Fprintln(w, line)
	}
}

// matchAddress checks whether the command should apply to this line.
func matchAddress(addr *address, line string, lineNum int) bool {
	if addr == nil {
		return true
	}
	switch addr.typ {
	case addrNone:
		return true
	case addrLine:
		return lineNum == addr.line
	case addrPattern:
		return addr.pattern.MatchString(line)
	case addrLast:
		// Handled separately in processReaderLastLine
		return false
	}
	return true
}

// applySubstitution replaces match(es) of the pattern in line.
func applySubstitution(line string, cmd *sedCommand) string {
	if cmd.global {
		return cmd.pattern.ReplaceAllString(line, cmd.replace)
	}
	// First match only
	loc := cmd.pattern.FindStringIndex(line)
	if loc == nil {
		return line
	}
	match := line[loc[0]:loc[1]]
	replaced := cmd.pattern.ReplaceAllString(match, cmd.replace)
	return line[:loc[0]] + replaced + line[loc[1]:]
}

func unwrapPathError(err error) error {
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err
	}
	return err
}
