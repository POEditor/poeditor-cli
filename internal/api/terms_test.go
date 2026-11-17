package api_test

import (
	"context"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/POEditor/poeditor-cli/internal/api"
)

func TestTermsList_withoutLanguage(t *testing.T) {
	var rec recorder
	client := rec.client(t, `{"terms":[{"term":"welcome","context":"home","plural":"","created":"2026-01-14T09:12:44+0000","updated":"","reference":"src/home.js","tags":["web","v2"],"comment":"Shown once"}]}`)

	terms, err := client.TermsList(context.Background(), 7717, "")

	if err != nil {
		t.Fatal(err)
	}
	rec.assertForm(t, "terms/list", url.Values{"id": {"7717"}})
	want := []api.Term{{
		TermKey:   api.TermKey{Term: "welcome", Context: "home"},
		Reference: "src/home.js",
		Comment:   "Shown once",
		Tags:      []string{"web", "v2"},
		Created:   api.Time{Time: time.Date(2026, 1, 14, 9, 12, 44, 0, time.UTC)},
	}}
	if !reflect.DeepEqual(terms, want) {
		t.Errorf("terms = %+v\nwant    %+v", terms, want)
	}
	if terms[0].IsPlural() || terms[0].Translation != nil {
		t.Errorf("a normal term without language: plural %v, translation %+v", terms[0].IsPlural(), terms[0].Translation)
	}
}

func TestTermsList_withLanguage(t *testing.T) {
	var rec recorder
	client := rec.client(t, `{"terms":[`+
		`{"term":"save","context":"","plural":"","created":"2026-01-14T09:12:44+0000","updated":"","reference":"","tags":[],"comment":"",`+
		`"translation":{"content":"Salvează","fuzzy":1,"proofread":0,"updated":"2026-03-01T08:00:00+0000"}},`+
		`{"term":"files","context":"","plural":"files_plural","created":"2026-01-14T09:12:44+0000","updated":"","reference":"","tags":[],"comment":"",`+
		`"translation":{"content":{"one":"1 fișier","few":"%d fișiere"},"fuzzy":0,"updated":""}}]}`)

	terms, err := client.TermsList(context.Background(), 7717, "ro")

	if err != nil {
		t.Fatal(err)
	}
	rec.assertForm(t, "terms/list", url.Values{"id": {"7717"}, "language": {"ro"}})

	zero := 0
	save := terms[0].Translation
	if save == nil || !reflect.DeepEqual(save.Content, api.Content{Text: "Salvează"}) || save.Fuzzy != 1 ||
		save.Proofread == nil || *save.Proofread != zero || save.Updated.IsZero() {
		t.Errorf("save translation = %+v", save)
	}

	files := terms[1]
	if !files.IsPlural() {
		t.Error("files: want a plural term")
	}
	tr := files.Translation
	wantForms := []api.PluralForm{{Form: "one", Text: "1 fișier"}, {Form: "few", Text: "%d fișiere"}}
	if tr == nil || !reflect.DeepEqual(tr.Content.Forms, wantForms) {
		t.Fatalf("files translation = %+v", tr)
	}
	if tr.Proofread != nil {
		t.Errorf("proofreading off: proofread = %v, want nil", *tr.Proofread)
	}
}

func strp(s string) *string { return &s }

