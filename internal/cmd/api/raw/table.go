package raw

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// printTable shows any API result as a table, since `poeditor api` cannot
// know its shape in advance:
//   - a result holding one key ({"projects": [...]}) shows that key's value;
//   - a list of objects: one row per object, one column per key (keys in
//     the order the API sent them, the union over all rows);
//   - an object: the same table with one row, so `projects view` reads like
//     `projects list`;
//   - a list of plain values: one VALUE column; a plain value: itself.
//
// Nested objects and lists appear as compact JSON in their cell. On a
// terminal, cells are made for reading (see terminalCell).
func printTable(f *cmdutil.Factory, raw json.RawMessage) error {
	raw = bytes.TrimSpace(raw)
	if isNull(raw) {
		return nil
	}
	if keys, values, ok := objectFields(raw); ok && len(keys) == 1 {
		raw = values[keys[0]]
	}

	switch raw[0] {
	case '[':
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return fmt.Errorf("the API returned invalid JSON: %w", err)
		}
		return printList(f, items)
	case '{':
		return printList(f, []json.RawMessage{raw})
	default:
		_, err := fmt.Fprintln(f.IO.Out, cell(raw))
		return err
	}
}

func printList(f *cmdutil.Factory, items []json.RawMessage) error {
	if len(items) == 0 {
		if f.IO.IsStdoutTTY() {
			fmt.Fprintln(f.IO.Out, "No results.")
		}
		return nil
	}

	var columns []string
	seen := map[string]bool{}
	rows := make([]map[string]json.RawMessage, len(items))
	for i, item := range items {
		keys, values, ok := objectFields(item)
		if !ok { // not a list of objects: one value per row
			return printValues(f, items)
		}
		rows[i] = values
		for _, key := range keys {
			if !seen[key] {
				seen[key] = true
				columns = append(columns, key)
			}
		}
	}

	header := make([]string, len(columns))
	for i, c := range columns {
		header[i] = strings.ToUpper(c)
	}
	table := cmdutil.NewTable(f.IO, header...)
	for _, row := range rows {
		fields := make([]string, len(columns))
		for i, c := range columns {
			if f.IO.IsStdoutTTY() {
				fields[i] = terminalCell(row[c])
			} else {
				fields[i] = cell(row[c])
			}
		}
		table.AddRow(fields...)
	}
	return table.Render()
}

func printValues(f *cmdutil.Factory, items []json.RawMessage) error {
	table := cmdutil.NewTable(f.IO, "VALUE")
	for _, item := range items {
		table.AddRow(cell(item))
	}
	return table.Render()
}

// objectFields parses a JSON object keeping the API's key order; ok is false
// when raw is not an object.
func objectFields(raw json.RawMessage) (keys []string, values map[string]json.RawMessage, ok bool) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, nil, false
	}
	values = map[string]json.RawMessage{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, nil, false
		}
		key := tok.(string) // object keys are always strings
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, nil, false
		}
		if _, dup := values[key]; !dup {
			keys = append(keys, key)
		}
		values[key] = value
	}
	return keys, values, true
}

// cell is a value as table text: strings unquoted, null empty, numbers and
// booleans as sent, objects and lists as compact, readable JSON.
func cell(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	switch {
	case isNull(raw):
		return ""
	case raw[0] == '"':
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return s
		}
	case raw[0] == '{' || raw[0] == '[':
		if text, err := readable(raw); err == nil {
			return string(text)
		}
	}
	return string(raw)
}

// terminalCell is a value as a person reads it: a list of plain values
// (tags) joined with ", ", and a date-time on two lines. Piped output keeps
// cell's form, which scripts can parse (a tag may itself contain a comma).
func terminalCell(raw json.RawMessage) string {
	if joined, ok := joinValues(raw); ok {
		return joined
	}
	return splitDateTime(cell(raw))
}

// joinValues joins a list of strings, numbers or booleans with ", "; ok is
// false for anything else (an object, a list holding objects or lists).
func joinValues(raw json.RawMessage) (string, bool) {
	var items []json.RawMessage
	if json.Unmarshal(raw, &items) != nil {
		return "", false
	}
	values := make([]string, len(items))
	for i, item := range items {
		if c := bytes.TrimSpace(item); len(c) > 0 && (c[0] == '{' || c[0] == '[') {
			return "", false
		}
		values[i] = cell(item)
	}
	return strings.Join(values, ", "), true
}

// dateTime is how API v2 sends dates: 2026-01-14T09:12:44+0000.
var dateTime = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2}:\d{2})([+-]\d{2}:?\d{2}|Z)?$`)

// splitDateTime puts a date-time's time on the line below its date, so date
// columns stay narrow. A UTC offset (which API v2 always sends) is left out;
// any other offset is kept. Text that is not exactly a date-time is returned
// as it is.
func splitDateTime(s string) string {
	m := dateTime.FindStringSubmatch(s)
	if m == nil {
		return s
	}
	switch m[3] {
	case "", "Z", "+0000", "+00:00", "-0000", "-00:00":
		return m[1] + "\n" + m[2]
	}
	return m[1] + "\n" + m[2] + m[3]
}

func isNull(raw json.RawMessage) bool {
	return len(raw) == 0 || string(raw) == "null"
}
