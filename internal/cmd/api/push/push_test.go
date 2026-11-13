package push_test

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestMain(m *testing.M) { testutil.Main(m) }

const config = `
projects:
  - id: 1001
    alias: web
    source_language: en
    files:
      - path: locales/{lang}.json
        format: key_value_json
        push:
          overwrite: true
          sync_terms: true
          tags: {all: [web], new: [fresh]}
      - {path: "emails/{lang}.po", format: po, languages: [fr]}
  - id: 2002
    account: client-a
    source_language: en
    files:
      - {path: "app/{lang}.json", format: json}
`

// server is a fake POEditor; its uploads answer with fixed counts, after
// refusing the first `busy` attempts with 4048 (or always, if busy < 0).
type server struct {
	fake    *testutil.FakeAPI
	busy    int
	failAt  int // 1-based upload attempt that fails with 4011; 0 = never
	attempt int
}

func newServer(t *testing.T) *server {
	t.Helper()
	s := &server{fake: testutil.NewFakeAPI(t)}
	langs := map[string][]string{"1001": {"de", "en", "fr"}, "2002": {"en", "ro"}}
	s.fake.StubFunc("languages/list", func(form url.Values) testutil.Stub {
		var out []map[string]any
		for _, code := range langs[form.Get("id")] {
			out = append(out, map[string]any{"code": code})
		}
		return testutil.OK(map[string]any{"languages": out})
	})
	s.fake.StubFunc("projects/upload", func(url.Values) testutil.Stub {
		s.attempt++
		if s.failAt == s.attempt {
			return testutil.Fail("4011", "Invalid API Token")
		}
		if s.busy != 0 {
			if s.busy > 0 {
				s.busy--
			}
			return testutil.Fail("4048", "Too many upload requests in a short period of time")
		}
		return testutil.OK(map[string]any{
			"terms":        map[string]int{"parsed": 10, "added": 2, "deleted": 1},
			"translations": map[string]int{"parsed": 10, "added": 2, "updated": 3},
		})
	})
	return s
}

