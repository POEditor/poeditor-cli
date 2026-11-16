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
