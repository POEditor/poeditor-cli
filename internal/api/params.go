package api

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// Helpers that encode Go values the way API v2 expects them in a form.

// idParam is the "id" (project id) parameter.
func idParam(id int) string { return strconv.Itoa(id) }

// jsonParam encodes a list or object parameter (filters, tags, bulk rows) as
// JSON, with HTML as written (<b>, not \u003cb\u003e).
func jsonParam(v any) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		// only called with []string, structs of []string and the bulk row
		// types (TranslationRow, NewTerm…), which always encode
		panic(err)
	}
	return strings.TrimSuffix(b.String(), "\n")
}
