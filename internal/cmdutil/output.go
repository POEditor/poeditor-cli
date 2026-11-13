package cmdutil

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

// Every runnable command declares what it prints, in Annotations[OutputAnnotation]:
// OutputData (it lists or shows data: must have --json, see AddJSONFlag) or
// OutputAction (it changes something and only reports on stderr). The root
// tests fail a command that declares neither, or a data command without --json.
const (
	OutputAnnotation = "poeditor/output"
	OutputData       = "data"
	OutputAction     = "action"
)

// AddJSONFlag adds --json to a data command and marks it as one.
func AddJSONFlag(cmd *cobra.Command, target *bool) {
	cmd.Flags().BoolVar(target, "json", false, "Output JSON")
	annotate(cmd, OutputData)
}

// MarkAction marks a command that changes state and prints no data.
func MarkAction(cmd *cobra.Command) { annotate(cmd, OutputAction) }

func annotate(cmd *cobra.Command, kind string) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[OutputAnnotation] = kind
}

// PrintJSON writes v as indented JSON.
func PrintJSON(ios *iostreams.IOStreams, v any) error {
	enc := json.NewEncoder(ios.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Table prints rows for people or for programs, decided by stdout:
//   - terminal: a header row and aligned columns;
//   - pipe or file: no header, one tab between fields, so `cut -f2` works.
type Table struct {
	ios    *iostreams.IOStreams
	header []string
	rows   [][]string
}

// NewTable starts a table with the given column headers.
func NewTable(ios *iostreams.IOStreams, header ...string) *Table {
	return &Table{ios: ios, header: header}
}

// AddRow adds one row; tabs and newlines inside fields are replaced by spaces
// so they cannot break the columns.
func (t *Table) AddRow(fields ...string) {
	row := make([]string, len(fields))
	for i, field := range fields {
		row[i] = strings.NewReplacer("\t", " ", "\r\n", " ", "\n", " ", "\r", " ").Replace(field)
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
	w := tabwriter.NewWriter(t.ios.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(t.header, "\t"))
	for _, row := range t.rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	return w.Flush()
}
