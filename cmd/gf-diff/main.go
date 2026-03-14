package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

const version = "0.1.0"

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorCyan    = "\033[36m"
	colorBoldRed = "\033[1;31m"
	colorBoldGrn = "\033[1;32m"
)

// isTerminal checks if the writer is connected to a terminal.
var isTerminal = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gf-diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "show version")
	unified := fs.Bool("u", false, "unified diff format")
	colorFlag := fs.String("color", "auto", "color output: auto|always|never")
	wordDiff := fs.Bool("word", false, "word-level diff within changed lines")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "gf-diff version "+version)
		return 0
	}

	// Validate color flag
	switch *colorFlag {
	case "auto", "always", "never":
	default:
		fmt.Fprintf(stderr, "gf-diff: invalid --color value: %s (use auto, always, or never)\n", *colorFlag)
		return 2
	}

	useColor := *colorFlag == "always" || (*colorFlag == "auto" && isTerminal(stdout))

	remaining := fs.Args()
	if len(remaining) != 2 {
		fmt.Fprintln(stderr, "usage: gf-diff <file1> <file2>")
		return 2
	}

	file1, file2 := remaining[0], remaining[1]

	lines1, err := readLines(file1)
	if err != nil {
		fmt.Fprintf(stderr, "gf-diff: %s: %v\n", file1, err)
		return 1
	}
	lines2, err := readLines(file2)
	if err != nil {
		fmt.Fprintf(stderr, "gf-diff: %s: %v\n", file2, err)
		return 1
	}

	edits := myersDiff(lines1, lines2)

	if !hasDifferences(edits) {
		return 0
	}

	opts := outputOpts{
		color:    useColor,
		wordDiff: *wordDiff,
	}

	if *unified {
		printUnified(stdout, file1, file2, edits, 3, opts)
	} else {
		printNormal(stdout, edits, opts)
	}

	return 1
}

type outputOpts struct {
	color    bool
	wordDiff bool
}

func printNormal(w io.Writer, edits []edit, opts outputOpts) {
	if opts.wordDiff {
		printNormalWordDiff(w, edits, opts)
		return
	}
	for _, e := range edits {
		switch e.op {
		case opDelete:
			if opts.color {
				fmt.Fprintf(w, "%s< %s%s\n", colorRed, e.line, colorReset)
			} else {
				fmt.Fprintf(w, "< %s\n", e.line)
			}
		case opInsert:
			if opts.color {
				fmt.Fprintf(w, "%s> %s%s\n", colorGreen, e.line, colorReset)
			} else {
				fmt.Fprintf(w, "> %s\n", e.line)
			}
		case opEqual:
			fmt.Fprintf(w, "  %s\n", e.line)
		}
	}
}

func printNormalWordDiff(w io.Writer, edits []edit, opts outputOpts) {
	i := 0
	for i < len(edits) {
		e := edits[i]
		if e.op == opEqual {
			fmt.Fprintf(w, "  %s\n", e.line)
			i++
			continue
		}
		// Collect adjacent delete/insert pairs for word diff
		var delLines, insLines []string
		for i < len(edits) && edits[i].op == opDelete {
			delLines = append(delLines, edits[i].line)
			i++
		}
		for i < len(edits) && edits[i].op == opInsert {
			insLines = append(insLines, edits[i].line)
			i++
		}
		// Pair up delete/insert lines for word diff
		maxPairs := len(delLines)
		if len(insLines) > maxPairs {
			maxPairs = len(insLines)
		}
		for j := 0; j < maxPairs; j++ {
			if j < len(delLines) && j < len(insLines) {
				oldWd, newWd := wordDiffLine(delLines[j], insLines[j])
				if opts.color {
					fmt.Fprintf(w, "%s< %s%s\n", colorRed, oldWd, colorReset)
					fmt.Fprintf(w, "%s> %s%s\n", colorGreen, newWd, colorReset)
				} else {
					fmt.Fprintf(w, "< %s\n", oldWd)
					fmt.Fprintf(w, "> %s\n", newWd)
				}
			} else if j < len(delLines) {
				if opts.color {
					fmt.Fprintf(w, "%s< %s%s\n", colorRed, delLines[j], colorReset)
				} else {
					fmt.Fprintf(w, "< %s\n", delLines[j])
				}
			} else {
				if opts.color {
					fmt.Fprintf(w, "%s> %s%s\n", colorGreen, insLines[j], colorReset)
				} else {
					fmt.Fprintf(w, "> %s\n", insLines[j])
				}
			}
		}
	}
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// readLinesFromString is a helper for testing.
func readLinesFromString(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

type editOp int

const (
	opEqual  editOp = iota
	opDelete        // line from file1 not in file2
	opInsert        // line from file2 not in file1
)

type edit struct {
	op   editOp
	line string
}

// myersDiff implements the Myers diff algorithm.
// Returns a list of edits (equal/delete/insert) representing the shortest edit script.
func myersDiff(a, b []string) []edit {
	n := len(a)
	m := len(b)

	if n == 0 && m == 0 {
		return nil
	}

	max := n + m
	offset := max
	size := 2*max + 1

	// trace[d] stores V snapshot after processing edit distance d
	var trace [][]int

	v := make([]int, size)
	for i := range v {
		v[i] = 0
	}

	var finalD int
	for d := 0; d <= max; d++ {
		// Save a copy of V before this iteration modifies it
		vc := make([]int, size)
		copy(vc, v)
		trace = append(trace, vc)

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1+offset] < v[k+1+offset]) {
				x = v[k+1+offset] // move down
			} else {
				x = v[k-1+offset] + 1 // move right
			}
			y := x - k

			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}

			v[k+offset] = x

			if x >= n && y >= m {
				finalD = d
				goto backtrack
			}
		}
	}

