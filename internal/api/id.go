package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// ID is a numeric id the server may send as a number or as a string: a
// model's id straight from the database is a string unless the endpoint
// converts it (projects/list does, contributors/list does not).
type ID int

// UnmarshalJSON reads a number or a numeric string.
func (id *ID) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	s := string(data)
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("id: %w", err)
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("id %s: not a number", data)
	}
	*id = ID(n)
	return nil
}
