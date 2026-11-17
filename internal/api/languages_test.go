package api_test

import (
	"context"
	"encoding/json"
	"net/url"
	"reflect"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/api"
)

func TestLanguagesList(t *testing.T) {
	var rec recorder
	client := rec.client(t, `{"languages":[{"name":"French","code":"fr","translations":10,"percentage":83.33,"updated":"2024-01-02T03:04:05+0000"}]}`)

	langs, err := client.LanguagesList(context.Background(), 7717)

	if err != nil {
		t.Fatal(err)
	}
	rec.assertForm(t, "languages/list", url.Values{"id": {"7717"}})
	updated, _ := api.ParseTime("2024-01-02T03:04:05+0000")
	if want := []api.Language{{LanguageRef: api.LanguageRef{Name: "French", Code: "fr"}, Translations: 10, Percentage: 83.33, Updated: updated}}; !reflect.DeepEqual(langs, want) {
		t.Errorf("languages = %+v", langs)
	}
}

func TestLanguagesAvailable(t *testing.T) {
	var rec recorder
	client := rec.client(t, `{"languages":[{"name":"English","code":"en"},{"name":"Portuguese (BR)","code":"pt-br"}]}`)

	langs, err := client.LanguagesAvailable(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	rec.assertForm(t, "languages/available", url.Values{})
	if want := []api.LanguageRef{{Name: "English", Code: "en"}, {Name: "Portuguese (BR)", Code: "pt-br"}}; !reflect.DeepEqual(langs, want) {
		t.Errorf("languages = %+v", langs)
	}
}

func TestLanguageCodes_sorted(t *testing.T) {
	var rec recorder
	client := rec.client(t, `{"languages":[{"code":"fr"},{"code":"de"},{"code":"en"}]}`)

	codes, err := client.LanguageCodes(context.Background(), 7717)

	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"de", "en", "fr"}; !reflect.DeepEqual(codes, want) {
		t.Errorf("codes = %v", codes)
	}
}

func TestLanguagesAdd(t *testing.T) {
	var rec recorder
	client := rec.client(t, `null`)

	if err := client.LanguagesAdd(context.Background(), 7717, "fr"); err != nil {
		t.Fatal(err)
	}
	rec.assertForm(t, "languages/add", url.Values{"id": {"7717"}, "language": {"fr"}})
}

func TestLanguagesDelete(t *testing.T) {
	var rec recorder
	client := rec.client(t, `null`)

	if err := client.LanguagesDelete(context.Background(), 7717, "fr"); err != nil {
		t.Fatal(err)
	}
	rec.assertForm(t, "languages/delete", url.Values{"id": {"7717"}, "language": {"fr"}})
}

func TestLanguagesUpdate(t *testing.T) {
	rows := []api.TranslationRow{
		{TermKey: api.TermKey{Term: "save", Context: "button"}, Translation: api.TranslationText{Content: api.Content{Text: "<b>Enregistrer</b>"}, Fuzzy: 1}},
		{TermKey: api.TermKey{Term: "files"}, Translation: api.TranslationText{Content: api.Content{Forms: []api.PluralForm{{Form: "one", Text: "1 fichier"}, {Form: "other", Text: "%d fichiers"}}}}},
	}
	tests := []struct {
		name string
		opts api.LanguageUpdateOptions
		want url.Values
	}{
		{"plain", api.LanguageUpdateOptions{}, url.Values{}},
		{"fuzzy trigger", api.LanguageUpdateOptions{FuzzyTrigger: true}, url.Values{"fuzzy_trigger": {"1"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec recorder
			client := rec.client(t, `{"translations":{"parsed":2,"added":1,"updated":1}}`)

			stats, err := client.LanguagesUpdate(context.Background(), 7717, "fr", rows, tt.opts)

			if err != nil {
				t.Fatal(err)
			}
			want := url.Values{"id": {"7717"}, "language": {"fr"}}
			for k, v := range tt.want {
				want[k] = v
			}
			want["data"] = rec.form["data"]
			rec.assertForm(t, "languages/update", want)
			var sent []map[string]any
			if err := json.Unmarshal([]byte(rec.form.Get("data")), &sent); err != nil {
				t.Fatalf("data is not JSON: %v", err)
			}
			wantData := `[{"term":"save","context":"button","translation":{"content":"<b>Enregistrer</b>","fuzzy":1}},` +
				`{"term":"files","context":"","translation":{"content":{"one":"1 fichier","other":"%d fichiers"},"fuzzy":0}}]`
			var wantSent []map[string]any
			_ = json.Unmarshal([]byte(wantData), &wantSent)
			if !reflect.DeepEqual(sent, wantSent) {
				t.Errorf("data = %s\nwant   %s", rec.form.Get("data"), wantData)
			}
			if tr := stats.Translations; tr == nil || tr.Parsed != 2 || tr.Added != 1 || tr.Updated != 1 {
				t.Errorf("stats = %+v", stats.Translations)
			}
		})
	}
}

func TestLanguagesUpdate_noRows(t *testing.T) {
	var rec recorder
	client := rec.client(t, `null`)

	_, err := client.LanguagesUpdate(context.Background(), 7717, "fr", nil, api.LanguageUpdateOptions{})

	if err == nil {
		t.Fatal("no error for an empty list")
	}
	if rec.path != "" {
		t.Errorf("a request was sent: %s", rec.path)
	}
}
