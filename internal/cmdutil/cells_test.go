package cmdutil_test

import (
	"testing"
	"time"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

func TestTimeCell(t *testing.T) {
	utc := api.Time{Time: time.Date(2026, 1, 14, 9, 12, 44, 0, time.UTC)}
	plus2 := api.Time{Time: time.Date(2026, 2, 1, 10, 0, 0, 0, time.FixedZone("", 7200))}
	tests := []struct {
		name string
		t    api.Time
		tty  bool
		want string
	}{
		{"terminal: time below the date, UTC offset left out", utc, true, "2026-01-14\n09:12:44"},
		{"terminal: another offset is kept", plus2, true, "2026-02-01\n10:00:00+0200"},
		{"piped: the API's format", utc, false, "2026-01-14T09:12:44+0000"},
		{"piped: offset kept", plus2, false, "2026-02-01T10:00:00+0200"},
		{"not set", api.Time{}, true, ""},
		{"not set, piped", api.Time{}, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			if got := cmdutil.TimeCell(ios, tt.t); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTimeLine(t *testing.T) {
	tests := map[string]struct {
		t    api.Time
		want string
	}{
		"utc":        {api.Time{Time: time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)}, "2026-03-01 08:00:00"},
		"other zone": {api.Time{Time: time.Date(2026, 3, 1, 10, 0, 0, 0, time.FixedZone("", 7200))}, "2026-03-01 10:00:00+0200"},
		"not set":    {api.Time{}, ""},
	}
	for name, tt := range tests {
		if got := cmdutil.TimeLine(tt.t); got != tt.want {
			t.Errorf("%s: got %q, want %q", name, got, tt.want)
		}
	}
}

func TestListCell(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		tty    bool
		want   string
	}{
		{"terminal: joined", []string{"web", "v2"}, true, "web, v2"},
		{"piped: JSON, a comma inside a value stays clear", []string{"web", "v2, beta", "<b>"}, false, `["web","v2, beta","<b>"]`},
		{"empty", nil, true, ""},
		{"empty, piped", []string{}, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			if got := cmdutil.ListCell(ios, tt.values); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
