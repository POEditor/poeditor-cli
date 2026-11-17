package projects_test

import (
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

var uploadResult = map[string]any{
	"terms":        map[string]int{"parsed": 3, "added": 2, "deleted": 1},
	"translations": map[string]int{"parsed": 3, "added": 2, "updated": 1},
}

func TestUpload(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want map[string]string // besides id and the file; "" = not sent
	}{
		{"terms only", []string{"--updating", "terms"},
			map[string]string{"updating": "terms", "language": "", "overwrite": "", "sync_terms": "", "tags": ""}},
		{"translations, every switch", []string{"--updating", "translations", "--language", "fr", "--overwrite", "--read-from-source", "--fuzzy-trigger"},
			map[string]string{"updating": "translations", "language": "fr", "overwrite": "1", "read_from_source": "1", "fuzzy_trigger": "1", "sync_terms": ""}},
		{"tags and sync, with --yes", []string{"--updating", "terms_translations", "-l", "en", "--sync-terms", "--yes",
			"--tag-all", "v2", "--tag-new", "new", "--tag-new", "web", "--tag-obsolete", "old", "--tag-overwritten", "check"},
			map[string]string{"sync_terms": "1", "tags": `{"all":["v2"],"new":["new","web"],"obsolete":["old"],"overwritten_translations":["check"]}`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("projects/upload", testutil.OK(uploadResult))
			e := env(t, fake)
			writeFile(t, e, "locales/en.json", `{"save":"Save"}`)

			res := e.Run(append([]string{"api", "projects", "upload", "locales/en.json", "-p", "web"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "projects/upload" || req.Form.Get("id") != "7717" || req.FileName != "en.json" || string(req.FileData) != `{"save":"Save"}` || req.Form.Has("yes") {
				t.Errorf("request = %s %v %s", req.Endpoint, req.Form, req.FileName)
			}
			for k, want := range tt.want {
				got := req.Form.Get(k)
				if k == "tags" && want != "" {
					assertJSON(t, got, want)
					continue
				}
				if got != want || (want == "" && req.Form.Has(k)) {
					t.Errorf("%s = %q, want %q", k, got, want)
				}
			}
			if want := "Uploaded locales/en.json to project web: terms: 3 parsed, 2 added, 1 deleted · translations: 2 added, 1 updated\n"; res.Stderr != want {
				t.Errorf("stderr = %q", res.Stderr)
			}
			if res.Stdout != "" {
				t.Errorf("stdout = %q", res.Stdout)
			}
		})
	}
}

// Too early: the command waits and retries, like push.
func TestUpload_waitsAndRetries(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	calls := 0
	fake.StubFunc("projects/upload", func(url.Values) testutil.Stub {
		if calls++; calls == 1 {
			return testutil.Fail(api.CodeTooManyUploads, "Too many upload requests in a short period of time")
		}
		return testutil.OK(uploadResult)
	})
	e := env(t, fake)
	writeFile(t, e, "en.json", "{}")

	res := e.Run("api", "projects", "upload", "en.json", "--id", "7717", "--updating", "terms", "--json")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if len(e.Sleeps) != 1 || !strings.HasPrefix(res.Stderr, "waiting 10s and retrying") || len(fake.Requests()) != 2 {
		t.Errorf("sleeps = %v, stderr = %q", e.Sleeps, res.Stderr)
	}
	testutil.Golden(t, res.Stdout)
}

func TestUpload_syncNeedsYes(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	e := testutil.NewEnv(t, fake) // not logged in: nothing is resolved without --yes
	writeFile(t, e, "en.json", "{}")

	res := e.Run("api", "projects", "upload", "en.json", "-p", "web", "--updating", "terms_translations", "-l", "en", "--sync-terms", "--tag-new", "v 2")

	want := "--sync-terms deletes every term of project web that is not in en.json, with its translations; nothing was sent. To upload, run:\n" +
		"  poeditor api projects upload en.json --project web --updating terms_translations --language en --sync-terms --tag-new 'v 2' --yes"
	if res.Err == nil || res.Err.Error() != want {
		t.Errorf("error = %v\nwant    %s", res.Err, want)
	}
	if n := len(fake.Requests()); n != 0 {
		t.Errorf("%d requests", n)
	}
}

func TestUpload_badInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no --updating", []string{"en.json", "-p", "web"}, `required flag(s) "updating" not set`},
		{"a bad --updating", []string{"en.json", "-p", "web", "--updating", "all"}, `--updating "all": use terms, terms_translations, translations`},
		{"translations need a language", []string{"en.json", "-p", "web", "--updating", "translations"}, "--updating translations needs --language"},
		{"stdin", []string{"-", "-p", "web", "--updating", "terms"}, "upload a file by its name"},
		{"a missing file", []string{"nope.json", "-p", "web", "--updating", "terms"}, "nope.json: no such file"},
		{"no file", []string{"-p", "web", "--updating", "terms"}, "accepts 1 arg(s), received 0"},
		{"no project", []string{"en.json", "--updating", "terms"}, `required flag "--project" not set`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)

			res := env(t, fake).Run(append([]string{"api", "projects", "upload"}, tt.args...)...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d requests", n)
			}
		})
	}
}

func TestUpload_errors(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/upload", testutil.Fail("4046", "The system was not able to parse the contents of your file."))
	e := env(t, fake)
	writeFile(t, e, "en.json", "{")

	res := e.Run("api", "projects", "upload", "en.json", "-p", "web", "--updating", "terms")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4046" || res.Stderr != "" {
		t.Errorf("error = %v, stderr = %q", res.Err, res.Stderr)
	}
	ne := testutil.NewEnv(t, nil)
	writeFile(t, ne, "en.json", "{}")
	if res := ne.Run("api", "projects", "upload", "en.json", "-p", "7717", "--updating", "terms"); !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("not logged in: %v", res.Err)
	}
}
