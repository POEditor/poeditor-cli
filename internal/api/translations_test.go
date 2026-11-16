package api_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/api"
)

func TestContent_readsBothShapes(t *testing.T) {
	tests := []struct {
		name   string
		json   string
		want   api.Content
		plural bool
		empty  bool
	}{
		{"text", `"Salvează"`, api.Content{Text: "Salvează"}, false, false},
		{"empty text", `""`, api.Content{}, false, true},
		{"null", `null`, api.Content{}, false, true},
		{
			"plural forms keep the API's order", `{"other":"%d files","one":"1 file"}`,
			api.Content{Forms: []api.PluralForm{{Form: "other", Text: "%d files"}, {Form: "one", Text: "1 file"}}}, true, false,
		},
		{"plural, every form empty", `{"one":"","other":""}`, api.Content{Forms: []api.PluralForm{{Form: "one"}, {Form: "other"}}}, true, true},
		{"PHP's empty array: plural, no forms", `[]`, api.Content{Forms: []api.PluralForm{}}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got api.Content
			if err := json.Unmarshal([]byte(tt.json), &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			if got.IsPlural() != tt.plural || got.IsEmpty() != tt.empty {
				t.Errorf("IsPlural = %v, IsEmpty = %v", got.IsPlural(), got.IsEmpty())
			}
		})
	}
}

func TestContent_refusesOtherShapes(t *testing.T) {
	for _, input := range []string{`12`, `true`, `["a"]`, `{"one":1}`} {
		var got api.Content
		if err := json.Unmarshal([]byte(input), &got); err == nil {
			t.Errorf("%s: no error, got %+v", input, got)
		}
	}
}

// Writing gives back the API's own shape, so --json and future writes
// (translations add) send what the API expects; HTML stays as written.
func TestContent_writesTheAPIShape(t *testing.T) {
	for _, s := range []string{`"Salvează"`, `{"other":"%d files","one":"1 file"}`, `{}`, `"a \"b\" <b>&</b>"`, `{"one":"<i>1</i>"}`} {
		var c api.Content
		if err := json.Unmarshal([]byte(s), &c); err != nil {
			t.Fatal(err)
		}
		got, err := c.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != s {
			t.Errorf("%s came back as %s", s, got)
		}
	}
}

func TestTranslation_IsEmpty(t *testing.T) {
	var untranslated, translated api.Translation
	if err := json.Unmarshal([]byte(`{"content":"","fuzzy":0,"proofread":0,"updated":""}`), &untranslated); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"content":"","fuzzy":0,"updated":"2026-03-01T08:00:00+0000"}`), &translated); err != nil {
		t.Fatal(err)
	}
	if !untranslated.IsEmpty() {
		t.Error("never translated: want empty")
	}
	if translated.IsEmpty() {
		t.Error("translated to an empty text (has a date): want not empty")
	}
}
