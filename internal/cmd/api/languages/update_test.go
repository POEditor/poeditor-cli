package languages_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

// A file as the --help shows it: context and fuzzy left out on the plural
// row, HTML and non-ASCII text kept as they are.
const translationsFile = `[
  {"term": "save", "context": "button", "translation": {"content": "<b>Enregistrer</b>", "fuzzy": 1}},
  {"term": "files", "translation": {"content": {"one": "1 fichier", "other": "%d fichiers"}}}
]`

// what goes out as "data": every field written, plural forms in file order
const translationsData = `[{"term":"save","context":"button","translation":{"content":"<b>Enregistrer</b>","fuzzy":1}},` +
	`{"term":"files","context":"","translation":{"content":{"one":"1 fichier","other":"%d fichiers"},"fuzzy":0}}]`

const updateResult = `{"translations":{"parsed":2,"added":1,"updated":1}}`

func writeFile(t *testing.T, env *testutil.Env, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(env.WorkDir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// sameJSON compares JSON documents, ignoring spacing and \u escapes.
func sameJSON(t *testing.T, got, want string) bool {
	t.Helper()
	var g, w any
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		t.Fatalf("not JSON: %v: %s", err, got)
	}
	_ = json.Unmarshal([]byte(want), &w)
	return reflect.DeepEqual(g, w)
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		stdin        bool
		fuzzyTrigger string // the param expected
	}{
		{"from a file", []string{"fr", "--project", "7717", "--from-file", "fr.json"}, false, ""},
		{"from stdin, --id", []string{"fr", "--id", "7717", "--from-file", "-"}, true, ""},
		{"fuzzy trigger", []string{"fr", "-p", "7717", "--from-file", "fr.json", "--fuzzy-trigger"}, false, "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("languages/update", testutil.OK(json.RawMessage(updateResult)))
			env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
			if tt.stdin {
				env.Stdin = translationsFile
			} else {
				writeFile(t, env, "fr.json", translationsFile)
			}

			res := env.Run(append([]string{"api", "languages", "update"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "languages/update" || req.Form.Get("id") != "7717" || req.Form.Get("language") != "fr" ||
				req.Form.Get("api_token") != testutil.TestToken {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			if !sameJSON(t, req.Form.Get("data"), translationsData) {
				t.Errorf("data = %s\nwant   %s", req.Form.Get("data"), translationsData)
			}
			if got := req.Form.Get("fuzzy_trigger"); got != tt.fuzzyTrigger {
				t.Errorf("fuzzy_trigger = %q, want %q", got, tt.fuzzyTrigger)
			}
			if res.Stdout != "" {
				t.Errorf("stdout = %q", res.Stdout)
			}
			if want := "Updated fr in project 7717: 2 parsed, 1 added, 1 updated.\n"; res.Stderr != want {
				t.Errorf("stderr = %q, want %q", res.Stderr, want)
			}
		})
	}
}

// --json prints the counts as our own object; the stderr line stays.
func TestUpdate_json(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/update", testutil.OK(json.RawMessage(updateResult)))
	env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
	env.WriteRepoConfig(`projects:
  - id: 7717
    alias: web
    files:
      - path: locales/{lang}.json
        format: key_value_json
`)
	env.Stdin = translationsFile

	res := env.Run("api", "languages", "update", "fr", "-p", "web", "--from-file", "-", "--json")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if want := "Updated fr in project web: 2 parsed, 1 added, 1 updated.\n"; res.Stderr != want {
		t.Errorf("stderr = %q", res.Stderr)
	}
	testutil.Golden(t, res.Stdout)
}

// The plural forms keep the file's order, which a map would lose.
func TestUpdate_pluralOrderKept(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/update", testutil.OK(json.RawMessage(updateResult)))
	env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
	env.Stdin = `[{"term":"files","translation":{"content":{"other":"%d fișiere","one":"1 fișier","few":"%d fișiere"}}}]`

	res := env.Run("api", "languages", "update", "ro", "-p", "7717", "--from-file", "-")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	data := fake.OnlyRequest(t).Form.Get("data")
	if o, n, f := strings.Index(data, `"other"`), strings.Index(data, `"one"`), strings.Index(data, `"few"`); !(o < n && n < f) {
		t.Errorf("forms reordered: %s", data)
	}
}

func TestUpdate_languageErrors(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/update", testutil.Fail(api.CodeLanguageNotInProject, "Project not containing specified language"))
	fake.Stub("languages/list", testutil.OK(map[string]any{"languages": []map[string]any{{"code": "en"}}}))
	env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
	env.Stdin = translationsFile

	res := env.Run("api", "languages", "update", "fr", "-p", "7717", "--from-file", "-")

	if want := "fr: project 7717 has no such language (it has: en)"; res.Err == nil || res.Err.Error() != want {
		t.Errorf("error = %v, want %q", res.Err, want)
	}
	if res.Stderr != "" {
		t.Errorf("stderr = %q", res.Stderr)
	}
}

// A bad file fails before any API call, saying what is wrong.
func TestUpdate_badFile(t *testing.T) {
	tests := []struct {
		name    string
		content string // "" = no file
		want    string
	}{
		{"missing", "", "--from-file: open"},
		{"not JSON", "term,translation", "--from-file fr.json: invalid character"},
		{"not a list", `{"term":"save"}`, "--from-file fr.json: json: cannot unmarshal object"},
		{"empty list", `[]`, "--from-file fr.json: no translations in it"},
		{"null", `null`, "--from-file fr.json: no translations in it"},
		{"a row without a term", `[{"term":"a","translation":{"content":"x"}},{"context":"b","translation":{"content":"y"}}]`,
			`--from-file fr.json: entry 2 has no "term"`},
		{"a misspelled field", `[{"term":"a","translation":{"contents":"x"}}]`, `unknown field "contents"`},
		{"fuzzy as true", `[{"term":"a","translation":{"content":"x","fuzzy":true}}]`, "cannot unmarshal bool"},
		{"two values", `[{"term":"a"}] [{"term":"b"}]`, "--from-file fr.json: more than one JSON value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
			if tt.content != "" {
				writeFile(t, env, "fr.json", tt.content)
			}

			res := env.Run("api", "languages", "update", "fr", "-p", "7717", "--from-file", "fr.json")

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d API requests, want none", n)
			}
		})
	}
}

func TestUpdate_apiFailure(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/update", testutil.Fail("4012", "Invalid project ID"))
	env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
	env.Stdin = translationsFile

	res := env.Run("api", "languages", "update", "fr", "-p", "7717", "--from-file", "-")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4012" {
		t.Fatalf("want API error 4012, got %v", res.Err)
	}
}

func TestUpdate_notLoggedIn(t *testing.T) {
	env := testutil.NewEnv(t, nil)
	env.Stdin = translationsFile

	res := env.Run("api", "languages", "update", "fr", "-p", "7717", "--from-file", "-")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("error = %v, want ErrNotLoggedIn", res.Err)
	}
}

func TestUpdate_badArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no code", []string{"-p", "7717", "--from-file", "-"}, "accepts 1 arg(s), received 0"},
		{"--from-file is required", []string{"fr", "-p", "7717"}, `required flag(s) "from-file" not set`},
		{"--project is required", []string{"fr", "--from-file", "-"}, `required flag "--project" not set`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
			env.Stdin = translationsFile

			res := env.Run(append([]string{"api", "languages", "update"}, tt.args...)...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d API requests, want none", n)
			}
		})
	}
}