func newEnv(t *testing.T, s *server, files ...string) *testutil.Env {
	t.Helper()
	env := testutil.NewEnv(t, s.fake).Login("personal", "p-tok").Login("client-a", "c-tok").WriteRepoConfig(config)
	for _, rel := range files {
		path := filepath.Join(env.WorkDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("content of "+rel), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return env
}

var allFiles = []string{"locales/en.json", "locales/fr.json", "emails/fr.po", "app/en.json"}

func uploads(s *server) []testutil.Request {
	var out []testutil.Request
	for _, r := range s.fake.Requests() {
		if r.Endpoint == "projects/upload" {
			out = append(out, r)
		}
	}
	return out
}

func summary(reqs []testutil.Request) []string {
	var out []string
	for _, r := range reqs {
		out = append(out, r.Form.Get("id")+" "+r.FileName+" "+r.Form.Get("language")+" "+r.Form.Get("updating"))
	}
	return out
}

func TestPush_sourceOnlyByDefault(t *testing.T) {
	s := newServer(t)
	env := newEnv(t, s, allFiles...)

	res := env.Run("push")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	reqs := uploads(s)
	if want := []string{"1001 en.json en terms_translations", "2002 en.json en terms_translations"}; !reflect.DeepEqual(summary(reqs), want) {
		t.Fatalf("uploads = %v, want %v", summary(reqs), want)
	}
	want := url.Values{
		"api_token": {"p-tok"}, "id": {"1001"}, "updating": {"terms_translations"}, "language": {"en"},
		"overwrite": {"1"}, "sync_terms": {"1"}, "tags": {`{"all":["web"],"new":["fresh"]}`},
	}
	if !reflect.DeepEqual(reqs[0].Form, want) {
		t.Errorf("first upload params:\n%v\nwant\n%v", reqs[0].Form, want)
	}
	if string(reqs[0].FileData) != "content of locales/en.json" {
		t.Errorf("uploaded %q", reqs[0].FileData)
	}
	if got := reqs[1].Form; got.Get("api_token") != "c-tok" || got.Get("tags") != "" || got.Get("overwrite") != "" {
		t.Errorf("project 2002 upload: %v (its own token, no push options)", got)
	}
	if !reflect.DeepEqual(env.Sleeps, []time.Duration{10 * time.Second}) {
		t.Errorf("sleeps = %v, want one 10s wait between the two uploads", env.Sleeps)
	}
	for _, r := range s.fake.Requests() {
		if r.Endpoint == "languages/list" {
			t.Error("a source-only push has no reason to list languages")
		}
	}
	testutil.Golden(t, res.Stderr)
}

func TestPush_translations(t *testing.T) {
	s := newServer(t)
	env := newEnv(t, s, allFiles...)

	res := env.Run("push", "--translations")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	want := []string{
		"1001 en.json en terms_translations", // source first
		"1001 fr.json fr translations",
		"1001 fr.po fr translations", // emails list only fr
		"2002 en.json en terms_translations",
	}
	reqs := uploads(s)
	if !reflect.DeepEqual(summary(reqs), want) {
		t.Fatalf("uploads = %v, want %v", summary(reqs), want)
	}
	if reqs[1].Form.Get("sync_terms") != "" || reqs[1].Form.Get("overwrite") != "1" {
		t.Errorf("translation upload: sync_terms only on the source, overwrite follows the file: %v", reqs[1].Form)
	}
	if len(env.Sleeps) != 3 {
		t.Errorf("%d waits for 4 uploads, want 3", len(env.Sleeps))
	}
	for _, line := range []string{"missing        locales/de.json\n", "missing        app/ro.json\n", "; 2 missing locally\n"} {
		if !strings.Contains(res.Stderr, line) {
			t.Errorf("stderr lacks %q:\n%s", line, res.Stderr)
		}
	}
}

func TestPush_translationsForSomeLanguages(t *testing.T) {
	s := newServer(t)

	res := newEnv(t, s, allFiles...).Run("push", "--project", "web", "--translations", "--language", "fr")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if want := []string{"1001 fr.json fr translations", "1001 fr.po fr translations"}; !reflect.DeepEqual(summary(uploads(s)), want) {
		t.Errorf("uploads = %v, want %v (the source only when listed)", summary(uploads(s)), want)
	}
}

func TestPush_retriesWhenTooEarly(t *testing.T) {
	s := newServer(t)
	s.busy = 2
	env := newEnv(t, s, allFiles...)

	res := env.Run("push", "--project", "web")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if n := len(uploads(s)); n != 3 {
		t.Errorf("%d attempts, want 3 (two refused, one accepted)", n)
	}
	if !reflect.DeepEqual(env.Sleeps, []time.Duration{10 * time.Second, 10 * time.Second}) {
		t.Errorf("sleeps = %v", env.Sleeps)
	}
	if !strings.Contains(res.Stderr, "waiting 10s and retrying: POEditor allows one upload every 10–20s…") {
		t.Errorf("stderr = %q", res.Stderr)
	}
}

func TestPush_givesUpAfterRetries(t *testing.T) {
	s := newServer(t)
	s.busy = -1

	res := newEnv(t, s, allFiles...).Run("push", "--project", "web")

	want := "locales/en.json: API error 4048: Too many upload requests in a short period of time (still refused after 3 retries; try again in a minute)"
	if res.Err == nil || res.Err.Error() != want {
		t.Fatalf("error = %v", res.Err)
	}
	if n := len(uploads(s)); n != 4 {
		t.Errorf("%d attempts, want 4", n)
	}
}

func TestPush_stopsAtFirstError(t *testing.T) {
	s := newServer(t)
	s.failAt = 2

	res := newEnv(t, s, allFiles...).Run("push", "--json")

	if res.Err == nil || res.Err.Error() != "app/en.json: API error 4011: Invalid API Token" {
		t.Fatalf("error = %v", res.Err)
	}
	if !strings.Contains(res.Stdout, `"file": "locales/en.json"`) || strings.Contains(res.Stdout, "app/en.json") {
		t.Errorf("--json should list the upload done before the error:\n%s", res.Stdout)
	}
}

func TestPush_dryRun(t *testing.T) {
	s := newServer(t)
	env := newEnv(t, s, allFiles...)

	res := env.Run("push", "--dry-run", "--project", "web", "--translations")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if n := len(uploads(s)); n != 0 {
		t.Errorf("a dry run uploaded %d files", n)
	}
	if len(env.Sleeps) != 0 {
		t.Errorf("a dry run waited: %v", env.Sleeps)
	}
	testutil.Golden(t, res.Stderr)
}

func TestPush_json(t *testing.T) {
	s := newServer(t)

	res := newEnv(t, s, allFiles...).Run("push", "--project", "web", "--translations", "--json")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	testutil.Golden(t, res.Stdout)
}

func TestPush_checksBeforeUploading(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		files   []string
		args    []string
		wantErr string
		is      error
		noLogin bool
	}{
		{name: "source file missing", config: config, files: []string{"app/en.json"}, args: []string{"push"},
			wantErr: "locales/en.json: source file not found (project web, source_language en); nothing was uploaded"},
		{name: "format that cannot be pushed", files: []string{"m/en.mo"}, args: []string{"push"},
			config:  "projects:\n  - {id: 1, source_language: en, files: [{path: \"m/{lang}.mo\", format: mo}]}\n",
			wantErr: "project 1: m/{lang}.mo: format mo can be pulled but not pushed"},
		{name: "no source language", files: []string{"a/en.json"}, args: []string{"push"},
			config:  "projects:\n  - {id: 1, files: [{path: \"a/{lang}.json\", format: json}]}\n",
			wantErr: "project 1 has no source_language in "},
		{name: "--language without --translations", config: config, files: allFiles, args: []string{"push", "--language", "fr"},
			wantErr: "--language only applies with --translations"},
		{name: "unknown language", config: config, files: allFiles, args: []string{"push", "--project", "web", "--translations", "--language", "ja"},
			wantErr: "--language ja: project web has no such language (it has: de, en, fr)"},
		{name: "not logged in", config: config, files: allFiles, args: []string{"push"}, is: accounts.ErrNotLoggedIn, noLogin: true},
		{name: "extra argument", config: config, files: allFiles, args: []string{"push", "now"}, wantErr: `unknown command "now"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t)
			env := testutil.NewEnv(t, s.fake)
			if !tt.noLogin {
				env.Login("personal", "p-tok").Login("client-a", "c-tok")
			}
			env.WriteRepoConfig(tt.config)
			for _, rel := range tt.files {
				path := filepath.Join(env.WorkDir, filepath.FromSlash(rel))
				_ = os.MkdirAll(filepath.Dir(path), 0o755)
				_ = os.WriteFile(path, []byte("x"), 0o644)
			}

			res := env.Run(tt.args...)

			switch {
			case tt.is != nil && !errors.Is(res.Err, tt.is):
				t.Fatalf("error = %v, want %v", res.Err, tt.is)
			case tt.wantErr != "" && (res.Err == nil || !strings.Contains(res.Err.Error(), tt.wantErr)):
				t.Fatalf("error = %v, want %q", res.Err, tt.wantErr)
			}
			if n := len(uploads(s)); n != 0 {
				t.Errorf("%d uploads, want none", n)
			}
		})
	}
}
