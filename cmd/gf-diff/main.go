package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gf-diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "show version")
	unified := fs.Bool("u", false, "unified diff format")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "gf-diff version "+version)
		return 0
	}

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

	if *unified {
		printUnified(stdout, file1, file2, edits, 3)
	} else {
		for _, e := range edits {
			switch e.op {
			case opDelete:
				fmt.Fprintf(stdout, "< %s\n", e.line)
			case opInsert:
				fmt.Fprintf(stdout, "> %s\n", e.line)
			case opEqual:
				fmt.Fprintf(stdout, "  %s\n", e.line)
			}
		}
	}

	return 1
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
func printUnified(w io.Writer, file1, file2 string, edits []edit, contextLines int) {
	fmt.Fprintf(w, "--- %s\n", file1)
	fmt.Fprintf(w, "+++ %s\n", file2)

	hunks := buildHunks(edits, contextLines)
	for _, h := range hunks {
		fmt.Fprintf(w, "@@ -%d,%d +%d,%d @@\n", h.oldStart, h.oldCount, h.newStart, h.newCount)
		for _, e := range h.lines {
			switch e.op {
			case opEqual:
				fmt.Fprintf(w, " %s\n", e.line)
			case opDelete:
				fmt.Fprintf(w, "-%s\n", e.line)
			case opInsert:
				fmt.Fprintf(w, "+%s\n", e.line)
			}
		}
	}
}
