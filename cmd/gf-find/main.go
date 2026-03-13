package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const version = "0.1.0"

type findOptions struct {
	namePattern string
	pathPattern string // glob pattern against full path
	typeFilter  string // "f" for files, "d" for directories
	sizeExpr    string // e.g. "+100k", "-1M", "50c"
	mtimeExpr   string // e.g. "+7", "-1", "0"
	execCmd     string // command template with {} placeholder
}

func main() {
	showVersion := flag.Bool("version", false, "バージョンを表示")
	namePattern := flag.String("name", "", "ファイル名パターン（glob形式）")
	pathPattern := flag.String("path", "", "フルパスに対するglobパターン")
	typeFilter := flag.String("type", "", "タイプフィルタ: f（ファイル）, d（ディレクトリ）")
	sizeExpr := flag.String("size", "", "サイズ条件: +N[ckMG]（より大きい）, -N[ckMG]（より小さい）, N[ckMG]（ちょうど）")
	mtimeExpr := flag.String("mtime", "", "更新日条件: +N（N日より前）, -N（N日以内）, N（ちょうどN日前）")
	execCmd := flag.String("exec", "", "マッチしたファイルに対して実行するコマンド（{}がパスに置換、確認プロンプト付き）")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gf-find [OPTIONS] [PATH]...\n\n再帰的にファイルを検索する。\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("gf-find version " + version)
		os.Exit(0)
	}

	// Validate -type
	if *typeFilter != "" && *typeFilter != "f" && *typeFilter != "d" {
		fmt.Fprintf(os.Stderr, "gf-find: -type の値は 'f'（ファイル）または 'd'（ディレクトリ）を指定してください\n")
		os.Exit(2)
	}

	// Validate -size
	if *sizeExpr != "" {
		if _, _, err := parseSizeExpr(*sizeExpr); err != nil {
			fmt.Fprintf(os.Stderr, "gf-find: -size の値が不正です: %v\n", err)
			os.Exit(2)
		}
	}

	// Validate -mtime
	if *mtimeExpr != "" {
		if _, _, err := parseMtimeExpr(*mtimeExpr); err != nil {
			fmt.Fprintf(os.Stderr, "gf-find: -mtime の値が不正です: %v\n", err)
			os.Exit(2)
		}
	}

	opts := findOptions{
		namePattern: *namePattern,
		pathPattern: *pathPattern,
		typeFilter:  *typeFilter,
		sizeExpr:    *sizeExpr,
		mtimeExpr:   *mtimeExpr,
		execCmd:     *execCmd,
	}

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	exitCode := 0
	for _, root := range paths {
		if err := find(root, opts); err != nil {
			fmt.Fprintf(os.Stderr, "gf-find: %v\n", err)
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

// nowFunc is overridable for testing
var nowFunc = time.Now

// promptReader is the reader for confirmation prompts (overridable for testing)
var promptReader *bufio.Reader

func getPromptReader() *bufio.Reader {
	if promptReader != nil {
		return promptReader
	}
	promptReader = bufio.NewReader(os.Stdin)
	return promptReader
}

func find(root string, opts findOptions) error {
	info, err := os.Lstat(root)
	if err != nil {
		return err
	}

	// If root is not a directory, check if it matches and print
	if !info.IsDir() {
		if matchEntry(root, info, opts) {
			handleMatch(root, opts)
		}
		return nil
	}

	return filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "gf-find: %v\n", err)
			return nil
		}
		if matchEntry(path, fi, opts) {
			handleMatch(path, opts)
		}
		return nil
	})
}

func handleMatch(path string, opts findOptions) {
	if opts.execCmd == "" {
		fmt.Println(path)
		return
	}
	executeCmd(path, opts.execCmd)
}

func executeCmd(path, cmdTemplate string) {
	cmdStr := strings.ReplaceAll(cmdTemplate, "{}", path)
	fmt.Fprintf(os.Stderr, "< %s >? ", cmdStr)
	reader := getPromptReader()
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		return
	}
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gf-find: exec エラー: %v\n", err)
	}
}

