package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, realExecutor{}))
}

// commandExecutor abstracts command execution for testing.
type commandExecutor interface {
	Execute(name string, args []string, stdout, stderr io.Writer) error
}

type realExecutor struct{}

func (r realExecutor) Execute(name string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, executor commandExecutor) int {
	fs := flag.NewFlagSet("gf-xargs", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "バージョンを表示")
	maxArgs := fs.Int("n", 0, "コマンドごとの最大引数数")
	parallel := fs.Int("P", 1, "並列実行数")
	nullDelim := fs.Bool("0", false, "null文字区切りで入力を読み取り")
	dryRun := fs.Bool("dry-run", false, "コマンドを実行せず表示のみ")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "gf-xargs version "+version)
		return 0
	}

	if *maxArgs < 0 {
		fmt.Fprintln(stderr, "gf-xargs: -n must be a positive integer")
		return 2
	}
	if *parallel < 1 {
		fmt.Fprintln(stderr, "gf-xargs: -P must be at least 1")
		return 2
	}

	// Remaining args form the command template.
	cmdArgs := fs.Args()

	// Read stdin lines into items.
	var items []string
	if *nullDelim {
		items = readItemsNull(stdin)
	} else {
		items = readItems(stdin)
	}

	if len(items) == 0 {
		return 0
	}

	// If no command specified, default to "echo".
	cmdName := "echo"
	var cmdExtraArgs []string
	if len(cmdArgs) > 0 {
		cmdName = cmdArgs[0]
		cmdExtraArgs = cmdArgs[1:]
	}

	// Split items into batches based on -n flag.
	batches := splitBatches(items, *maxArgs)

	if *dryRun {
		for _, batch := range batches {
			fullArgs := make([]string, 0, len(cmdExtraArgs)+len(batch))
			fullArgs = append(fullArgs, cmdExtraArgs...)
			fullArgs = append(fullArgs, batch...)
			fmt.Fprintln(stdout, shellJoin(cmdName, fullArgs))
		}
		return 0
	}

	if *parallel <= 1 {
		// Sequential execution.
		for _, batch := range batches {
			fullArgs := make([]string, 0, len(cmdExtraArgs)+len(batch))
			fullArgs = append(fullArgs, cmdExtraArgs...)
			fullArgs = append(fullArgs, batch...)

			if err := executor.Execute(cmdName, fullArgs, stdout, stderr); err != nil {
				fmt.Fprintf(stderr, "gf-xargs: %s: %v\n", cmdName, err)
				return 1
			}
		}
	} else {
		// Parallel execution.
		exitCode := runParallel(batches, cmdName, cmdExtraArgs, *parallel, stdout, stderr, executor)
		if exitCode != 0 {
			return exitCode
		}
	}

	return 0
}

// splitBatches splits items into batches of at most n items.
// If n <= 0, all items go into a single batch.
func splitBatches(items []string, n int) [][]string {
	if n <= 0 {
		return [][]string{items}
	}
	var batches [][]string
	for i := 0; i < len(items); i += n {
		end := i + n
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}
	return batches
}

// runParallel executes batches in parallel with at most maxP concurrent processes.
func runParallel(batches [][]string, cmdName string, cmdExtraArgs []string, maxP int, stdout, stderr io.Writer, executor commandExecutor) int {
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxP)
	exitCode := 0

	for _, batch := range batches {
		wg.Add(1)
		sem <- struct{}{} // acquire semaphore
		go func(b []string) {
			defer wg.Done()
			defer func() { <-sem }() // release semaphore

			fullArgs := make([]string, 0, len(cmdExtraArgs)+len(b))
			fullArgs = append(fullArgs, cmdExtraArgs...)
			fullArgs = append(fullArgs, b...)

			// Use per-goroutine buffers to avoid interleaving output.
			var outBuf, errBuf bytes.Buffer
			if err := executor.Execute(cmdName, fullArgs, &outBuf, &errBuf); err != nil {
				mu.Lock()
				fmt.Fprintf(stderr, "gf-xargs: %s: %v\n", cmdName, err)
				exitCode = 1
				mu.Unlock()
			}

			mu.Lock()
			outBuf.WriteTo(stdout)
			errBuf.WriteTo(stderr)
			mu.Unlock()
		}(batch)
	}
	wg.Wait()
	return exitCode
}

// readItemsNull reads null-byte delimited items from stdin.
func readItemsNull(r io.Reader) []string {
	var items []string
	scanner := bufio.NewScanner(r)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, 0); i >= 0 {
			return i + 1, data[:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	for scanner.Scan() {
		item := scanner.Text()
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

// shellJoin formats a command and its args as a shell command string.
// Arguments containing spaces or special characters are single-quoted.
func shellJoin(name string, args []string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, shellQuote(name))
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

// shellQuote returns a shell-safe representation of s.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	safe := true
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '/' || c == ':' || c == '=' || c == '+' || c == ',') {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// readItems reads whitespace-delimited tokens from stdin,
// handling quoted strings (single and double quotes).
func readItems(r io.Reader) []string {
	var items []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		items = append(items, splitArgs(line)...)
	}
	return items
}

// splitArgs splits a line into arguments, respecting single and double quotes.
func splitArgs(line string) []string {
	var args []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case (ch == ' ' || ch == '\t') && !inSingle && !inDouble:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
