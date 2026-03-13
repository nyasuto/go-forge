package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const version = "0.1.0"

type sedCommand struct {
	pattern *regexp.Regexp
	replace string
}

func main() {
	showVersion := flag.Bool("version", false, "show version")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-sed EXPRESSION [FILE...]\n")
		fmt.Fprintf(os.Stderr, "Stream editor for filtering and transforming text.\n\n")
		fmt.Fprintf(os.Stderr, "Supported expressions:\n")
		fmt.Fprintf(os.Stderr, "  s/PATTERN/REPLACEMENT/   Replace first match of PATTERN with REPLACEMENT\n\n")
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
	exitCode := run(files, os.Stdin, os.Stdout, os.Stderr, cmd)
	os.Exit(exitCode)
}

func parseExpression(expr string) (*sedCommand, error) {
	if !strings.HasPrefix(expr, "s") {
		return nil, fmt.Errorf("unknown command: '%s'", expr)
	}
	if len(expr) < 2 {
		return nil, fmt.Errorf("invalid expression: '%s'", expr)
	}

	delim := expr[1]
	rest := expr[2:]

	// Split by delimiter, handling escaped delimiters
	parts := splitByDelim(rest, delim)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid substitution expression: '%s'", expr)
	}

	pattern := parts[0]
	replace := parts[1]

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %v", err)
	}

	return &sedCommand{
		pattern: re,
		replace: replace,
	}, nil
}

// splitByDelim splits a string by a delimiter byte, respecting backslash escapes.
// Returns the parts (without the delimiter). The trailing delimiter is optional.
func splitByDelim(s string, delim byte) []string {
	var parts []string
	var current strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			if s[i+1] == delim {
				current.WriteByte(delim)
				i += 2
				continue
			}
			current.WriteByte(s[i])
			current.WriteByte(s[i+1])
			i += 2
			continue
		}
		if s[i] == delim {
			parts = append(parts, current.String())
			current.Reset()
			i++
			continue
		}
		current.WriteByte(s[i])
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

func processReader(r io.Reader, w *bufio.Writer, cmd *sedCommand) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		result := applySubstitution(line, cmd)
		fmt.Fprintln(w, result)
	}
}

// applySubstitution replaces the first match of the pattern in line.
func applySubstitution(line string, cmd *sedCommand) string {
	loc := cmd.pattern.FindStringIndex(line)
	if loc == nil {
		return line
	}
	// Use ReplaceAllString on just the first match
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
