package cmdutil

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ReadJSONList reads a --from-file of bulk input: one JSON list of at least
// one item, in the API's shape. A field the type does not know is an error
// (a misspelled "contents" would otherwise be dropped silently), and check,
// when not nil, vets each item. Errors name the file and the item (1-based),
// so it is all checked before any API call.
func ReadJSONList[T any](f *Factory, path string, check func(T) error) ([]T, error) {
	data, err := f.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("--from-file: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var items []T
	if err := dec.Decode(&items); err != nil {
		return nil, fmt.Errorf("--from-file %s: %w (want a JSON list: see --help)", path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("--from-file %s: more than one JSON value (want one list: see --help)", path)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("--from-file %s: the list is empty", path)
	}
	if check != nil {
		for i, item := range items {
			if err := check(item); err != nil {
				return nil, fmt.Errorf("--from-file %s: entry %d: %w", path, i+1, err)
			}
		}
	}
	return items, nil
}
