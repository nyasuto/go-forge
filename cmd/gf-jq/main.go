package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gf-jq", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "show version")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "gf-jq version "+version)
		return 0
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(stderr, "gf-jq: missing filter expression")
		return 2
	}

	filter := remaining[0]
	files := remaining[1:]

	tokens, err := parseFilter(filter)
	if err != nil {
		fmt.Fprintf(stderr, "gf-jq: %v\n", err)
		return 2
	}

	exitCode := 0

	if len(files) == 0 || (len(files) == 1 && files[0] == "-") {
		if code := processReader(stdin, tokens, stdout, stderr); code != 0 {
			exitCode = code
		}
	} else {
		for _, file := range files {
			f, err := os.Open(file)
			if err != nil {
				fmt.Fprintf(stderr, "gf-jq: %v\n", err)
				exitCode = 1
				continue
			}
			if code := processReader(f, tokens, stdout, stderr); code != 0 {
				exitCode = code
			}
			f.Close()
		}
	}

	return exitCode
}

// token types for filter expression
type tokenType int

const (
	tokenKey   tokenType = iota // .key
	tokenIndex                  // .[0]
	tokenDot                    // . (identity)
)

type token struct {
	typ   tokenType
	key   string
	index int
}

func parseFilter(filter string) ([]token, error) {
	if filter == "." {
		return []token{{typ: tokenDot}}, nil
	}

	if !strings.HasPrefix(filter, ".") {
		return nil, fmt.Errorf("invalid filter: %q (must start with '.')", filter)
	}

	var tokens []token
	s := filter[1:] // skip leading dot

	for len(s) > 0 {
		if s[0] == '[' {
			// array index: [N]
			end := strings.IndexByte(s, ']')
			if end == -1 {
				return nil, fmt.Errorf("invalid filter: unclosed bracket in %q", filter)
			}
			indexStr := s[1:end]
			idx, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %q", indexStr)
			}
			tokens = append(tokens, token{typ: tokenIndex, index: idx})
			s = s[end+1:]
			if len(s) > 0 && s[0] == '.' {
				s = s[1:]
			}
		} else {
			// key access
			end := indexOfAny(s, ".[")
			if end == -1 {
				end = len(s)
			}
			key := s[:end]
			if key == "" {
				return nil, fmt.Errorf("invalid filter: empty key in %q", filter)
			}
			tokens = append(tokens, token{typ: tokenKey, key: key})
			s = s[end:]
			if len(s) > 0 && s[0] == '.' {
				s = s[1:]
			}
		}
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("invalid filter: %q", filter)
	}

	return tokens, nil
}

func indexOfAny(s, chars string) int {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		for _, c := range chars {
			if r == c {
				return i
			}
		}
		i += size
	}
	return -1
}

func processReader(r io.Reader, tokens []token, stdout, stderr io.Writer) int {
	data, err := io.ReadAll(r)
	if err != nil {
		fmt.Fprintf(stderr, "gf-jq: read error: %v\n", err)
		return 1
	}

	var input any
	if err := json.Unmarshal(data, &input); err != nil {
		fmt.Fprintf(stderr, "gf-jq: invalid JSON: %v\n", err)
		return 1
	}

	result, err := applyFilter(input, tokens)
	if err != nil {
		fmt.Fprintf(stderr, "gf-jq: %v\n", err)
		return 1
	}

	outputJSON(result, stdout)
	return 0
}

func applyFilter(data any, tokens []token) (any, error) {
	if len(tokens) == 1 && tokens[0].typ == tokenDot {
		return data, nil
	}

	current := data
	for _, tok := range tokens {
		switch tok.typ {
		case tokenKey:
			obj, ok := current.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("cannot index %T with key %q", current, tok.key)
			}
			val, exists := obj[tok.key]
			if !exists {
				return nil, nil // null
			}
			current = val
		case tokenIndex:
			arr, ok := current.([]any)
			if !ok {
				return nil, fmt.Errorf("cannot index %T with number", current)
			}
			idx := tok.index
			if idx < 0 {
				idx = len(arr) + idx
			}
			if idx < 0 || idx >= len(arr) {
				return nil, nil // null
			}
			current = arr[idx]
		case tokenDot:
			// identity, no-op
		}
	}
	return current, nil
}

func outputJSON(v any, w io.Writer) {
	if v == nil {
		fmt.Fprintln(w, "null")
		return
	}

	switch val := v.(type) {
	case string:
		data, _ := json.Marshal(val)
		fmt.Fprintln(w, string(data))
	case float64:
		// Output integers without decimal point
		if val == float64(int64(val)) {
			fmt.Fprintln(w, strconv.FormatInt(int64(val), 10))
		} else {
			fmt.Fprintln(w, strconv.FormatFloat(val, 'f', -1, 64))
		}
	case bool:
		fmt.Fprintln(w, strconv.FormatBool(val))
	default:
		data, _ := json.MarshalIndent(val, "", "  ")
		fmt.Fprintln(w, string(data))
	}
}