backtrack:
	x := n
	y := m
	var edits []edit

	for d := finalD; d > 0; d-- {
		vd := trace[d]
		k := x - y

		var prevK int
		if k == -d || (k != d && vd[k-1+offset] < vd[k+1+offset]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := vd[prevK+offset]
		prevY := prevX - prevK

		// Diagonal (equal lines)
		for x > prevX && y > prevY {
			x--
			y--
			edits = append(edits, edit{op: opEqual, line: a[x]})
		}

		if x == prevX {
			y--
			edits = append(edits, edit{op: opInsert, line: b[y]})
		} else {
			x--
			edits = append(edits, edit{op: opDelete, line: a[x]})
		}
	}

	// Remaining diagonal at d=0
	for x > 0 && y > 0 {
		x--
		y--
		edits = append(edits, edit{op: opEqual, line: a[x]})
	}

	// Reverse
	for i, j := 0, len(edits)-1; i < j; i, j = i+1, j-1 {
		edits[i], edits[j] = edits[j], edits[i]
	}

	return edits
}

// hasDifferences returns true if the edit script contains any non-equal operations.
func hasDifferences(edits []edit) bool {
	for _, e := range edits {
		if e.op != opEqual {
			return true
		}
	}
	return false
}

// hunk represents a unified diff hunk.
type hunk struct {
	oldStart int // 1-based line number in file1
	oldCount int
	newStart int // 1-based line number in file2
	newCount int
	lines    []edit
}

// buildHunks groups edits into hunks with the given context lines.
func buildHunks(edits []edit, contextLines int) []hunk {
	if len(edits) == 0 {
		return nil
	}

	// Find indices of change edits (non-equal)
	var changeIndices []int
	for i, e := range edits {
		if e.op != opEqual {
			changeIndices = append(changeIndices, i)
		}
	}
	if len(changeIndices) == 0 {
		return nil
	}

	// Group changes into hunk ranges
	type hunkRange struct {
		start, end int // indices into edits slice (inclusive start, exclusive end)
	}

	var ranges []hunkRange
	rangeStart := changeIndices[0] - contextLines
	if rangeStart < 0 {
		rangeStart = 0
	}
	rangeEnd := changeIndices[0] + 1

	for i := 1; i < len(changeIndices); i++ {
		idx := changeIndices[i]
		// If this change is within context distance of previous range end, merge
		if idx-rangeEnd <= 2*contextLines {
			rangeEnd = idx + 1
		} else {
			// Finalize previous range with trailing context
			end := rangeEnd + contextLines
			if end > len(edits) {
				end = len(edits)
			}
			ranges = append(ranges, hunkRange{rangeStart, end})

			rangeStart = idx - contextLines
			if rangeStart < 0 {
				rangeStart = 0
			}
			rangeEnd = idx + 1
		}
	}
	// Finalize last range
	end := rangeEnd + contextLines
	if end > len(edits) {
		end = len(edits)
	}
	ranges = append(ranges, hunkRange{rangeStart, end})

	// Build hunks from ranges
	var hunks []hunk
	for _, r := range ranges {
		h := hunk{}
		// Calculate line numbers by counting edits before this range
		oldLine := 1
		newLine := 1
		for i := 0; i < r.start; i++ {
			switch edits[i].op {
			case opEqual:
				oldLine++
				newLine++
			case opDelete:
				oldLine++
			case opInsert:
				newLine++
			}
		}
		h.oldStart = oldLine
		h.newStart = newLine

		for i := r.start; i < r.end; i++ {
			h.lines = append(h.lines, edits[i])
			switch edits[i].op {
			case opEqual:
				h.oldCount++
				h.newCount++
			case opDelete:
				h.oldCount++
			case opInsert:
				h.newCount++
			}
		}
		hunks = append(hunks, h)
	}

	return hunks
}