// Each bulk term endpoint sends its rows as "data", in the API's field names,
// and returns the term counts.
func TestTermsWrites(t *testing.T) {
	tags := []string{}
	tests := []struct {
		name     string
		endpoint string
		call     func(*api.Client) (api.Stats, error)
		data     string
		extra    url.Values
		result   string
		want     api.TermStats
	}{
		{"add", "terms/add", func(c *api.Client) (api.Stats, error) {
			return c.TermsAdd(context.Background(), 7717, []api.NewTerm{
				{TermKey: api.TermKey{Term: "save", Context: "button"}, Plural: "saves", Reference: "a.js:1", Comment: "<b>x</b>", Tags: []string{"web"}},
				{TermKey: api.TermKey{Term: "bare"}},
			})
		}, `[{"term":"save","context":"button","plural":"saves","reference":"a.js:1","comment":"<b>x</b>","tags":["web"]},{"term":"bare","context":""}]`,
			nil, `{"terms":{"parsed":2,"added":1}}`, api.TermStats{Parsed: 2, Added: 1}},
		{"update", "terms/update", func(c *api.Client) (api.Stats, error) {
			return c.TermsUpdate(context.Background(), 7717, []api.TermUpdate{
				{TermKey: api.TermKey{Term: "save"}, NewTerm: strp("store"), NewContext: strp(""), Reference: strp(""), Plural: strp("stores"), Tags: &tags},
				{TermKey: api.TermKey{Term: "only-key", Context: "c"}},
			}, api.TermsUpdateOptions{})
		}, `[{"term":"save","context":"","new_term":"store","new_context":"","reference":"","plural":"stores","tags":[]},{"term":"only-key","context":"c"}]`,
			nil, `{"terms":{"parsed":2,"updated":2}}`, api.TermStats{Parsed: 2, Updated: 2}},
		{"update with fuzzy trigger", "terms/update", func(c *api.Client) (api.Stats, error) {
			return c.TermsUpdate(context.Background(), 7717, []api.TermUpdate{{TermKey: api.TermKey{Term: "save"}, Plural: strp("saves")}}, api.TermsUpdateOptions{FuzzyTrigger: true})
		}, `[{"term":"save","context":"","plural":"saves"}]`,
			url.Values{"fuzzy_trigger": {"1"}}, `{"terms":{"parsed":1,"updated":1}}`, api.TermStats{Parsed: 1, Updated: 1}},
		{"delete", "terms/delete", func(c *api.Client) (api.Stats, error) {
			return c.TermsDelete(context.Background(), 7717, []api.TermKey{{Term: "save", Context: "button"}})
		}, `[{"term":"save","context":"button"}]`,
			nil, `{"terms":{"parsed":1,"deleted":1}}`, api.TermStats{Parsed: 1, Deleted: 1}},
		{"add comment", "terms/add_comment", func(c *api.Client) (api.Stats, error) {
			return c.TermsAddComment(context.Background(), 7717, []api.TermComment{
				{TermKey: api.TermKey{Term: "save"}, Comment: "Check length", Pinned: 1},
				{TermKey: api.TermKey{Term: "open"}, Comment: "ok"},
			})
		}, `[{"term":"save","context":"","comment":"Check length","pinned":1},{"term":"open","context":"","comment":"ok"}]`,
			nil, `{"terms":{"parsed":2,"with_added_comment":2}}`, api.TermStats{Parsed: 2, WithAddedComment: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rec recorder
			client := rec.client(t, tt.result)

			stats, err := tt.call(client)

			if err != nil {
				t.Fatal(err)
			}
			want := url.Values{"id": {"7717"}, "data": {tt.data}}
			for k, v := range tt.extra {
				want[k] = v
			}
			rec.assertForm(t, tt.endpoint, want)
			if stats.Terms == nil || *stats.Terms != tt.want {
				t.Errorf("stats = %+v, want %+v", stats.Terms, tt.want)
			}
		})
	}
}

func TestTermsWrites_nothingToSend(t *testing.T) {
	var rec recorder
	client := rec.client(t, `null`)

	if _, err := client.TermsDelete(context.Background(), 7717, nil); err == nil {
		t.Error("no error for an empty list")
	}
	if rec.path != "" {
		t.Errorf("a request was sent: %s", rec.path)
	}
}

func TestTermWriteChecks(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string // "" = valid
	}{
		{"key", api.TermKey{Term: "save"}.Check(), ""},
		{"key without term", api.TermKey{Context: "c"}.Check(), `no "term"`},
		{"key with a blank term", api.TermKey{Term: " \n"}.Check(), `no "term"`},
		{"update", api.TermUpdate{TermKey: api.TermKey{Term: "save"}, NewTerm: strp("store")}.Check(), ""},
		{"update without term", api.TermUpdate{NewTerm: strp("store")}.Check(), `no "term"`},
		{"update to an empty term", api.TermUpdate{TermKey: api.TermKey{Term: "save"}, NewTerm: strp(" ")}.Check(), `"new_term" is empty`},
		{"comment", api.TermComment{TermKey: api.TermKey{Term: "save"}, Comment: "x"}.Check(), ""},
		{"comment without text", api.TermComment{TermKey: api.TermKey{Term: "save"}, Comment: "  "}.Check(), `no "comment"`},
		{"comment without term", api.TermComment{Comment: "x"}.Check(), `no "term"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ""
			if tt.err != nil {
				got = tt.err.Error()
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
