package api_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/api"
)

// recorder answers every request with body and keeps the last request's
// path and form fields (url-encoded or multipart).
type recorder struct {
	path string
	form url.Values
	file string
}

func (rec *recorder) client(t *testing.T, body string) *api.Client {
	return server(t, func(w http.ResponseWriter, r *http.Request) {
		rec.path = r.URL.Path
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			_ = r.ParseForm()
		}
		rec.form = r.PostForm
		if r.MultipartForm != nil {
			rec.form = r.MultipartForm.Value
			if fh := r.MultipartForm.File["file"]; len(fh) == 1 {
				f, _ := fh[0].Open()
				data, _ := io.ReadAll(f)
				rec.file = fh[0].Filename + ":" + string(data)
			}
		}
		io.WriteString(w, `{"response":{"status":"success","code":"200","message":"OK"},"result":`+body+`}`)
	})
}

func (rec *recorder) assertForm(t *testing.T, path string, want url.Values) {
	t.Helper()
	if rec.path != "/v2/"+path {
		t.Errorf("path = %q, want /v2/%s", rec.path, path)
	}
	got := url.Values{}
	for k, v := range rec.form {
		if k != "api_token" {
			got[k] = v
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("form = %v\nwant   %v", got, want)
	}
}

func TestProjectsList(t *testing.T) {
	var rec recorder
	client := rec.client(t, `{"projects":[{"id":7717,"name":"Web","public":0,"open":1,"created":"2024-01-02T03:04:05+0000"}]}`)

	projects, err := client.ProjectsList(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	rec.assertForm(t, "projects/list", url.Values{})
	want := []api.Project{{ID: 7717, Name: "Web", Open: 1, Created: "2024-01-02T03:04:05+0000"}}
	if !reflect.DeepEqual(projects, want) {
		t.Errorf("projects = %+v", projects)
	}
}

func TestProjectsView(t *testing.T) {
	var rec recorder
	client := rec.client(t, `{"project":{"id":7717,"name":"Web","reference_language":"en","fallback_language":"","terms":12}}`)

	p, err := client.ProjectsView(context.Background(), 7717)

	if err != nil {
		t.Fatal(err)
	}
	rec.assertForm(t, "projects/view", url.Values{"id": {"7717"}})
	if p.ID != 7717 || p.Name != "Web" || p.ReferenceLanguage != "en" || p.Terms != 12 {
		t.Errorf("project = %+v", p)
	}
}

func TestProjectsExport(t *testing.T) {
	cases := []struct {
		name string
		opts api.ExportOptions
		want url.Values
	}{
		{
			name: "only the required parameters",
			opts: api.ExportOptions{Language: "fr", Type: "po"},
			want: url.Values{"id": {"7717"}, "language": {"fr"}, "type": {"po"}},
		},
		{
			name: "every option",
			opts: api.ExportOptions{Language: "fr", Type: "po", Filters: []string{"translated", "not_fuzzy"},
				Tags: []string{"web", `a"b`}, TagsMatch: "any", Order: "terms", FallbackLanguage: "en"},
			want: url.Values{"id": {"7717"}, "language": {"fr"}, "type": {"po"},
				"filters": {`["translated","not_fuzzy"]`}, "tags": {`["web","a\"b"]`}, "tags_match": {"any"},
				"order": {"terms"}, "fallback_language": {"en"}},
		},
		{
			name: "tags_match is not sent without tags",
			opts: api.ExportOptions{Language: "fr", Type: "po", TagsMatch: "any"},
			want: url.Values{"id": {"7717"}, "language": {"fr"}, "type": {"po"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rec recorder
			client := rec.client(t, `{"url":"https://api.poeditor.com/v2/download/file/abc"}`)

			link, err := client.ProjectsExport(context.Background(), 7717, tc.opts)

			if err != nil {
				t.Fatal(err)
			}
			rec.assertForm(t, "projects/export", tc.want)
			if link != "https://api.poeditor.com/v2/download/file/abc" {
				t.Errorf("link = %q", link)
			}
		})
	}
}

func TestProjectsExport_noLink(t *testing.T) {
	var rec recorder
	client := rec.client(t, `{"url":""}`)

	_, err := client.ProjectsExport(context.Background(), 7717, api.ExportOptions{Language: "fr", Type: "po"})

	if err == nil || err.Error() != "the API returned no download link" {
		t.Errorf("err = %v", err)
	}
}

func TestProjectsUpload(t *testing.T) {
	cases := []struct {
		name string
		opts api.UploadOptions
		want url.Values
	}{
		{
			name: "only the required parameters",
			opts: api.UploadOptions{Updating: "terms_translations", Language: "en"},
			want: url.Values{"id": {"7717"}, "updating": {"terms_translations"}, "language": {"en"}},
		},
		{
			name: "every option",
			opts: api.UploadOptions{Updating: "translations", Language: "fr", Overwrite: true, SyncTerms: true,
				ReadFromSource: true, FuzzyTrigger: true,
				Tags: api.UploadTags{All: []string{"web"}, Obsolete: []string{"gone"}}},
			want: url.Values{"id": {"7717"}, "updating": {"translations"}, "language": {"fr"},
				"overwrite": {"1"}, "sync_terms": {"1"}, "read_from_source": {"1"}, "fuzzy_trigger": {"1"},
				"tags": {`{"all":["web"],"obsolete":["gone"]}`}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rec recorder
			client := rec.client(t, `{"terms":{"parsed":3,"added":2,"deleted":1},"translations":{"parsed":3,"added":1,"updated":2}}`)

			res, err := client.ProjectsUpload(context.Background(), 7717, "en.json", []byte(`{"a":"b"}`), tc.opts)

			if err != nil {
				t.Fatal(err)
			}
			rec.assertForm(t, "projects/upload", tc.want)
			if rec.file != `en.json:{"a":"b"}` {
				t.Errorf("file = %q", rec.file)
			}
			if res.Terms == nil || res.Terms.Parsed != 3 || res.Terms.Added != 2 || res.Terms.Deleted != 1 {
				t.Errorf("terms = %+v", res.Terms)
			}
			if res.Translations == nil || res.Translations.Updated != 2 {
				t.Errorf("translations = %+v", res.Translations)
			}
		})
	}
}

func TestUploadTags_String(t *testing.T) {
	if got := (api.UploadTags{}).String(); got != "" {
		t.Errorf("empty = %q", got)
	}
	all := api.UploadTags{All: []string{"a"}, New: []string{"b"}, Obsolete: []string{"c"}, OverwrittenTranslations: []string{"d"}}
	if got := all.String(); got != `{"all":["a"],"new":["b"],"obsolete":["c"],"overwritten_translations":["d"]}` {
		t.Errorf("all groups = %q", got)
	}
}

func TestProjectsUpload_multiLanguageDetails(t *testing.T) {
	var rec recorder
	client := rec.client(t, `{"terms":{"parsed":2,"added":0,"deleted":0},"translations":{"parsed":0,"added":0,"updated":0,`+
		`"details":[{"language":{"name":"French","code":"fr"},"parsed":2,"added":1,"updated":1}]}}`)

	res, err := client.ProjectsUpload(context.Background(), 7717, "all.xliff", nil, api.UploadOptions{Updating: "translations"})

	if err != nil {
		t.Fatal(err)
	}
	want := []api.LanguageStats{{Language: api.LanguageRef{Name: "French", Code: "fr"}, Parsed: 2, Added: 1, Updated: 1}}
	if res.Translations == nil || !reflect.DeepEqual(res.Translations.Details, want) {
		t.Errorf("translations = %+v", res.Translations)
	}
}
