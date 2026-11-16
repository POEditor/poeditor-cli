package cmdutil

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

// Cell formats shared by every table, so each kind of value looks the same
// in every command.

// TimeCell is a date as a table cell:
//   - terminal: the date, and the time on the line below, so date columns
//     stay narrow (2026-01-14 / 09:12:44); a UTC offset — what API v2
//     sends — is left out, any other is kept (09:12:44+0200);
//   - pipe or file: the API's own format (2026-01-14T09:12:44+0000);
//   - not set: empty.
func TimeCell(ios *iostreams.IOStreams, t api.Time) string {
	if t.IsZero() {
		return ""
	}
	if !ios.IsStdoutTTY() {
		return t.Format(api.TimeLayout)
	}
	return t.Format("2006-01-02\n" + clockLayout(t))
}

// TimeLine is a date on one line, for text that is not a date column
// (2026-03-01 08:00:00; a UTC offset left out, any other kept); "" when not
// set.
func TimeLine(t api.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 " + clockLayout(t))
}

func clockLayout(t api.Time) string {
	if _, offset := t.Zone(); offset != 0 {
		return "15:04:05-0700"
	}
	return "15:04:05"
}

// ListCell is a list of values (tags) as a table cell: joined with ", " on a
// terminal; piped, compact JSON (["a","b, c"]), since a value may itself hold
// a comma. An empty list is an empty cell.
func ListCell(ios *iostreams.IOStreams, values []string) string {
	if len(values) == 0 {
		return ""
	}
	if ios.IsStdoutTTY() {
		return strings.Join(values, ", ")
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(values) // strings always encode
	return strings.TrimSuffix(b.String(), "\n")
}
