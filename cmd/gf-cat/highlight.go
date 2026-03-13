package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorKeyword = "\033[1;34m"  // bold blue
	colorString  = "\033[32m"    // green
	colorComment = "\033[90m"    // gray
	colorNumber  = "\033[36m"    // cyan
	colorKey     = "\033[33m"    // yellow
	colorBool    = "\033[35m"    // magenta
)

type language struct {
	keywords      map[string]bool
	lineComment   string
	blockComStart string
	blockComEnd   string
	hashComment   bool // # style comments
}

var languages = map[string]*language{
	".go": {
		keywords: toSet([]string{
			"break", "case", "chan", "const", "continue", "default", "defer",
			"else", "fallthrough", "for", "func", "go", "goto", "if",
			"import", "interface", "map", "package", "range", "return",
			"select", "struct", "switch", "type", "var",
			"true", "false", "nil",
		}),
		lineComment:   "//",
		blockComStart: "/*",
		blockComEnd:   "*/",
	},
	".py": {
		keywords: toSet([]string{
			"and", "as", "assert", "async", "await", "break", "class",
			"continue", "def", "del", "elif", "else", "except", "finally",
			"for", "from", "global", "if", "import", "in", "is", "lambda",
			"nonlocal", "not", "or", "pass", "raise", "return", "try",
			"while", "with", "yield",
			"True", "False", "None",
		}),
		hashComment: true,
	},
	".js": {
		keywords: toSet([]string{
			"async", "await", "break", "case", "catch", "class", "const",
			"continue", "debugger", "default", "delete", "do", "else",
			"export", "extends", "finally", "for", "function", "if",
			"import", "in", "instanceof", "let", "new", "of", "return",
			"static", "super", "switch", "this", "throw", "try", "typeof",
			"var", "void", "while", "yield",
			"true", "false", "null", "undefined",
		}),
		lineComment:   "//",
		blockComStart: "/*",
		blockComEnd:   "*/",
	},
	".json": nil, // special handling
	".yaml": nil, // special handling
	".yml":  nil, // same as .yaml
}

func toSet(words []string) map[string]bool {
	s := make(map[string]bool, len(words))
	for _, w := range words {
		s[w] = true
	}
	return s
}

func detectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := languages[ext]; ok {
		return ext
	}
	return ""
}

func highlightLine(line, lang string) string {
	switch lang {
	case ".json":
		return highlightJSON(line)
	case ".yaml", ".yml":
		return highlightYAML(line)
	default:
		l := languages[lang]
		if l == nil {
			return line
		}
		return highlightCode(line, l)
	}
}

func highlightCode(line string, lang *language) string {
	// Check for line comments first
	if lang.lineComment != "" {
		if idx := findCommentStart(line, lang.lineComment); idx >= 0 {
			before := highlightCodeTokens(line[:idx], lang)
			return before + colorComment + line[idx:] + colorReset
		}
	}
	if lang.hashComment {
		if idx := findCommentStart(line, "#"); idx >= 0 {
			before := highlightCodeTokens(line[:idx], lang)
			return before + colorComment + line[idx:] + colorReset
		}
	}
	return highlightCodeTokens(line, lang)
}

// findCommentStart finds comment marker that is not inside a string
func findCommentStart(line, marker string) int {
	inString := rune(0)
	escaped := false
	runes := []rune(line)
	markerRunes := []rune(marker)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString != 0 {
			escaped = true
			continue
		}
		if inString != 0 {
			if ch == inString {
				inString = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' || ch == '`' {
			inString = ch
			continue
		}
		if i+len(markerRunes) <= len(runes) {
			match := true
			for j, mr := range markerRunes {
				if runes[i+j] != mr {
					match = false
					break
				}
			}
			if match {
				return i
			}
		}
	}
	return -1
}

func highlightCodeTokens(line string, lang *language) string {
	var result strings.Builder
	runes := []rune(line)
	i := 0

	for i < len(runes) {
		ch := runes[i]

		// Strings
		if ch == '"' || ch == '\'' || ch == '`' {
			end := findStringEnd(runes, i)
			result.WriteString(colorString)
			result.WriteString(string(runes[i:end]))
			result.WriteString(colorReset)
			i = end
			continue
		}

		// Numbers
		if unicode.IsDigit(ch) && (i == 0 || !unicode.IsLetter(runes[i-1]) && runes[i-1] != '_') {
			start := i
			for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.' || runes[i] == 'x' || runes[i] == 'X' ||
				(runes[i] >= 'a' && runes[i] <= 'f') || (runes[i] >= 'A' && runes[i] <= 'F')) {
				i++
			}
			result.WriteString(colorNumber)
			result.WriteString(string(runes[start:i]))
			result.WriteString(colorReset)
			continue
		}

		// Identifiers / keywords
		if unicode.IsLetter(ch) || ch == '_' {
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			word := string(runes[start:i])
			if lang.keywords[word] {
				result.WriteString(colorKeyword)
				result.WriteString(word)
				result.WriteString(colorReset)
			} else {
				result.WriteString(word)
			}
			continue
		}

		result.WriteRune(ch)
		i++
	}

	return result.String()
}

