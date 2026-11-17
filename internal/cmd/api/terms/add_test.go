package terms_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		file       string // fr.json content; "" = none
		stdin      string
		data       string // the data param expected
		result     map[string]int
		wantStderr string
	}{
		{"one term, every flag", []string{"save", "--project", "web", "--context", "button", "--plural", "saves",
			"--reference", "src/save.js:12", "--comment", "<b>Max</b> 10", "--tag", "web", "--tag", "v2, beta"}, "", "",
			`[{"term":"save","context":"button","plural":"saves","reference":"src/save.js:12","comment":"<b>Max</b> 10","tags":["web","v2, beta"]}]`,
			map[string]int{"parsed": 1, "added": 1}, "Added 1 of 1 term to project web.\n"},
		{"already there", []string{"save", "--id", "7717"}, "", "",
			`[{"term":"save","context":""}]`,
			map[string]int{"parsed": 1, "added": 0}, "Added 0 of 1 term to project web.\n1 term already in the project.\n"},
		{"from a file", []string{"-p", "web", "--from-file", "terms.json"},
			`[{"term":"save","context":"button","tags":["web"]},{"term":"%d file","plural":"%d files","comment":"x"}]`, "",
			`[{"term":"save","context":"button","tags":["web"]},{"term":"%d file","context":"","plural":"%d files","comment":"x"}]`,
			map[string]int{"parsed": 2, "added": 2}, "Added 2 of 2 terms to project web.\n"},
		{"from stdin", []string{"-p", "web", "--from-file", "-"}, "", `[{"term":"a"},{"term":"b"},{"term":"c"}]`,
			`[{"term":"a","context":""},{"term":"b","context":""},{"term":"c","context":""}]`,
			map[string]int{"parsed": 3, "added": 1}, "Added 1 of 3 terms to project web.\n2 terms already in the project.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("terms/add", testutil.OK(map[string]any{"terms": tt.result}))
			env := writeEnv(t, fake)
			if tt.file != "" {
				writeFile(t, env, "terms.json", tt.file)
			}
			env.Stdin = tt.stdin

			res := env.Run(append([]string{"api", "terms", "add"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "terms/add" || req.Form.Get("id") != "7717" || req.Form.Get("api_token") != testutil.TestToken {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			assertData(t, req.Form.Get("data"), tt.data)
			if res.Stdout != "" {
				t.Errorf("stdout = %q", res.Stdout)
			}
			if res.Stderr != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", res.Stderr, tt.wantStderr)
			}
		})
	}
}

func TestAdd_json(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("terms/add", testutil.OK(map[string]any{"terms": map[string]int{"parsed": 2, "added": 1}}))
	env := writeEnv(t, fake)
	env.Stdin = `[{"term":"a"},{"term":"b"}]`

	res := env.Run("api", "terms", "add", "-p", "web", "--from-file", "-", "--json")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	testutil.Golden(t, res.Stdout)
}

// Everything is checked before any API call (and before the account).
func TestAdd_badInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		file string
		want string
	}{
		{"no term", []string{"-p", "web"}, "", "give a term, or --from-file (see --help)"},
		{"an empty term", []string{"  ", "-p", "web"}, "", "the term is empty"},
		{"two terms", []string{"a", "b", "-p", "web"}, "", "accepts at most 1 arg(s), received 2"},
		{"a term and a file", []string{"a", "-p", "web", "--from-file", "t.json"}, `[{"term":"x"}]`, "--from-file: give the terms in the file or as arguments, not both"},
		{"a term's flag and a file", []string{"-p", "web", "--from-file", "t.json", "--tag", "web"}, `[{"term":"x"}]`, "--tag: with --from-file, give it in the file, for each term"},
		{"an entry without term", []string{"-p", "web", "--from-file", "t.json"}, `[{"term":"x"},{"context":"y"}]`, `--from-file t.json: entry 2: no "term"`},
		{"an unknown field", []string{"-p", "web", "--from-file", "t.json"}, `[{"term":"x","tag":"web"}]`, `unknown field "tag"`},
		{"tags as a string", []string{"-p", "web", "--from-file", "t.json"}, `[{"term":"x","tags":"web"}]`, "cannot unmarshal string"},
		{"--project is required", []string{"save"}, "", `required flag "--project" not set`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			env := writeEnv(t, fake)
			if tt.file != "" {
				writeFile(t, env, "t.json", tt.file)
			}

			res := env.Run(append([]string{"api", "terms", "add"}, tt.args...)...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d API requests, want none", n)
			}
		})
	}
}

func TestAdd_apiFailure(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("terms/add", testutil.Fail("4032", "Strings limit reached"))

	res := testutil.Run(t, fake, "api", "terms", "add", "save", "-p", "7717")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4032" {
		t.Fatalf("want API error 4032, got %v", res.Err)
	}
	if res.Stdout != "" || res.Stderr != "" {
		t.Errorf("stdout = %q, stderr = %q", res.Stdout, res.Stderr)
	}
}

func TestAdd_notLoggedIn(t *testing.T) {
	res := testutil.Run(t, nil, "api", "terms", "add", "save", "-p", "7717")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("error = %v, want ErrNotLoggedIn", res.Err)
	}
}
