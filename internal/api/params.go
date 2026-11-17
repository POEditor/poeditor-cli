package api

import (
	"encoding/json"
	"strconv"
)

// Helpers that encode Go values the way API v2 expects them in a form.

// idParam is the "id" (project id) parameter.
func idParam(id int) string { return strconv.Itoa(id) }

// jsonParam encodes a list or object parameter (filters, tags) as JSON.
func jsonParam(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		// only called with []string, structs of []string and TranslationRow
		// lists, which always encode
		panic(err)
	}
	return string(b)
}