func findStringEnd(runes []rune, start int) int {
	quote := runes[start]
	escaped := false
	for i := start + 1; i < len(runes); i++ {
		if escaped {
			escaped = false
			continue
		}
		if runes[i] == '\\' && quote != '`' {
			escaped = true
			continue
		}
		if runes[i] == quote {
			return i + 1
		}
	}
	return len(runes)
}

func highlightJSON(line string) string {
	var result strings.Builder
	runes := []rune(line)
	i := 0

	for i < len(runes) {
		ch := runes[i]

		// Strings - check if it's a key (followed by colon)
		if ch == '"' {
			end := findStringEnd(runes, i)
			strContent := string(runes[i:end])

			// Look ahead for colon (skip whitespace)
			isKey := false
			for j := end; j < len(runes); j++ {
				if runes[j] == ':' {
					isKey = true
					break
				} else if !unicode.IsSpace(runes[j]) {
					break
				}
			}

			if isKey {
				result.WriteString(colorKey)
			} else {
				result.WriteString(colorString)
			}
			result.WriteString(strContent)
			result.WriteString(colorReset)
			i = end
			continue
		}

		// Numbers
		if ch == '-' || unicode.IsDigit(ch) {
			start := i
			if ch == '-' {
				i++
			}
			for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.' || runes[i] == 'e' || runes[i] == 'E' || runes[i] == '+' || runes[i] == '-') {
				i++
			}
			if i > start+(func() int {
				if runes[start] == '-' {
					return 1
				}
				return 0
			}()) {
				result.WriteString(colorNumber)
				result.WriteString(string(runes[start:i]))
				result.WriteString(colorReset)
				continue
			}
			i = start
		}

		// Booleans and null
		remaining := string(runes[i:])
		for _, keyword := range []string{"true", "false", "null"} {
			if strings.HasPrefix(remaining, keyword) {
				next := i + len([]rune(keyword))
				if next >= len(runes) || !unicode.IsLetter(runes[next]) {
					result.WriteString(colorBool)
					result.WriteString(keyword)
					result.WriteString(colorReset)
					i = next
					goto continueOuter
				}
			}
		}

		result.WriteRune(ch)
		i++
	continueOuter:
	}

	return result.String()
}

func highlightYAML(line string) string {
	trimmed := strings.TrimSpace(line)

	// Comments
	if strings.HasPrefix(trimmed, "#") {
		return colorComment + line + colorReset
	}

	// Check for inline comment (# not in a string)
	if idx := findCommentStart(line, "#"); idx >= 0 {
		before := highlightYAMLContent(line[:idx])
		return before + colorComment + line[idx:] + colorReset
	}

	return highlightYAMLContent(line)
}

func highlightYAMLContent(line string) string {
	// Key: value pattern
	colonIdx := strings.Index(line, ":")
	if colonIdx < 0 {
		return highlightYAMLValue(line)
	}

	key := line[:colonIdx]
	// Verify key is a valid YAML key (no leading special chars except spaces)
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" || trimmedKey == "-" {
		return highlightYAMLValue(line)
	}

	rest := line[colonIdx:]
	result := fmt.Sprintf("%s%s%s%s", colorKey, key, colorReset, ":")
	if len(rest) > 1 {
		value := rest[1:]
		result += highlightYAMLValue(value)
	}
	return result
}

func highlightYAMLValue(s string) string {
	trimmed := strings.TrimSpace(s)

	// Boolean / null
	switch trimmed {
	case "true", "false", "yes", "no", "on", "off", "null", "~":
		return strings.Replace(s, trimmed, colorBool+trimmed+colorReset, 1)
	}

	// Numbers
	if len(trimmed) > 0 && (unicode.IsDigit(rune(trimmed[0])) || (trimmed[0] == '-' && len(trimmed) > 1)) {
		isNum := true
		hasDot := false
		start := 0
		if trimmed[0] == '-' {
			start = 1
		}
		for _, ch := range trimmed[start:] {
			if ch == '.' && !hasDot {
				hasDot = true
			} else if !unicode.IsDigit(ch) {
				isNum = false
				break
			}
		}
		if isNum {
			return strings.Replace(s, trimmed, colorNumber+trimmed+colorReset, 1)
		}
	}

	// Strings (quoted)
	if len(trimmed) >= 2 && ((trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"') ||
		(trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'')) {
		return strings.Replace(s, trimmed, colorString+trimmed+colorReset, 1)
	}

	return s
}
