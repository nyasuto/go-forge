package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

const version = "0.1.0"

type grepOptions struct {
	ignoreCase bool
	invert     bool
	count      bool
	lineNumber bool
	recursive  bool
	jsonField  string
}

func main() {
	showVersion := flag.Bool("version", false, "バージョンを表示")
	opts := grepOptions{}
	flag.BoolVar(&opts.ignoreCase, "i", false, "大文字小文字を無視")
	flag.BoolVar(&opts.invert, "v", false, "マッチしない行を表示")
	flag.BoolVar(&opts.count, "c", false, "マッチした行数を表示")
	flag.BoolVar(&opts.lineNumber, "n", false, "行番号を表示")
	flag.BoolVar(&opts.recursive, "r", false, "ディレクトリを再帰的に検索")
	flag.StringVar(&opts.jsonField, "j", "", "JSONの指定キーのみを対象にマッチ")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-grep [OPTIONS] PATTERN [FILE]...\n\nパターンにマッチする行を表示する。\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("gf-grep version " + version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "gf-grep: パターンが指定されていません\n")
		os.Exit(2)
	}

	pattern := args[0]
	files := args[1:]

	re, err := compilePattern(pattern, opts.ignoreCase)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gf-grep: 不正な正規表現: %v\n", err)
		os.Exit(2)
	}

	exitCode := 1 // 1 = no match found

	if len(files) == 0 {
		if grep(os.Stdin, os.Stdout, re, "", opts) {
			exitCode = 0
		}
		os.Exit(exitCode)
	}

	// Expand files with -r
	if opts.recursive {
		expanded, errs := expandRecursive(files)
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "gf-grep: %v\n", e)
		}
		files = expanded
	}

	showFilename := len(files) > 1

	for _, name := range files {
		if name == "-" {
			if grep(os.Stdin, os.Stdout, re, stdinPrefix(showFilename), opts) {
				exitCode = 0
			}
			continue
		}

		f, err := os.Open(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gf-grep: %v\n", err)
			if exitCode != 0 {
				exitCode = 2
			}
			continue
		}

		prefix := ""
		if showFilename {
			prefix = name + ":"
		}
		if grep(f, os.Stdout, re, prefix, opts) {
			exitCode = 0
		}
		f.Close()
	}

	os.Exit(exitCode)
}

func compilePattern(pattern string, ignoreCase bool) (*regexp.Regexp, error) {
	if ignoreCase {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

func stdinPrefix(show bool) string {
	if show {
		return "(standard input):"
	}
	return ""
}

// expandRecursive walks directories and returns all regular files.
func expandRecursive(paths []string) ([]string, []error) {
	var files []string
	var errs []error
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !info.IsDir() {
			files = append(files, p)
			continue
		}
		filepath.Walk(p, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				errs = append(errs, err)
				return nil
			}
			if !fi.IsDir() && fi.Mode().IsRegular() {
				files = append(files, path)
			}
			return nil
		})
	}
	return files, errs
}

// grep searches for pattern in r and writes matching lines to w.
// Returns true if at least one match was found.
func grep(r io.Reader, w io.Writer, re *regexp.Regexp, prefix string, opts grepOptions) bool {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	found := false
	lineNum := 0
	matchCount := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		var matched bool
		if opts.jsonField != "" {
			matched = matchJSONField(line, re, opts.jsonField)
		} else {
			matched = re.MatchString(line)
		}

		if opts.invert {
			matched = !matched
		}

		if matched {
			found = true
			matchCount++
			if !opts.count {
				printLine(bw, prefix, line, lineNum, opts.lineNumber)
			}
		}
	}

	if opts.count {
		if prefix != "" {
			fmt.Fprintf(bw, "%s%d\n", prefix, matchCount)
		} else {
			fmt.Fprintf(bw, "%d\n", matchCount)
		}
	}

	return found
}

// matchJSONField parses line as JSON and checks if the specified field's value matches the pattern.
// Supports dot-separated nested keys (e.g., "user.name").
func matchJSONField(line string, re *regexp.Regexp, field string) bool {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return false
	}
	val, ok := lookupJSON(obj, field)
	if !ok {
		return false
	}
	s := fmt.Sprintf("%v", val)
	return re.MatchString(s)
}

// lookupJSON retrieves a value from a nested map using dot-separated keys.
func lookupJSON(obj map[string]interface{}, field string) (interface{}, bool) {
	keys := splitDot(field)
	var current interface{} = obj
	for _, key := range keys {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = m[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// splitDot splits a string by dots.
func splitDot(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func printLine(w *bufio.Writer, prefix, line string, lineNum int, showLineNum bool) {
	if prefix != "" {
		fmt.Fprint(w, prefix)
	}
	if showLineNum {
		fmt.Fprintf(w, "%d:", lineNum)
	}
	fmt.Fprintf(w, "%s\n", line)
}

