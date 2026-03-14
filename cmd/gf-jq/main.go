package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
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

	stages, err := parseFilter(filter)
	if err != nil {
		fmt.Fprintf(stderr, "gf-jq: %v\n", err)
		return 2
	}

	exitCode := 0

	if len(files) == 0 || (len(files) == 1 && files[0] == "-") {
		if code := processReader(stdin, stages, stdout, stderr); code != 0 {
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
			if code := processReader(f, stages, stdout, stderr); code != 0 {
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
	tokenKey      tokenType = iota // .key
	tokenIndex                     // .[N]
	tokenDot                       // . (identity)
	tokenIterator                  // .[]
	tokenFunc                      // length, keys, values
	tokenSelect                    // select(condition)
)

type selectCond struct {
	filter []token // left side filter path
	op     string  // comparison operator ("==", "!=", ">", "<", ">=", "<="), "" for truthiness
	value  any     // right side literal value
}

type token struct {
	typ   tokenType
	key   string
	index int
	cond  *selectCond // for tokenSelect
}

// parseFilter parses a filter expression into a pipeline of stages separated by |
func parseFilter(filter string) ([][]token, error) {
	parts := splitPipeline(filter)
	var stages [][]token
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("invalid filter: empty pipeline stage in %q", filter)
		}
		tokens, err := parseFilterStage(part)
		if err != nil {
			return nil, err
		}
		stages = append(stages, tokens)
	}
	if len(stages) == 0 {
		return nil, fmt.Errorf("invalid filter: %q", filter)
	}
	return stages, nil
}

// splitPipeline splits a filter by | but respects parentheses and quoted strings
func splitPipeline(filter string) []string {
	var parts []string
	depth := 0
	inString := false
	start := 0
	for i := 0; i < len(filter); i++ {
		ch := filter[i]
		if ch == '"' && !inString {
			inString = true
			continue
		}
		if inString {
			if ch == '\\' {
				i++
			} else if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
		} else if ch == '|' && depth == 0 {
			parts = append(parts, filter[start:i])
			start = i + 1
		}
	}
	parts = append(parts, filter[start:])
	return parts
}

// parseFilterStage parses a single pipeline stage
func parseFilterStage(stage string) ([]token, error) {
	if stage == "." {
		return []token{{typ: tokenDot}}, nil
	}

	// Check for bare function names
	switch stage {
	case "length", "keys", "values":
		return []token{{typ: tokenFunc, key: stage}}, nil
	}

	// Check for select(...)
	if strings.HasPrefix(stage, "select(") && strings.HasSuffix(stage, ")") {
		condStr := stage[7 : len(stage)-1]
		cond, err := parseSelectCondition(condStr)
		if err != nil {
			return nil, err
		}
		return []token{{typ: tokenSelect, cond: cond}}, nil
	}

	if !strings.HasPrefix(stage, ".") {
		return nil, fmt.Errorf("invalid filter: %q (must start with '.')", stage)
	}

	var tokens []token
	s := stage[1:] // skip leading dot

	for len(s) > 0 {
		if s[0] == '[' {
			if len(s) > 1 && s[1] == ']' {
				// .[] iterator
				tokens = append(tokens, token{typ: tokenIterator})
				s = s[2:]
				if len(s) > 0 && s[0] == '.' {
					s = s[1:]
				}
			} else {
				// array index: [N]
				end := strings.IndexByte(s, ']')
				if end == -1 {
					return nil, fmt.Errorf("invalid filter: unclosed bracket in %q", stage)
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
			}
		} else {
			// key access
			end := indexOfAny(s, ".[")
			if end == -1 {
				end = len(s)
			}
			key := s[:end]
			if key == "" {
				return nil, fmt.Errorf("invalid filter: empty key in %q", stage)
			}
			tokens = append(tokens, token{typ: tokenKey, key: key})
			s = s[end:]
			if len(s) > 0 && s[0] == '.' {
				s = s[1:]
			}
		}
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("invalid filter: %q", stage)
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

// parseSelectCondition parses the content inside select(...)
func parseSelectCondition(s string) (*selectCond, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("select: empty condition")
	}

	// Try to find comparison operator (check two-char operators first)
	operators := []string{"==", "!=", ">=", "<=", ">", "<"}
	for _, op := range operators {
		idx := findOperator(s, op)
		if idx >= 0 {
			left := strings.TrimSpace(s[:idx])
			right := strings.TrimSpace(s[idx+len(op):])

			filterTokens, err := parseFilterStage(left)
			if err != nil {
				return nil, fmt.Errorf("select: invalid left side: %v", err)
			}

			var val any
			if err := json.Unmarshal([]byte(right), &val); err != nil {
				return nil, fmt.Errorf("select: invalid value %q: %v", right, err)
			}

			return &selectCond{filter: filterTokens, op: op, value: val}, nil
		}
	}

	// No operator found: truthiness check
	filterTokens, err := parseFilterStage(s)
	if err != nil {
		return nil, fmt.Errorf("select: invalid condition: %v", err)
	}
	return &selectCond{filter: filterTokens, op: "", value: nil}, nil
}

// findOperator finds operator position, skipping quoted strings
func findOperator(s, op string) int {
	inString := false
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			inString = !inString
			continue
		}
		if inString {
			if s[i] == '\\' {
				i++ // skip escaped char
			}
			continue
		}
		if i+len(op) <= len(s) && s[i:i+len(op)] == op {
			return i
		}
	}
	return -1
}

