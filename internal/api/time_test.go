package api_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/POEditor/poeditor-cli/internal/api"
)

func TestTime_unmarshal(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		want   time.Time // zero = not set
		offset int       // seconds east of UTC
	}{
		{"utc", `"2026-01-14T09:12:44+0000"`, time.Date(2026, 1, 14, 9, 12, 44, 0, time.UTC), 0},
		{"other offset kept", `"2026-02-01T10:00:00+0200"`, time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC), 7200},
		{"not set", `""`, time.Time{}, 0},
		{"null", `null`, time.Time{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got api.Time
			if err := json.Unmarshal([]byte(tt.json), &got); err != nil {
				t.Fatal(err)
			}
			if !got.Equal(tt.want) || got.IsZero() != tt.want.IsZero() {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if _, offset := got.Zone(); offset != tt.offset {
				t.Errorf("offset = %d, want %d", offset, tt.offset)
			}
			if !tt.want.IsZero() && got.Location() != time.UTC && tt.offset == 0 {
				t.Errorf("+0000 must be UTC, not %v (the machine's zone would make output differ)", got.Location())
			}
		})
	}
}

func TestTime_unmarshalRefusesOtherFormats(t *testing.T) {
	for _, input := range []string{`"2026-01-14"`, `"yesterday"`, `12`} {
		var got api.Time
		if err := json.Unmarshal([]byte(input), &got); err == nil {
			t.Errorf("%s: no error, got %v", input, got)
		}
	}
}

// --json output keeps the API's format, and "" for a date not set.
func TestTime_marshalLikeTheAPI(t *testing.T) {
	for _, s := range []string{`"2026-01-14T09:12:44+0000"`, `"2026-02-01T10:00:00+0200"`, `""`} {
		var parsed api.Time
		if err := json.Unmarshal([]byte(s), &parsed); err != nil {
			t.Fatal(err)
		}
		got, err := json.Marshal(parsed)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != s {
			t.Errorf("%s came back as %s", s, got)
		}
	}
}
