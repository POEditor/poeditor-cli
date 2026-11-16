package cmdutil

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"

	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

const (
	// columnGap is the space between two columns on a terminal.
	columnGap = 2
	// minColumnWidth is the narrowest a shrunk column gets.
	minColumnWidth = 3
	// ellipsis ends a cut header.
	ellipsis = "…"
)

// Table prints rows for people or for programs, decided by stdout:
//   - terminal: a header row and aligned columns, fitted to the terminal
//     width: short columns keep their width, the long ones share what is
//     left and their cells wrap over several lines (at spaces; inside a word
//     only when the word alone is too wide), so nothing is hidden. Line
//     breaks in a field are kept. When any row spans several lines, a blank
//     line separates the rows;
//   - pipe or file: no header, one tab between fields, nothing cut, so
//     `cut -f2` gets the real values.
//
// Widths are display widths, so CJK text (two columns per character) lines
// up.
type Table struct {
	ios    *iostreams.IOStreams
	header []string
	rows   [][]string
}

// NewTable starts a table with the given column headers.
func NewTable(ios *iostreams.IOStreams, header ...string) *Table {
	return &Table{ios: ios, header: header}
}

var (
	// pipeCleaner turns tabs and line breaks into spaces, so a field cannot
	// break the columns or the rows of piped output.
	pipeCleaner = strings.NewReplacer("\t", " ", "\r\n", " ", "\n", " ", "\r", " ")
	// terminalCleaner keeps line breaks (one "\n" each), since a terminal
	// cell spans several lines anyway.
	terminalCleaner = strings.NewReplacer("\t", " ", "\r\n", "\n", "\r", "\n")
)

// AddRow adds one row.
func (t *Table) AddRow(fields ...string) {
	cleaner := pipeCleaner
	if t.ios.IsStdoutTTY() {
		cleaner = terminalCleaner
	}
	row := make([]string, len(fields))
	for i, field := range fields {
		row[i] = cleaner.Replace(field)
	}
	t.rows = append(t.rows, row)
}

// Render writes the table to stdout.
func (t *Table) Render() error {
	if !t.ios.IsStdoutTTY() {
		for _, row := range t.rows {
			if _, err := fmt.Fprintln(t.ios.Out, strings.Join(row, "\t")); err != nil {
				return err
			}
		}
		return nil
	}

	lines := append([][]string{t.header}, t.rows...)
	widths := fitColumns(naturalWidths(lines), t.ios.TerminalWidth())
	rows := make([][]string, len(t.rows))
	wrapped := false
	for i, row := range t.rows {
		rows[i] = rowLines(row, widths)
		wrapped = wrapped || len(rows[i]) > 1
	}
	var b strings.Builder
	b.WriteString(formatLine(t.header, widths)) // a header too wide is cut
	b.WriteByte('\n')
	for i, lines := range rows {
		if wrapped && i > 0 {
			b.WriteByte('\n') // tells rows apart once some span several lines
		}
		for _, line := range lines {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	_, err := fmt.Fprint(t.ios.Out, b.String())
	return err
}

// naturalWidths is each column's widest cell line, header included.
func naturalWidths(lines [][]string) []int {
	var widths []int
	for _, line := range lines {
		for i, cell := range line {
			if i == len(widths) {
				widths = append(widths, 0)
			}
			for _, line := range strings.Split(cell, "\n") {
				widths[i] = max(widths[i], runewidth.StringWidth(line))
			}
		}
	}
	return widths
}

// fitColumns shrinks natural widths to fit total columns (0 = no limit).
// The space is shared fairly: a column narrower than its share keeps its
// width and leaves the rest to the others, repeatedly; the columns still too
// wide split what remains (never below minColumnWidth, so a very narrow
// terminal still shows a few characters per line).
func fitColumns(natural []int, total int) []int {
	widths := append([]int(nil), natural...)
	if total <= 0 || len(widths) == 0 {
		return widths
	}
	available := total - columnGap*(len(widths)-1)
	if sum(widths) <= available {
		return widths
	}

	wide := make([]bool, len(widths)) // still to be cut
	for i := range wide {
		wide[i] = true
	}
	for {
		remaining, count := available, 0
		for i, w := range widths {
			if wide[i] {
				count++
			} else {
				remaining -= w
			}
		}
		if count == 0 {
			return widths
		}
		share := remaining / count
		settled := false
		for i, w := range natural {
			if wide[i] && w <= share {
				wide[i], widths[i], settled = false, w, true
			}
		}
		if settled {
			continue
		}
		extra := remaining - share*count // the remainder goes to the first ones
		for i := range widths {
			if wide[i] {
				widths[i] = max(share, minColumnWidth)
				if extra > 0 {
					widths[i]++
					extra--
				}
			}
		}
		return widths
	}
}

// rowLines is one row as printed: as many lines as its tallest cell, the
// other cells blank on the extra lines.
func rowLines(row []string, widths []int) []string {
	cells := make([][]string, len(row))
	height := 1
	for i, cell := range row {
		cells[i] = wrap(cell, widths[i])
		height = max(height, len(cells[i]))
	}
	lines := make([]string, height)
	line := make([]string, len(row))
	for l := range lines {
		for i := range cells {
			line[i] = ""
			if l < len(cells[i]) {
				line[i] = cells[i][l]
			}
		}
		lines[l] = formatLine(line, widths)
	}
	return lines
}

// wrap breaks text into lines at most width display columns wide: at spaces,
// and inside a word only when the word alone is wider than the column. Line
// breaks in the text are kept; a line that fits is left as it is.
func wrap(text string, width int) []string {
	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		if runewidth.StringWidth(paragraph) <= width {
			lines = append(lines, paragraph)
			continue
		}
		line, lineWidth := "", 0
		for _, word := range strings.Fields(paragraph) {
			for _, part := range splitWord(word, width) {
				w := runewidth.StringWidth(part)
				switch {
				case lineWidth == 0:
					line, lineWidth = part, w
				case lineWidth+1+w <= width:
					line, lineWidth = line+" "+part, lineWidth+1+w
				default:
					lines = append(lines, line)
					line, lineWidth = part, w
				}
			}
		}
		lines = append(lines, line)
	}
	return lines
}

// splitWord cuts a word wider than width into pieces that fit.
func splitWord(word string, width int) []string {
	var parts []string
	for runewidth.StringWidth(word) > width {
		head := runewidth.Truncate(word, width, "")
		if head == "" { // a character wider than the column: it goes alone
			_, size := utf8.DecodeRuneInString(word)
			head = word[:size]
		}
		parts = append(parts, head)
		word = word[len(head):]
	}
	return append(parts, word)
}

// formatLine pads (or cuts) each cell to its width; the last cell is not
// padded, so no line ends in spaces.
func formatLine(line []string, widths []int) string {
	var b strings.Builder
	for i, cell := range line {
		if runewidth.StringWidth(cell) > widths[i] {
			cell = runewidth.Truncate(cell, widths[i], ellipsis)
		}
		b.WriteString(cell)
		if i < len(line)-1 {
			b.WriteString(strings.Repeat(" ", widths[i]-runewidth.StringWidth(cell)+columnGap))
		}
	}
	return strings.TrimRight(b.String(), " ")
}

func sum(values []int) int {
	total := 0
	for _, v := range values {
		total += v
	}
	return total
}
