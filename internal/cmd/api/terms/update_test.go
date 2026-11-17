package terms_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestUpdate(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		stdin      string
		data       string
		fuzzy      string // fuzzy_trigger expected
		result     map[string]int
		wantStderr string
	}{
		{"rename, found by its context", []string{"save", "-p", "web", "--context", "button", "--new-term", "Save", "--new-context", "toolbar"}, "",
			`[{"term":"save","context":"button","new_term":"Save","new_context":"toolbar"}]`, "",
			map[string]int{"parsed": 1, "updated": 1}, "Updated 1 of 1 term in project web.\n"},
		{"clear plural and reference: only what was given is sent", []string{"save", "-p", "web", "--plural", "", "--reference", ""}, "",
			`[{"term":"save","context":"","plural":"","reference":""}]`, "",
			map[string]int{"parsed": 1, "updated": 1}, "Updated 1 of 1 term in project web.\n"},
		{"tags replace", []string{"save", "--id", "7717", "--tag", "web", "--tag", "v2"}, "",
			`[{"term":"save","context":"","tags":["web","v2"]}]`, "",
			map[string]int{"parsed": 1, "updated": 1}, "Updated 1 of 1 term in project web.\n"},
		{"only the fuzzy trigger", []string{"save", "-p", "web", "--fuzzy-trigger"}, "",
			`[{"term":"save","context":""}]`, "1",
			map[string]int{"parsed": 1, "updated": 1}, "Updated 1 of 1 term in project web.\n"},
		{"not found", []string{"save", "-p", "web", "--context", "nope", "--plural", "saves"}, "",
			`[{"term":"save","context":"nope","plural":"saves"}]`, "",
			map[string]int{"parsed": 1, "updated": 0}, "Updated 0 of 1 term in project web.\n1 term was not found: a term is matched by its text and its context (--context).\n"},
		{"from a file, with the fuzzy trigger", []string{"-p", "web", "--from-file", "-", "--fuzzy-trigger"},
			`[{"term":"a","new_term":"A","tags":[]},{"term":"b","context":"c","reference":"x.js"},{"term":"z"}]`,
			`[{"term":"a","context":"","new_term":"A","tags":[]},{"term":"b","context":"c","reference":"x.js"},{"term":"z","context":""}]`, "1",
			map[string]int{"parsed": 3, "updated": 1}, "Updated 1 of 3 terms in project web.\n2 terms were not found: a term is matched by its text and its context (--context).\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("terms/update", testutil.OK(map[string]any{"terms": tt.result}))
			env := writeEnv(t, fake)
			env.Stdin = tt.stdin

			res := env.Run(append([]string{"api", "terms", "update"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "terms/update" || req.Form.Get("id") != "7717" || req.Form.Get("api_token") != testutil.TestToken {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			assertData(t, req.Form.Get("data"), tt.data)
			if got := req.Form.Get("fuzzy_trigger"); got != tt.fuzzy {
				t.Errorf("fuzzy_trigger = %q, want %q", got, tt.fuzzy)
			}
			if res.Stdout != "" {
				t.Errorf("stdout = %q", res.Stdout)
			}
			if res.Stderr != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", res.Stderr, tt.wantStderr)
			}
		})
	}
}

func TestUpdate_json(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("terms/update", testutil.OK(map[string]any{"terms": map[string]int{"parsed": 2, "updated": 1}}))
	env := writeEnv(t, fake)
	env.Stdin = `[{"term":"a","plural":"as"},{"term":"b","plural":"bs"}]`

	res := env.Run("api", "terms", "update", "-p", "web", "--from-file", "-", "--json")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	testutil.Golden(t, res.Stdout)
}

func TestUpdate_badInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		file string
		want string
	}{
		{"no term", []string{"-p", "web", "--plural", "x"}, "", "give a term, or --from-file (see --help)"},
		{"nothing to change", []string{"save", "-p", "web", "--context", "button"}, "", "nothing to change: give --new-term, --new-context, --plural, --reference, --tag or --fuzzy-trigger"},
		{"renamed to nothing", []string{"save", "-p", "web", "--new-term", " "}, "", `the term: "new_term" is empty`},
		{"an empty term", []string{"", "-p", "web", "--plural", "x"}, "", `the term: no "term"`},
		{"a term and a file", []string{"save", "-p", "web", "--from-file", "t.json"}, `[{"term":"x"}]`, "not both"},
		{"--context with a file", []string{"-p", "web", "--from-file", "t.json", "--context", "c"}, `[{"term":"x"}]`, "--context: with --from-file, give it in the file"},
		{"--new-term with a file", []string{"-p", "web", "--from-file", "t.json", "--new-term", "y"}, `[{"term":"x"}]`, "--new-term: with --from-file"},
		{"an entry renamed to nothing", []string{"-p", "web", "--from-file", "t.json"}, `[{"term":"x","new_term":""}]`, `entry 1: "new_term" is empty`},
		{"a misspelled field", []string{"-p", "web", "--from-file", "t.json"}, `[{"term":"x","newterm":"y"}]`, `unknown field "newterm"`},
		{"--project is required", []string{"save", "--plural", "x"}, "", `required flag "--project" not set`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			env := writeEnv(t, fake)
			if tt.file != "" {
				writeFile(t, env, "t.json", tt.file)
			}

			res := env.Run(append([]string{"api", "terms", "update"}, tt.args...)...)

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
	fake.Stub("terms/update", testutil.Fail("4033", "Currently processing an uploaded file. Please try again later."))

	res := testutil.Run(t, fake, "api", "terms", "update", "save", "-p", "7717", "--plural", "saves")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4033" {
		t.Fatalf("want API error 4033, got %v", res.Err)
	}
	if res.Stdout != "" || res.Stderr != "" {
		t.Errorf("stdout = %q, stderr = %q", res.Stdout, res.Stderr)
	}
}

func TestUpdate_notLoggedIn(t *testing.T) {
	res := testutil.Run(t, nil, "api", "terms", "update", "save", "-p", "7717", "--plural", "saves")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("error = %v, want ErrNotLoggedIn", res.Err)
	}
}