func processReader(r io.Reader, stages [][]token, stdout, stderr io.Writer) int {
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

	results, err := applyPipeline(input, stages)
	if err != nil {
		fmt.Fprintf(stderr, "gf-jq: %v\n", err)
		return 1
	}

	for _, result := range results {
		outputJSON(result, stdout)
	}
	return 0
}

// applyPipeline chains pipeline stages, passing each output as input to the next stage
func applyPipeline(data any, stages [][]token) ([]any, error) {
	results := []any{data}
	for _, stage := range stages {
		var next []any
		for _, input := range results {
			outputs, err := applyStage(input, stage)
			if err != nil {
				return nil, err
			}
			next = append(next, outputs...)
		}
		results = next
	}
	return results, nil
}

// applyStage applies a single pipeline stage, which may produce multiple outputs (from .[] iterator)
func applyStage(data any, tokens []token) ([]any, error) {
	results := []any{data}
	for _, tok := range tokens {
		var next []any
		for _, current := range results {
			switch tok.typ {
			case tokenDot:
				next = append(next, current)
			case tokenKey:
				obj, ok := current.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("cannot index %T with key %q", current, tok.key)
				}
				val, exists := obj[tok.key]
				if !exists {
					next = append(next, nil)
				} else {
					next = append(next, val)
				}
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
					next = append(next, nil)
				} else {
					next = append(next, arr[idx])
				}
			case tokenIterator:
				switch v := current.(type) {
				case []any:
					next = append(next, v...)
				case map[string]any:
					keys := make([]string, 0, len(v))
					for k := range v {
						keys = append(keys, k)
					}
					sort.Strings(keys)
					for _, k := range keys {
						next = append(next, v[k])
					}
				default:
					return nil, fmt.Errorf("cannot iterate over %T", current)
				}
			case tokenFunc:
				val, err := applyFunc(tok.key, current)
				if err != nil {
					return nil, err
				}
				next = append(next, val)
			case tokenSelect:
				keep, err := evalSelect(tok.cond, current)
				if err != nil {
					return nil, err
				}
				if keep {
					next = append(next, current)
				}
			}
		}
		results = next
	}
	return results, nil
}