// printUnified outputs the diff in unified format.
func printUnified(w io.Writer, file1, file2 string, edits []edit, contextLines int, opts outputOpts) {
	if opts.color {
		fmt.Fprintf(w, "%s--- %s%s\n", colorBoldRed, file1, colorReset)
		fmt.Fprintf(w, "%s+++ %s%s\n", colorBoldGrn, file2, colorReset)
	} else {
		fmt.Fprintf(w, "--- %s\n", file1)
		fmt.Fprintf(w, "+++ %s\n", file2)
	}

	hunks := buildHunks(edits, contextLines)
	for _, h := range hunks {
		if opts.color {
			fmt.Fprintf(w, "%s@@ -%d,%d +%d,%d @@%s\n", colorCyan, h.oldStart, h.oldCount, h.newStart, h.newCount, colorReset)
		} else {
			fmt.Fprintf(w, "@@ -%d,%d +%d,%d @@\n", h.oldStart, h.oldCount, h.newStart, h.newCount)
		}
		if opts.wordDiff {
			printUnifiedHunkWordDiff(w, h.lines, opts)
		} else {
			for _, e := range h.lines {
				printUnifiedLine(w, e, opts)
			}
		}
	}
}

func printUnifiedLine(w io.Writer, e edit, opts outputOpts) {
	switch e.op {
	case opEqual:
		fmt.Fprintf(w, " %s\n", e.line)
	case opDelete:
		if opts.color {
			fmt.Fprintf(w, "%s-%s%s\n", colorRed, e.line, colorReset)
		} else {
			fmt.Fprintf(w, "-%s\n", e.line)
		}
	case opInsert:
		if opts.color {
			fmt.Fprintf(w, "%s+%s%s\n", colorGreen, e.line, colorReset)
		} else {
			fmt.Fprintf(w, "+%s\n", e.line)
		}
	}
}

func printUnifiedHunkWordDiff(w io.Writer, lines []edit, opts outputOpts) {
	i := 0
	for i < len(lines) {
		e := lines[i]
		if e.op == opEqual {
			fmt.Fprintf(w, " %s\n", e.line)
			i++
			continue
		}
		// Collect adjacent delete/insert pairs
		var delLines, insLines []string
		for i < len(lines) && lines[i].op == opDelete {
			delLines = append(delLines, lines[i].line)
			i++
		}
		for i < len(lines) && lines[i].op == opInsert {
			insLines = append(insLines, lines[i].line)
			i++
		}
		maxPairs := len(delLines)
		if len(insLines) > maxPairs {
			maxPairs = len(insLines)
		}
		for j := 0; j < maxPairs; j++ {
			if j < len(delLines) && j < len(insLines) {
				oldWd, newWd := wordDiffLine(delLines[j], insLines[j])
				if opts.color {
					fmt.Fprintf(w, "%s-%s%s\n", colorRed, oldWd, colorReset)
					fmt.Fprintf(w, "%s+%s%s\n", colorGreen, newWd, colorReset)
				} else {
					fmt.Fprintf(w, "-%s\n", oldWd)
					fmt.Fprintf(w, "+%s\n", newWd)
				}
			} else if j < len(delLines) {
				printUnifiedLine(w, edit{op: opDelete, line: delLines[j]}, opts)
			} else {
				printUnifiedLine(w, edit{op: opInsert, line: insLines[j]}, opts)
			}
		}
	}
}

// splitWords splits a line into tokens: words and whitespace sequences.
func splitWords(s string) []string {
	var tokens []string
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		if unicode.IsSpace(runes[i]) {
			j := i
			for j < len(runes) && unicode.IsSpace(runes[j]) {
				j++
			}
			tokens = append(tokens, string(runes[i:j]))
			i = j
		} else {
			j := i
			for j < len(runes) && !unicode.IsSpace(runes[j]) {
				j++
			}
			tokens = append(tokens, string(runes[i:j]))
			i = j
		}
	}
	return tokens
}

// wordDiffLine computes word-level diff between two lines and returns
// annotated strings with [- -] and [+ +] markers around changed words.
func wordDiffLine(oldLine, newLine string) (string, string) {
	oldWords := splitWords(oldLine)
	newWords := splitWords(newLine)

	wordEdits := myersDiff(oldWords, newWords)

	var oldBuf, newBuf strings.Builder
	for _, e := range wordEdits {
		switch e.op {
		case opEqual:
			oldBuf.WriteString(e.line)
			newBuf.WriteString(e.line)
		case opDelete:
			oldBuf.WriteString("[-")
			oldBuf.WriteString(e.line)
			oldBuf.WriteString("-]")
		case opInsert:
			newBuf.WriteString("[+")
			newBuf.WriteString(e.line)
			newBuf.WriteString("+]")
		}
	}

	return oldBuf.String(), newBuf.String()
}