func matchEntry(path string, fi os.FileInfo, opts findOptions) bool {
	if !matchName(fi.Name(), opts.namePattern) {
		return false
	}
	if !matchPath(path, opts.pathPattern) {
		return false
	}
	if !matchType(fi, opts.typeFilter) {
		return false
	}
	if opts.sizeExpr != "" && !matchSize(fi, opts.sizeExpr) {
		return false
	}
	if opts.mtimeExpr != "" && !matchMtime(fi, opts.mtimeExpr) {
		return false
	}
	return true
}

func matchName(name, pattern string) bool {
	if pattern == "" {
		return true
	}
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return false
	}
	return matched
}

func matchPath(path, pattern string) bool {
	if pattern == "" {
		return true
	}
	// Try matching against full path using filepath.Match
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false
	}
	return matched
}

func matchType(fi os.FileInfo, typeFilter string) bool {
	if typeFilter == "" {
		return true
	}
	switch typeFilter {
	case "f":
		return !fi.IsDir()
	case "d":
		return fi.IsDir()
	}
	return true
}

// parseSizeExpr parses a size expression like "+100k", "-1M", "50c", "100".
// Returns: comparison operator (+, -, or =), size in bytes, error.
func parseSizeExpr(expr string) (string, int64, error) {
	if expr == "" {
		return "", 0, fmt.Errorf("空の式")
	}

	op := "="
	s := expr
	if s[0] == '+' || s[0] == '-' {
		op = string(s[0])
		s = s[1:]
	}

	if len(s) == 0 {
		return "", 0, fmt.Errorf("数値が指定されていません")
	}

	// Check for unit suffix
	multiplier := int64(512) // default: 512-byte blocks (like find)
	lastChar := s[len(s)-1]
	switch lastChar {
	case 'c':
		multiplier = 1
		s = s[:len(s)-1]
	case 'k':
		multiplier = 1024
		s = s[:len(s)-1]
	case 'M':
		multiplier = 1024 * 1024
		s = s[:len(s)-1]
	case 'G':
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	default:
		if lastChar < '0' || lastChar > '9' {
			return "", 0, fmt.Errorf("不明な単位: %c", lastChar)
		}
	}

	if len(s) == 0 {
		return "", 0, fmt.Errorf("数値が指定されていません")
	}

	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("数値のパースに失敗: %v", err)
	}

	return op, n * multiplier, nil
}

func matchSize(fi os.FileInfo, expr string) bool {
	op, threshold, err := parseSizeExpr(expr)
	if err != nil {
		return false
	}

	size := fi.Size()
	switch op {
	case "+":
		return size > threshold
	case "-":
		return size < threshold
	case "=":
		return size == threshold
	}
	return false
}

// parseMtimeExpr parses an mtime expression like "+7", "-1", "0".
// Returns: comparison operator (+, -, or =), days, error.
func parseMtimeExpr(expr string) (string, int, error) {
	if expr == "" {
		return "", 0, fmt.Errorf("空の式")
	}

	op := "="
	s := expr
	if s[0] == '+' || s[0] == '-' {
		op = string(s[0])
		s = s[1:]
	}

	if len(s) == 0 {
		return "", 0, fmt.Errorf("数値が指定されていません")
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return "", 0, fmt.Errorf("数値のパースに失敗: %v", err)
	}

	return op, n, nil
}

func matchMtime(fi os.FileInfo, expr string) bool {
	op, days, err := parseMtimeExpr(expr)
	if err != nil {
		return false
	}

	now := nowFunc()
	modTime := fi.ModTime()
	// Calculate age in days (fractional days truncated)
	age := int(now.Sub(modTime).Hours() / 24)

	switch op {
	case "+":
		return age > days
	case "-":
		return age < days
	case "=":
		return age == days
	}
	return false
}


