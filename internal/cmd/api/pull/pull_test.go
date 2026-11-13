package pull_test

import (
	"errors"

	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
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
        language_map: {pt-br: pt-BR}
        pull:
          tags: [web, shared]
          tags_match: any
          filters: [translated, not_fuzzy]
          order: terms
          fallback_language: en
  - id: 2002
    account: client-a
    source_language: en
    files:
      - {path: "emails/{lang}.po", format: po, languages: [en]}
`

// server is a fake POEditor holding two projects; every export of
// (project, language, format) downloads the text "<id>/<lang>.<format>".
type server struct {
	fake    *testutil.FakeAPI
	failing string // "id/lang" whose export fails
}

func newServer(t *testing.T) *server {
	t.Helper()
	s := &server{fake: testutil.NewFakeAPI(t)}
	langs := map[string][]string{"1001": {"en", "fr", "pt-br"}, "2002": {"en", "de"}}
	s.fake.StubFunc("languages/list", func(form url.Values) testutil.Stub {
		var out []map[string]any
		for _, code := range langs[form.Get("id")] {
			out = append(out, map[string]any{"code": code})
		}
		return testutil.OK(map[string]any{"languages": out})
	})
	s.fake.StubFunc("projects/export", func(form url.Values) testutil.Stub {
		key := form.Get("id") + "/" + form.Get("language")
		if key == s.failing {
			return testutil.Fail("4044", "Invalid language code")
		}
		name := key + "." + form.Get("type")
		return testutil.OK(map[string]any{"url": s.fake.File(strings.ReplaceAll(name, "/", "_"), []byte(name))})
	})
	return s
}

func newEnv(t *testing.T, s *server) *testutil.Env {
	t.Helper()
	return testutil.NewEnv(t, s.fake).Login("personal", "p-tok").Login("client-a", "c-tok").WriteRepoConfig(config)
}

func read(t *testing.T, env *testutil.Env, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(env.WorkDir, filepath.FromSlash(rel)))
	if err != nil {
		return "<missing>"
	}
	return string(data)
}

func write(t *testing.T, env *testutil.Env, rel, content string) {
	t.Helper()
	path := filepath.Join(env.WorkDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exports(s *server) []url.Values {
	var out []url.Values
	for _, r := range s.fake.Requests() {
		if r.Endpoint == "projects/export" {
			out = append(out, r.Form)
		}
	}
	return out
}

func TestPull_everything(t *testing.T) {
	s := newServer(t)
	env := newEnv(t, s)
	write(t, env, "locales/en.json", "1001/en.key_value_json") // same as POEditor
	write(t, env, "locales/fr.json", "old local text")         // differs

	res := env.Run("pull")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	testutil.Golden(t, res.Stderr)
	if res.Stdout != "" {
		t.Errorf("stdout = %q", res.Stdout)
	}
	for rel, want := range map[string]string{
		"locales/en.json":    "1001/en.key_value_json",
		"locales/fr.json":    "1001/fr.key_value_json",
		"locales/pt-BR.json": "1001/pt-br.key_value_json", // language_map
		"emails/en.po":       "2002/en.po",
		"emails/de.po":       "<missing>", // the file lists only en
	} {
		if got := read(t, env, rel); got != want {
			t.Errorf("%s = %q, want %q", rel, got, want)
		}
	}

	ex := exports(s)
	if len(ex) != 4 {
		t.Fatalf("%d exports, want 4", len(ex))
	}
	want := url.Values{
		"api_token": {"p-tok"}, "id": {"1001"}, "language": {"en"}, "type": {"key_value_json"},
		"tags": {`["web","shared"]`}, "tags_match": {"any"}, "filters": {`["translated","not_fuzzy"]`},
		"order": {"terms"}, "fallback_language": {"en"},
	}
	if !reflect.DeepEqual(ex[0], want) {
		t.Errorf("first export params:\n%v\nwant\n%v", ex[0], want)
	}
	if got := ex[3]; got.Get("api_token") != "c-tok" || got.Get("tags") != "" || got.Get("type") != "po" {
		t.Errorf("project 2002 export: %v (want its own account's token, no pull options)", got)
	}
}

func TestPull_flags(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		files []string // export keys, in order
	}{
		{"--project by alias", []string{"pull", "--project", "web"}, []string{"1001/en", "1001/fr", "1001/pt-br"}},
		{"--project by id", []string{"pull", "--project", "2002"}, []string{"2002/en"}},
		{"--language", []string{"pull", "--language", "fr", "--project", "web"}, []string{"1001/fr"}},
		{"--language comma separated", []string{"pull", "--project", "web", "--language", "en,pt-br"}, []string{"1001/en", "1001/pt-br"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t)

			res := newEnv(t, s).Run(tt.args...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			var got []string
			for _, form := range exports(s) {
				got = append(got, form.Get("id")+"/"+form.Get("language"))
			}
			if !reflect.DeepEqual(got, tt.files) {
				t.Errorf("exported %v, want %v", got, tt.files)
			}
		})
	}
}

func TestPull_dryRun(t *testing.T) {
	s := newServer(t)
	env := newEnv(t, s)
	write(t, env, "locales/en.json", "1001/en.key_value_json")
	write(t, env, "locales/fr.json", "old local text")

	res := env.Run("pull", "--dry-run", "--project", "web")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	want := "unchanged      locales/en.json\n" +
		"would update   locales/fr.json\n" +
		"would create   locales/pt-BR.json\n" +
		"3 files: 1 would be created, 1 would be updated, 1 unchanged (dry run: nothing was written)\n"
	if res.Stderr != want {
		t.Errorf("stderr:\n%s\nwant:\n%s", res.Stderr, want)
	}
	if read(t, env, "locales/fr.json") != "old local text" || read(t, env, "locales/pt-BR.json") != "<missing>" {
		t.Error("a dry run wrote files")
	}
}

func TestPull_json(t *testing.T) {
	s := newServer(t)
	env := newEnv(t, s)
	write(t, env, "locales/en.json", "1001/en.key_value_json")

	res := env.Run("pull", "--json")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	testutil.Golden(t, res.Stdout)
}

func TestPull_stopsAtFirstError(t *testing.T) {
	s := newServer(t)
	s.failing = "1001/fr"
	env := newEnv(t, s)

	res := env.Run("pull", "--json")

	if res.Err == nil || res.Err.Error() != "locales/fr.json: API error 4044: Invalid language code" {
		t.Fatalf("error = %v", res.Err)
	}
	if read(t, env, "locales/en.json") != "1001/en.key_value_json" {
		t.Error("the file pulled before the error should stay")
	}
	if read(t, env, "locales/pt-BR.json") != "<missing>" || read(t, env, "emails/en.po") != "<missing>" {
		t.Error("pull went on after the error")
	}
	if n := len(exports(s)); n != 2 {
		t.Errorf("%d exports, want 2 (en, then the failing fr)", n)
	}
	wantJSON := `"file": "locales/en.json"`
	if !strings.Contains(res.Stdout, wantJSON) || strings.Contains(res.Stdout, "fr.json") {
		t.Errorf("--json should list what was done before the error:\n%s", res.Stdout)
	}
}

func TestPull_expiredDownload(t *testing.T) {
	s := newServer(t)
	s.fake.StubFunc("projects/export", func(url.Values) testutil.Stub {
		return testutil.OK(map[string]any{"url": s.fake.FileURL("never-registered")})
	})

	res := newEnv(t, s).Run("pull", "--project", "2002")

	if res.Err == nil || !strings.HasPrefix(res.Err.Error(), "emails/en.po: downloading the exported file failed (HTTP 404)") {
		t.Fatalf("error = %v", res.Err)
	}
}

func TestPull_refusals(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		login   bool
		args    []string
		wantErr string
		is      error
	}{
		{name: "no config", login: true, args: []string{"pull"}, is: repoconfig.ErrNotFound},
		{name: "invalid config", config: "projects:\n  - {id: 1, source_language: en, files: [{path: a.json, format: json}]}\n",
			login: true, args: []string{"pull"}, wantErr: "fix "},
		{name: "unknown project", config: config, login: true, args: []string{"pull", "--project", "nope"}, wantErr: `no project "nope"`},
		{name: "unknown language", config: config, login: true, args: []string{"pull", "--project", "web", "--language", "ja"},
			wantErr: "--language ja: project web has no such language (it has: en, fr, pt-br)"},
		{name: "not logged in", config: config, args: []string{"pull", "--project", "web"}, is: accounts.ErrNotLoggedIn},
		{name: "extra argument", config: config, login: true, args: []string{"pull", "now"}, wantErr: `unknown command "now"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t)
			env := testutil.NewEnv(t, s.fake)
			if tt.login {
				env.Login("personal", "p-tok").Login("client-a", "c-tok")
			}
			if tt.config != "" {
				env.WriteRepoConfig(tt.config)
			}

			res := env.Run(tt.args...)

			switch {
			case tt.is != nil && !errors.Is(res.Err, tt.is):
				t.Fatalf("error = %v, want %v", res.Err, tt.is)
			case tt.wantErr != "" && (res.Err == nil || !strings.Contains(res.Err.Error(), tt.wantErr)):
				t.Fatalf("error = %v, want %q", res.Err, tt.wantErr)
			}
			if n := len(exports(s)); n != 0 {
				t.Errorf("%d exports, want none", n)
			}
		})
	}
}