// applyFunc applies a built-in function to a value
func applyFunc(name string, data any) (any, error) {
	switch name {
	case "length":
		if data == nil {
			return float64(0), nil
		}
		switch v := data.(type) {
		case []any:
			return float64(len(v)), nil
		case map[string]any:
			return float64(len(v)), nil
		case string:
			return float64(utf8.RuneCountInString(v)), nil
		case float64:
			if v < 0 {
				return -v, nil
			}
			return v, nil
		default:
			return nil, fmt.Errorf("cannot get length of %T", data)
		}
	case "keys":
		switch v := data.(type) {
		case map[string]any:
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			result := make([]any, len(keys))
			for i, k := range keys {
				result[i] = k
			}
			return result, nil
		case []any:
			result := make([]any, len(v))
			for i := range v {
				result[i] = float64(i)
			}
			return result, nil
		default:
			if data == nil {
				return nil, fmt.Errorf("cannot get keys of null")
			}
			return nil, fmt.Errorf("cannot get keys of %T", data)
		}
	case "values":
		switch v := data.(type) {
		case map[string]any:
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			result := make([]any, len(keys))
			for i, k := range keys {
				result[i] = v[k]
			}
			return result, nil
		case []any:
			// values of an array is the array itself
			return v, nil
		default:
			if data == nil {
				return nil, fmt.Errorf("cannot get values of null")
			}
			return nil, fmt.Errorf("cannot get values of %T", data)
		}
	default:
		return nil, fmt.Errorf("unknown function: %s", name)
	}
}

// evalSelect evaluates a select condition against a value
func evalSelect(cond *selectCond, data any) (bool, error) {
	// Apply the filter to get the left side value
	results, err := applyStage(data, cond.filter)
	if err != nil {
		return false, err
	}

	if len(results) == 0 {
		return false, nil
	}

	val := results[0]

	// Truthiness check (no operator)
	if cond.op == "" {
		return isTruthy(val), nil
	}

	// Comparison
	return compareValues(val, cond.op, cond.value)
}

// isTruthy returns true if the value is truthy (not null and not false)
func isTruthy(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}

// compareValues compares two values with the given operator
func compareValues(left any, op string, right any) (bool, error) {
	// Handle null comparisons
	if left == nil && right == nil {
		switch op {
		case "==":
			return true, nil
		case "!=":
			return false, nil
		default:
			return false, nil
		}
	}
	if left == nil || right == nil {
		switch op {
		case "==":
			return false, nil
		case "!=":
			return true, nil
		default:
			return false, nil
		}
	}

	// Numeric comparison
	lNum, lOk := toFloat64(left)
	rNum, rOk := toFloat64(right)
	if lOk && rOk {
		switch op {
		case "==":
			return lNum == rNum, nil
		case "!=":
			return lNum != rNum, nil
		case ">":
			return lNum > rNum, nil
		case "<":
			return lNum < rNum, nil
		case ">=":
			return lNum >= rNum, nil
		case "<=":
			return lNum <= rNum, nil
		}
	}

	// String comparison
	lStr, lSok := left.(string)
	rStr, rSok := right.(string)
	if lSok && rSok {
		switch op {
		case "==":
			return lStr == rStr, nil
		case "!=":
			return lStr != rStr, nil
		case ">":
			return lStr > rStr, nil
		case "<":
			return lStr < rStr, nil
		case ">=":
			return lStr >= rStr, nil
		case "<=":
			return lStr <= rStr, nil
		}
	}

	// Bool comparison (== and != only)
	lBool, lBok := left.(bool)
	rBool, rBok := right.(bool)
	if lBok && rBok {
		switch op {
		case "==":
			return lBool == rBool, nil
		case "!=":
			return lBool != rBool, nil
		default:
			return false, fmt.Errorf("cannot compare booleans with %s", op)
		}
	}

	// Type mismatch
	switch op {
	case "==":
		return false, nil
	case "!=":
		return true, nil
	default:
		return false, nil
	}
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
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
