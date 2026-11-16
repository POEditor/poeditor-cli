package config_test

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/repoconfig"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestMain(m *testing.M) { testutil.Main(m) }

const twoProjects = `
projects:
  - id: 1001
    alias: web
    source_language: en
    files:
      - {path: "locales/{lang}.json", format: key_value_json}
      - {path: "emails/{lang}.po", format: po, languages: [en, fr]}
  - id: 2002
    account: client-a
    source_language: en
    files:
      - path: "app/src/main/res/values-{lang}/strings.xml"
        format: android_strings
        language_map: {pt-br: pt-rBR}
        pull: {fallback_language: en}
`

func TestShow(t *testing.T) {
	t.Run("table", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).Login("personal", "p").WriteRepoConfig(twoProjects)

		res := env.Run("config", "show")

		if res.Err != nil {
			t.Fatal(res.Err)
		}
		testutil.Golden(t, res.Stdout)
		if res.Stderr != "" {
			t.Errorf("stderr = %q", res.Stderr)
		}
	})

	t.Run("--account overrides every project", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).WriteRepoConfig(twoProjects)
		env.TTY = false

		res := env.Run("config", "show", "--account", "other")

		want := "web\t1001\tother (--account)\tkey_value_json\tlocales/{lang}.json\n" +
			"web\t1001\tother (--account)\tpo\temails/{lang}.po\n" +
			"2002\t2002\tother (--account)\tandroid_strings\tapp/src/main/res/values-{lang}/strings.xml\n"
		if res.Err != nil || res.Stdout != want {
			t.Fatalf("got %q, %v\nwant %q", res.Stdout, res.Err, want)
		}
	})

	t.Run("json", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).WriteRepoConfig(twoProjects)

		res := env.Run("config", "show", "--json")

		if res.Err != nil {
			t.Fatal(res.Err)
		}
		got := strings.ReplaceAll(res.Stdout, env.WorkDir, "<repo>")
		got = strings.ReplaceAll(got, strings.ReplaceAll(env.WorkDir, `\`, `\\`), "<repo>") // Windows paths in JSON
		testutil.Golden(t, got)
	})

	t.Run("found from a subdirectory", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).WriteRepoConfig(twoProjects)
		env.WorkDir = filepath.Join(env.WorkDir, "src", "deep")
		if err := os.MkdirAll(env.WorkDir, 0o755); err != nil {
			t.Fatal(err)
		}

		res := env.Run("config", "show")

		if res.Err != nil || !strings.Contains(res.Stdout, "locales/{lang}.json") {
			t.Fatalf("got %q, %v", res.Stdout, res.Err)
		}
	})

	t.Run("warnings go to stderr, still succeeds", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).WriteRepoConfig(`
projects:
  - {id: 1, files: [{path: "a/{lang}.json", format: json}]}
`)

		res := env.Run("config", "show")

		if res.Err != nil {
			t.Fatal(res.Err)
		}
		if !strings.Contains(res.Stderr, "warning: project 1: `source_language` is not set") {
			t.Errorf("stderr = %q", res.Stderr)
		}
	})

	t.Run("errors fail the command", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).WriteRepoConfig(`
projects:
  - {id: 1, source_language: en, files: [{path: "a.json", format: json}, {path: "b/{lang}.json"}]}
`)

		res := env.Run("config", "show")

		if res.Err == nil || !strings.HasPrefix(res.Err.Error(), "2 problems in ") {
			t.Fatalf("error = %v", res.Err)
		}
		if strings.Count(res.Stderr, "error: ") != 2 {
			t.Errorf("stderr = %q", res.Stderr)
		}
	})

	t.Run("no file", func(t *testing.T) {
		res := testutil.NewEnv(t, nil).Run("config", "show")

		if !errors.Is(res.Err, repoconfig.ErrNotFound) {
			t.Fatalf("error = %v", res.Err)
		}
	})

	t.Run("unknown key", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).WriteRepoConfig("projects:\n  - id: 1\n    filez: []\n")

		res := env.Run("config", "show")

		if res.Err == nil || !strings.Contains(res.Err.Error(), "field filez not found") {
			t.Fatalf("error = %v", res.Err)
		}
	})
}

// fakeProjects answers projects/view and languages/list for the given
// project ids, and fails any other id the way the API does.
func fakeProjects(t *testing.T, projects map[string][]string) *testutil.FakeAPI {
	t.Helper()
	fake := testutil.NewFakeAPI(t)
	fake.StubFunc("projects/view", func(form url.Values) testutil.Stub {
		if _, ok := projects[form.Get("id")]; !ok {
			return testutil.Fail("403", "You don't have permission to access this resource")
		}
		return testutil.OK(map[string]any{"project": map[string]any{"id": formID(form), "name": "Project " + form.Get("id")}})
	})
	fake.StubFunc("languages/list", func(form url.Values) testutil.Stub {
		var langs []map[string]any
		for _, code := range projects[form.Get("id")] {
			langs = append(langs, map[string]any{"name": code, "code": code})
		}
		return testutil.OK(map[string]any{"languages": langs})
	})
	return fake
}

func TestCheck(t *testing.T) {
	t.Run("all good", func(t *testing.T) {
		fake := fakeProjects(t, map[string][]string{"1001": {"en", "fr", "de"}, "2002": {"en", "pt-br"}})
		env := testutil.NewEnv(t, fake).Login("personal", "p-tok").Login("client-a", "c-tok").WriteRepoConfig(twoProjects)

		res := env.Run("config", "check")

		if res.Err != nil {
			t.Fatal(res.Err)
		}
		want := "ok  web: \"Project 1001\" (1001), account personal, 3 languages\n" +
			"ok  2002: \"Project 2002\" (2002), account client-a, 2 languages\n"
		if res.Stdout != want {
			t.Errorf("stdout = %q\nwant %q", res.Stdout, want)
		}
		tokens := map[string]string{}
		for _, req := range fake.Requests() {
			tokens[req.Form.Get("id")] = req.Form.Get("api_token")
		}
		if tokens["1001"] != "p-tok" || tokens["2002"] != "c-tok" {
			t.Errorf("each project must use its own account's token: %v", tokens)
		}
	})

	t.Run("missing languages and an inaccessible project", func(t *testing.T) {
		fake := fakeProjects(t, map[string][]string{"1001": {"en", "de"}})
		env := testutil.NewEnv(t, fake).Login("personal", "p").Login("client-a", "c").WriteRepoConfig(twoProjects)

		res := env.Run("config", "check")

		if res.Err == nil || !strings.HasPrefix(res.Err.Error(), "2 problems in ") {
			t.Fatalf("error = %v", res.Err)
		}
		testutil.Golden(t, res.Stdout)
	})

	t.Run("project account not set up", func(t *testing.T) {
		fake := fakeProjects(t, map[string][]string{"1001": {"en", "fr"}, "2002": {"en"}})
		env := testutil.NewEnv(t, fake).Login("personal", "p").WriteRepoConfig(twoProjects)

		res := env.Run("config", "check")

		if res.Err == nil {
			t.Fatal("want an error")
		}
		if !strings.Contains(res.Stdout, "FAILED  2002\n    error: account \"client-a\" isn't set up. Run `poeditor auth login --account client-a`\n") {
			t.Errorf("stdout = %q", res.Stdout)
		}
		for _, req := range fake.Requests() {
			if req.Form.Get("id") == "2002" {
				t.Error("no API call expected for a project without a token")
			}
		}
	})

	t.Run("invalid file: no API calls", func(t *testing.T) {
		fake := testutil.NewFakeAPI(t)
		env := testutil.NewEnv(t, fake).Login("personal", "p").WriteRepoConfig(`
projects:
  - {id: 1, source_language: en, files: [{path: "a.json", format: json}]}
`)

		res := env.Run("config", "check")

		if res.Err == nil || !strings.Contains(res.Stderr, "`path` needs {lang}") {
			t.Fatalf("err = %v, stderr = %q", res.Err, res.Stderr)
		}
		if n := len(fake.Requests()); n != 0 {
			t.Errorf("%d API calls, want 0", n)
		}
	})

	t.Run("json", func(t *testing.T) {
		fake := fakeProjects(t, map[string][]string{"1001": {"en", "fr"}, "2002": {"en", "pt-br"}})
		env := testutil.NewEnv(t, fake).Login("personal", "p").Login("client-a", "c").WriteRepoConfig(twoProjects)

		res := env.Run("config", "check", "--json")

		if res.Err != nil {
			t.Fatal(res.Err)
		}
		if !strings.HasPrefix(res.Stdout, "{\n  \"ok\": true,") || !strings.Contains(res.Stdout, "\"account\": \"client-a\"") ||
			!strings.Contains(res.Stdout, "\"project\": {\n        \"id\": 1001,\n        \"alias\": \"web\",\n        \"name\": \"Project 1001\"\n      }") {
			t.Errorf("stdout = %s", res.Stdout)
		}
	})
}
