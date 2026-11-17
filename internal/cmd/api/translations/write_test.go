package translations_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

// add and update take the same input and differ in endpoint and counts.
var writeCommands = []struct {
	verb, endpoint, count string
}{
	{"add", "translations/add", "added"},
	{"update", "translations/update", "updated"},
}

func TestWrite(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		stdin string
		data  string
	}{
		{"one translation", []string{"save", "Salvează", "--project", "web", "--language", "ro", "--context", "button"}, "",
			`[{"term":"save","context":"button","translation":{"content":"Salvează","fuzzy":0}}]`},
		{"plural forms, in their order", []string{"%d files", "--id", "7717", "-l", "ro", "--plural-form", "other=%d de fișiere", "--plural-form", "one=1 fișier", "--plural-form", "few=%d fișiere"}, "",
			`[{"term":"%d files","context":"","translation":{"content":{"other":"%d de fișiere","one":"1 fișier","few":"%d fișiere"},"fuzzy":0}}]`},
		{"from stdin", []string{"-p", "web", "-l", "ro", "--from-file", "-"},
			`[{"term":"save","translation":{"content":"<b>Salvează</b>"}},{"term":"x","context":"c","translation":{"content":{"one":"a","other":"b"}}}]`,
			`[{"term":"save","context":"","translation":{"content":"<b>Salvează</b>","fuzzy":0}},{"term":"x","context":"c","translation":{"content":{"one":"a","other":"b"},"fuzzy":0}}]`},
	}
	for _, c := range writeCommands {
		for _, tt := range tests {
			t.Run(c.verb+"/"+tt.name, func(t *testing.T) {
				fake := testutil.NewFakeAPI(t)
				fake.Stub(c.endpoint, testutil.OK(map[string]any{"translations": map[string]int{"parsed": 1, c.count: 1}}))
				e := env(t, fake)
				e.Stdin = tt.stdin

				res := e.Run(append([]string{"api", "translations", c.verb}, tt.args...)...)

				if res.Err != nil {
					t.Fatal(res.Err)
				}
				req := fake.OnlyRequest(t)
				if req.Endpoint != c.endpoint || req.Form.Get("id") != "7717" || req.Form.Get("language") != "ro" || req.Form.Get("api_token") != testutil.TestToken {
					t.Errorf("request = %s %v", req.Endpoint, req.Form)
				}
				assertData(t, req.Form.Get("data"), tt.data)
				if res.Stdout != "" {
					t.Errorf("stdout = %q", res.Stdout)
				}
			})
		}
	}
}

// The summary counts the translations sent; the note says why some were not
// written.
func TestWrite_summary(t *testing.T) {
	tests := []struct {
		verb   string
		result map[string]int
		want   string
	}{
		{"add", map[string]int{"parsed": 2, "added": 2}, "Added 2 of 2 ro translations to project web.\n"},
		{"add", map[string]int{"parsed": 2, "added": 1}, "Added 1 of 2 ro translations to project web.\n" +
			"1 not added: the term already has a ro translation (translations update overwrites it), or is not in the project.\n"},
		{"update", map[string]int{"parsed": 1, "updated": 0}, "Updated 0 of 2 ro translations in project web.\n" +
			"2 not updated: the term has no ro translation yet (translations add), or is not in the project.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.verb, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("translations/"+tt.verb, testutil.OK(map[string]any{"translations": tt.result}))
			e := env(t, fake)
			e.Stdin = `[{"term":"a","translation":{"content":"x"}},{"term":"b","translation":{"content":"y"}}]`

			res := e.Run("api", "translations", tt.verb, "-p", "web", "-l", "ro", "--from-file", "-")

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			if res.Stderr != tt.want {
				t.Errorf("stderr = %q\nwant     %q", res.Stderr, tt.want)
			}
		})
	}
}

func TestWrite_json(t *testing.T) {
	for _, c := range writeCommands {
		t.Run(c.verb, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub(c.endpoint, testutil.OK(map[string]any{"translations": map[string]int{"parsed": 1, c.count: 1}}))

			res := env(t, fake).Run("api", "translations", c.verb, "save", "Salvează", "-p", "web", "-l", "ro", "--json")

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			var got map[string]any
			if err := json.Unmarshal([]byte(res.Stdout), &got); err != nil {
				t.Fatal(err)
			}
			testutil.Golden(t, res.Stdout)
		})
	}
}

