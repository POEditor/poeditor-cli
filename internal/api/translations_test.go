package api_test

import (
	"context"
	"encoding/json"
	"net/url"
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

func TestTranslationsWrites(t *testing.T) {
	rows := []api.TranslationRow{
		{TermKey: api.TermKey{Term: "save", Context: "button"}, Translation: api.TranslationText{Content: api.Content{Text: "<b>Salvează</b>"}}},
		{TermKey: api.TermKey{Term: "files"}, Translation: api.TranslationText{Content: api.Content{Forms: []api.PluralForm{{Form: "one", Text: "1 fișier"}, {Form: "other", Text: "%d fișiere"}}}}},
	}
	rowsData := `[{"term":"save","context":"button","translation":{"content":"<b>Salvează</b>","fuzzy":0}},` +
		`{"term":"files","context":"","translation":{"content":{"one":"1 fișier","other":"%d fișiere"},"fuzzy":0}}]`
	tests := []struct {
		name     string
		endpoint string
		call     func(*api.Client) (api.Stats, error)
		data     string
		result   string
		want     api.TranslationStats
	}{
		{"add", "translations/add", func(c *api.Client) (api.Stats, error) {
			return c.TranslationsAdd(context.Background(), 7717, "ro", rows)
		}, rowsData, `{"translations":{"parsed":2,"added":1}}`, api.TranslationStats{Parsed: 2, Added: 1}},
		{"update", "translations/update", func(c *api.Client) (api.Stats, error) {
			return c.TranslationsUpdate(context.Background(), 7717, "ro", rows)
		}, rowsData, `{"translations":{"parsed":2,"updated":2}}`, api.TranslationStats{Parsed: 2, Updated: 2}},
		{"delete", "translations/delete", func(c *api.Client) (api.Stats, error) {
			return c.TranslationsDelete(context.Background(), 7717, "ro", []api.TermKey{{Term: "save", Context: "button"}})
		}, `[{"term":"save","context":"button"}]`, `{"translations":{"parsed":1,"deleted":1}}`, api.TranslationStats{Parsed: 1, Deleted: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec recorder
			client := rec.client(t, tt.result)

			stats, err := tt.call(client)

			if err != nil {
				t.Fatal(err)
			}
			rec.assertForm(t, tt.endpoint, url.Values{"id": {"7717"}, "language": {"ro"}, "data": {tt.data}})
			if stats.Translations == nil || !reflect.DeepEqual(*stats.Translations, tt.want) {
				t.Errorf("stats = %+v, want %+v", stats.Translations, tt.want)
			}
		})
	}
}

func TestTranslationsAutomatic(t *testing.T) {
	tests := []struct {
		name string
		opts api.AutomaticOptions
		want url.Values
	}{
		{"targets only", api.AutomaticOptions{
			Provider: api.ProviderDeepL, SourceLanguage: "en", ProviderSourceLanguage: "en",
			Targets: []api.AutomaticTarget{{ProjectLanguage: "pt-br", ProviderLanguage: "pt-BR"}, {ProjectLanguage: "fr", ProviderLanguage: "fr"}},
		}, url.Values{
			"provider": {"deepl"}, "source_language": {"en"}, "provider_source_language": {"en"},
			"target_languages": {`[{"project_language":"pt-br","provider_language":"pt-BR"},{"project_language":"fr","provider_language":"fr"}]`},
		}},
		{"tags, match mode and provider options", api.AutomaticOptions{
			Provider: api.ProviderGoogle, SourceLanguage: "en", ProviderSourceLanguage: "en",
			Targets: []api.AutomaticTarget{{ProjectLanguage: "fr", ProviderLanguage: "fr"}},
			Tags:    []string{"new", "<web>"}, TagsMatch: api.TagsMatchAll, Options: map[string]string{"format": "html"},
		}, url.Values{
			"provider": {"google"}, "source_language": {"en"}, "provider_source_language": {"en"},
			"target_languages": {`[{"project_language":"fr","provider_language":"fr"}]`},
			"options":          {`{"format":"html","tags":["new","<web>"],"tags_match":"all"}`},
		}},
		{"a match mode without tags is not sent", api.AutomaticOptions{
			Provider: api.ProviderGoogle, SourceLanguage: "en", ProviderSourceLanguage: "en", TagsMatch: api.TagsMatchAny,
		}, url.Values{
			"provider": {"google"}, "source_language": {"en"}, "provider_source_language": {"en"}, "target_languages": {`[]`},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec recorder
			client := rec.client(t, `{"successful":[{"source":"en","target":"fr","chars":7}],"failed":[{"source":"en","target":null,"message":"Invalid pair."}]}`)

			result, err := client.TranslationsAutomatic(context.Background(), 7717, tt.opts)

			if err != nil {
				t.Fatal(err)
			}
			want := url.Values{"id": {"7717"}}
			for k, v := range tt.want {
				want[k] = v
			}
			rec.assertForm(t, "translations/automatic", want)
			wantResult := api.AutomaticResult{
				Successful: []api.AutomaticPair{{Source: "en", Target: "fr", Chars: 7}},
				Failed:     []api.AutomaticPair{{Source: "en", Message: "Invalid pair."}},
			}
			if !reflect.DeepEqual(result, wantResult) {
				t.Errorf("result = %+v", result)
			}
		})
	}
}
