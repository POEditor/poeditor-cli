package terms_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestMain(m *testing.M) { testutil.Main(m) }

// terms/list as the API sends it (ApiV2_Terms_List): without a language…
const termsResult = `{"terms":[` +
	`{"term":"welcome_message","context":"home","plural":"","created":"2026-01-14T09:12:44+0000","updated":"","reference":"src/home.js:12","tags":["web","v2"],"comment":"Shown on first launch"},` +
	`{"term":"files","context":"","plural":"files_plural","created":"2025-06-02T17:03:10+0000","updated":"2026-02-11T10:15:00+0000","reference":"","tags":[],"comment":""}]}`

// …and with one: a translated term (proofreading on), a plural one, an
// untranslated one, and text with HTML, a line break and \u escapes.
const translatedResult = `{"terms":[` +
	`{"term":"save","context":"button","plural":"","created":"2026-01-14T09:12:44+0000","updated":"","reference":"","tags":["web"],"comment":"",` +
	`"translation":{"content":"Salvează","fuzzy":1,"proofread":0,"updated":"2026-03-01T08:00:00+0000"}},` +
	`{"term":"files","context":"","plural":"files_plural","created":"2025-06-02T17:03:10+0000","updated":"","reference":"","tags":[],"comment":"",` +
	`"translation":{"content":{"one":"1 fișier","few":"%d fișiere","other":"%d de fișiere"},"fuzzy":0,"proofread":1,"updated":"2026-03-02T09:30:00+0000"}},` +
	`{"term":"terms_of_use","context":"","plural":"","created":"2025-06-02T17:03:10+0000","updated":"","reference":"","tags":[],"comment":"",` +
	`"translation":{"content":"","fuzzy":0,"proofread":0,"updated":""}},` +
	`{"term":"intro","context":"","plural":"","created":"2025-06-02T17:03:10+0000","updated":"","reference":"","tags":[],"comment":"",` +
	`"translation":{"content":"<b>Bine ați venit!</b>\nApăsați Start.","fuzzy":0,"proofread":1,"updated":"2026-03-03T11:00:00+0000"}}]}`

// proofreading off: the API leaves "proofread" out.
const noProofreadResult = `{"terms":[` +
	`{"term":"save","context":"","plural":"","created":"2026-01-14T09:12:44+0000","updated":"","reference":"","tags":[],"comment":"",` +
	`"translation":{"content":"Salvează","fuzzy":0,"updated":"2026-03-01T08:00:00+0000"}}]}`

const emptyResult = `{"terms":[]}`

