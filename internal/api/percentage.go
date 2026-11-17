package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Percentage is a progress percentage of the API (languages/list
// "percentage"). The server builds it with ProjectStats::getProgressPercentage,
// typed float|int|string: round(…, 2) gives a float (83.33, 99.99, 100.0), a
// project without terms gives the int 0, and the declared string is read too
// ("83.33"). "" and null are 0.
type Percentage float64

// UnmarshalJSON reads a number, a numeric string, "" or null.
func (p *Percentage) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if string(data) == "null" {
		*p = 0
		return nil
	}
	var s string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("percentage: %w", err)
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*p = 0
			return nil
		}
	} else {
		s = string(data)
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("percentage %s: not a number", data)
	}
	*p = Percentage(f)
	return nil
}

// String is the percentage as people write it: 83.33%, 100%, 0%.
func (p Percentage) String() string {
	return strconv.FormatFloat(float64(p), 'f', -1, 64) + "%"
}