func TestWrite_badInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		file string
		want string
	}{
		{"no text", []string{"save", "-p", "web", "-l", "ro"}, "", "give a term and its translation (or --plural-form for a plural term), or --from-file"},
		{"text and plural forms", []string{"save", "x", "-p", "web", "-l", "ro", "--plural-form", "one=a"}, "", "give the text or --plural-form, not both"},
		{"an empty text", []string{"save", "", "-p", "web", "-l", "ro"}, "", `the translation: no "content" (to remove a translation: translations delete)`},
		{"a plural form without =", []string{"files", "-p", "web", "-l", "ro", "--plural-form", "one"}, "", `--plural-form "one": use form=text`},
		{"a plural form twice", []string{"files", "-p", "web", "-l", "ro", "--plural-form", "one=a", "--plural-form", "one=b"}, "", "--plural-form one: given twice"},
		{"--language is required", []string{"save", "x", "-p", "web"}, "", `required flag(s) "language" not set`},
		{"--project is required", []string{"save", "x", "-l", "ro"}, "", `required flag "--project" not set`},
		{"a term and a file", []string{"save", "-p", "web", "-l", "ro", "--from-file", "t.json"}, `[{"term":"x","translation":{"content":"y"}}]`, "not both"},
		{"--plural-form with a file", []string{"-p", "web", "-l", "ro", "--from-file", "t.json", "--plural-form", "one=a"}, `[{"term":"x","translation":{"content":"y"}}]`, "--plural-form: with --from-file"},
		{"fuzzy in the file", []string{"-p", "web", "-l", "ro", "--from-file", "t.json"}, `[{"term":"x","translation":{"content":"y","fuzzy":1}}]`,
			`entry 1: "fuzzy": translations add and update ignore it; languages update sets it`},
		{"an entry without content", []string{"-p", "web", "-l", "ro", "--from-file", "t.json"}, `[{"term":"x"}]`, `entry 1: no "content"`},
		{"a misspelled field", []string{"-p", "web", "-l", "ro", "--from-file", "t.json"}, `[{"term":"x","translation":{"contents":"y"}}]`, `unknown field "contents"`},
	}
	for _, c := range writeCommands {
		for _, tt := range tests {
			t.Run(c.verb+"/"+tt.name, func(t *testing.T) {
				fake := testutil.NewFakeAPI(t)
				e := env(t, fake)
				if tt.file != "" {
					writeFile(t, e, "t.json", tt.file)
				}

				res := e.Run(append([]string{"api", "translations", c.verb}, tt.args...)...)

				if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
					t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
				}
				if n := len(fake.Requests()); n != 0 {
					t.Errorf("%d API requests, want none", n)
				}
			})
		}
	}
}

func TestWrite_languageError(t *testing.T) {
	for _, c := range writeCommands {
		t.Run(c.verb, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub(c.endpoint, testutil.Fail(api.CodeLanguageNotInProject, "Project not containing specified language"))
			fake.Stub("languages/list", testutil.OK(map[string]any{"languages": []map[string]any{{"code": "fr"}}}))

			res := env(t, fake).Run("api", "translations", c.verb, "save", "x", "-p", "web", "-l", "ro")

			if want := "--language ro: project web has no such language (it has: fr)"; res.Err == nil || res.Err.Error() != want {
				t.Errorf("error = %v, want %q", res.Err, want)
			}
			if res.Stderr != "" {
				t.Errorf("stderr = %q", res.Stderr)
			}
		})
	}
}

func TestWrite_apiFailure(t *testing.T) {
	for _, c := range writeCommands {
		t.Run(c.verb, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub(c.endpoint, testutil.Fail("4012", "Invalid project ID"))

			res := testutil.Run(t, fake, "api", "translations", c.verb, "save", "x", "-p", "7717", "-l", "ro")

			var apiErr *api.Error
			if !errors.As(res.Err, &apiErr) || apiErr.Code != "4012" {
				t.Fatalf("want API error 4012, got %v", res.Err)
			}
			if n := len(fake.Requests()); n != 1 {
				t.Errorf("%d requests: no languages/list for other errors", n)
			}
		})
	}
}

func TestWrite_notLoggedIn(t *testing.T) {
	for _, c := range writeCommands {
		res := testutil.Run(t, nil, "api", "translations", c.verb, "save", "x", "-p", "7717", "-l", "ro")

		if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
			t.Errorf("%s: error = %v, want ErrNotLoggedIn", c.verb, res.Err)
		}
	}
}
