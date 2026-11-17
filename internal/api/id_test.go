package api_test

import (
	"encoding/json"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/api"
)

func TestID(t *testing.T) {
	for in, want := range map[string]api.ID{`7717`: 7717, `"7717"`: 7717, `" 1"`: 0} {
		var id api.ID
		err := json.Unmarshal([]byte(in), &id)
		if in == `" 1"` {
			if err == nil {
				t.Errorf("%s: no error", in)
			}
			continue
		}
		if err != nil || id != want {
			t.Errorf("%s: got %d, %v; want %d", in, id, err, want)
		}
	}
	for _, in := range []string{`null`, `""`, `"x"`, `1.5`, `true`} {
		var id api.ID
		if err := json.Unmarshal([]byte(in), &id); err == nil {
			t.Errorf("%s: no error, got %d", in, id)
		}
	}
}
