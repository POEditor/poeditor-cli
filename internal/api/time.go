package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// TimeLayout is how API v2 writes dates: PHP's DateTime::ISO8601
// (2026-01-14T09:12:44+0000; no colon in the offset, so Go's RFC 3339
// parsing refuses it).
const TimeLayout = "2006-01-02T15:04:05-0700"

// Time is a date field of the API (created, updated…). The API sends "" for
// a date that is not set (server: isoDate): that is the zero Time, IsZero()
// reports it.
type Time struct{ time.Time }

// UnmarshalJSON reads TimeLayout, "" or null.
func (t *Time) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if string(data) == "null" {
		*t = Time{}
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("date: %w", err)
	}
	if s == "" {
		*t = Time{}
		return nil
	}
	parsed, err := ParseTime(s)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

// MarshalJSON writes the date as the API does, so --json output keeps its
// format; a zero Time is "".
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte(`""`), nil
	}
	return json.Marshal(t.Format(TimeLayout))
}

// ParseTime reads a date in TimeLayout. The offset becomes a fixed zone (UTC
// for +0000), never the machine's local zone, so the same input gives the
// same Time everywhere.
func ParseTime(s string) (Time, error) {
	parsed, err := time.Parse(TimeLayout, s)
	if err != nil {
		return Time{}, fmt.Errorf("date %q: want the API's format, e.g. 2026-01-14T09:12:44+0000", s)
	}
	_, offset := parsed.Zone()
	if offset == 0 {
		return Time{parsed.UTC()}, nil
	}
	return Time{parsed.In(time.FixedZone("", offset))}, nil
}
