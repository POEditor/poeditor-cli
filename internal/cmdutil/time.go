package cmdutil

import (
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

// TimeCell is a date as a table cell, the same in every command:
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
	clock := "15:04:05"
	if _, offset := t.Zone(); offset != 0 {
		clock += "-0700"
	}
	return t.Format("2006-01-02\n" + clock)
}