func TestList_output(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		result   string
		tty      bool
		width    int
		language string // the language param expected
	}{
		{"table", []string{"--project", "7717"}, termsResult, true, 0, ""},
		{"table_piped", []string{"--project", "7717"}, termsResult, false, 0, ""},
		{"table_empty", []string{"--project", "7717"}, emptyResult, true, 0, ""},
		{"table_empty_piped", []string{"--project", "7717"}, emptyResult, false, 0, ""},
		{"translations", []string{"--project", "7717", "--language", "ro"}, translatedResult, true, 0, "ro"},
		{"translations_narrow", []string{"-p", "7717", "-l", "ro"}, translatedResult, true, 100, "ro"},
		{"translations_piped", []string{"-p", "7717", "-l", "ro"}, translatedResult, false, 0, "ro"},
		{"translations_no_proofreading", []string{"-p", "7717", "-l", "ro"}, noProofreadResult, true, 0, "ro"},
		{"json", []string{"-p", "7717", "--json"}, termsResult, true, 0, ""},
		{"json_translations", []string{"-p", "7717", "-l", "ro", "--json"}, translatedResult, true, 0, "ro"},
		{"json_no_proofreading", []string{"-p", "7717", "-l", "ro", "--json"}, noProofreadResult, true, 0, "ro"},
		{"json_empty", []string{"-p", "7717", "--json"}, emptyResult, true, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("terms/list", testutil.OK(json.RawMessage(tt.result)))
			env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
			env.TTY = tt.tty
			env.Width = tt.width

			res := env.Run(append([]string{"api", "terms", "list"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "terms/list" || req.Form.Get("api_token") != testutil.TestToken || req.Form.Get("id") != "7717" {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			if got := req.Form.Get("language"); got != tt.language {
				t.Errorf("language = %q, want %q", got, tt.language)
			}
			if res.Stderr != "" {
				t.Errorf("stderr = %q", res.Stderr)
			}
			testutil.Golden(t, res.Stdout)
		})
	}
}

// --id, the API docs' name, is a hidden synonym of --project.
func TestList_idSynonym(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("terms/list", testutil.OK(json.RawMessage(emptyResult)))

	res := testutil.Run(t, fake, "api", "terms", "list", "--id", "327059", "--language", "fr")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if req := fake.OnlyRequest(t); req.Form.Get("id") != "327059" || req.Form.Get("language") != "fr" {
		t.Errorf("form = %v", req.Form)
	}
	help := testutil.Run(t, nil, "api", "terms", "list", "--help")
	if !strings.Contains(help.Stdout, "--id works too") || strings.Contains(help.Stdout, "--id string") {
		t.Errorf("--help should name --id on the --project line, not as a row of its own:\n%s", help.Stdout)
	}
}

// --project takes an alias from .poeditor.yml, and its account.
func TestList_projectAlias(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("terms/list", testutil.OK(json.RawMessage(emptyResult)))
	env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken).Login("client-a", "client-token")
	env.WriteRepoConfig(`projects:
  - id: 4321
    alias: client
    account: client-a
    files:
      - path: locales/{lang}.json
        format: key_value_json
`)

	res := env.Run("api", "terms", "list", "--project", "client")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	req := fake.OnlyRequest(t)
	if req.Form.Get("id") != "4321" || req.Form.Get("api_token") != "client-token" {
		t.Errorf("want project 4321 with client-a's token, got %v", req.Form)
	}
	if res.Stdout != "Project client has no terms.\n" {
		t.Errorf("stdout = %q", res.Stdout)
	}
}

func TestList_languageErrors(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		languages any // languages/list result; nil = not stubbed
		want      string
	}{
		{"not in the project: lists the project's languages", api.CodeLanguageNotInProject,
			map[string]any{"languages": []map[string]any{{"name": "French", "code": "fr"}, {"name": "English", "code": "en"}}},
			"--language xx: project 7717 has no such language (it has: en, fr)"},
		{"not in the project, the list fails too", api.CodeLanguageNotInProject, nil,
			"--language xx: project 7717 has no such language"},
		{"unknown code", api.CodeUnknownLanguage, nil,
			"--language xx: not a language code POEditor knows (e.g. en, fr, pt-br; all of them: `poeditor api languages available`)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("terms/list", testutil.Fail(tt.code, "language error"))
			if tt.languages != nil {
				fake.Stub("languages/list", testutil.OK(tt.languages))
			} else {
				fake.Stub("languages/list", testutil.Fail("4011", "nope"))
			}

			res := testutil.Run(t, fake, "api", "terms", "list", "-p", "7717", "-l", "xx")

			if res.Err == nil || res.Err.Error() != tt.want {
				t.Errorf("error = %v, want %q", res.Err, tt.want)
			}
			if res.Stdout != "" {
				t.Errorf("stdout = %q", res.Stdout)
			}
		})
	}
}

func TestList_apiFailure(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("terms/list", testutil.Fail("4012", "Invalid project ID"))

	res := testutil.Run(t, fake, "api", "terms", "list", "-p", "7717")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4012" {
		t.Fatalf("want API error 4012, got %v", res.Err)
	}
	if res.Stdout != "" {
		t.Errorf("stdout = %q", res.Stdout)
	}
	if n := len(fake.Requests()); n != 1 {
		t.Errorf("%d requests, want 1 (no languages/list for other errors)", n)
	}
}

func TestList_notLoggedIn(t *testing.T) {
	res := testutil.Run(t, nil, "api", "terms", "list", "-p", "7717")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("error = %v, want ErrNotLoggedIn", res.Err)
	}
}

func TestList_badArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"--project is required", []string{"api", "terms", "list"}, `required flag "--project" not set`},
		{"--project and --id together", []string{"api", "terms", "list", "-p", "1", "--id", "2"}, "none of the others can be"},
		{"no arguments", []string{"api", "terms", "list", "-p", "7717", "extra"}, `unknown command "extra"`},
		{"an alias needs .poeditor.yml", []string{"api", "terms", "list", "-p", "web"}, "not a project id, and there is no .poeditor.yml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)

			res := testutil.Run(t, fake, tt.args...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d API requests, want none", n)
			}
		})
	}
}
