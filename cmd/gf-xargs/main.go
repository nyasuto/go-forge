package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
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

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "gf-xargs version "+version)
		return 0
	}

	// Remaining args form the command template.
	cmdArgs := fs.Args()

	// Read stdin lines into items.
	items := readItems(stdin)

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

	// Build full argument list: command extra args + all items from stdin.
	fullArgs := make([]string, 0, len(cmdExtraArgs)+len(items))
	fullArgs = append(fullArgs, cmdExtraArgs...)
	fullArgs = append(fullArgs, items...)

	if err := executor.Execute(cmdName, fullArgs, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "gf-xargs: %s: %v\n", cmdName, err)
		return 1
	}

	return 0
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
