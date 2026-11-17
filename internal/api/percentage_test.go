package api_test

import (
	"encoding/json"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/api"
)

func TestPercentage_unmarshal(t *testing.T) {
	tests := []struct {
		in   string
		want api.Percentage
	}{
		{`83.33`, 83.33},
		{`100.0`, 100},
		{`99.99`, 99.99},
		{`0`, 0},
		{`"83.33"`, 83.33},
		{`" 50 "`, 50},
		{`""`, 0},
		{`null`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			p := api.Percentage(-1)
			if err := json.Unmarshal([]byte(tt.in), &p); err != nil {
				t.Fatal(err)
			}
			if p != tt.want {
				t.Errorf("got %v, want %v", p, tt.want)
			}
		})
	}
}

func TestPercentage_unmarshalRefuses(t *testing.T) {
	for _, in := range []string{`"n/a"`, `true`, `[]`, `{}`} {
		var p api.Percentage
		if err := json.Unmarshal([]byte(in), &p); err == nil {
			t.Errorf("%s: no error, got %v", in, p)
		}
	}
}

func TestPercentage_string(t *testing.T) {
	for p, want := range map[api.Percentage]string{83.33: "83.33%", 100: "100%", 0: "0%", 99.99: "99.99%", 12.5: "12.5%"} {
		if got := p.String(); got != want {
			t.Errorf("%v: got %q, want %q", float64(p), got, want)
		}
	}
}
