package terms_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

// Without --yes nothing is sent, nor is an account needed; the error says
// what would go and gives the line to run.
func TestDelete_needsYes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		file string
		want string
	}{
		{"one term", []string{"save", "--project", "web"}, "",
			`deleting term "save" deletes its translations too and cannot be undone; nothing was deleted. To delete, run:` +
				"\n  poeditor api terms delete save --project web --yes"},
		{"with its context and an account", []string{"it's", "--id", "7717", "--context", "home page", "--account", "client-a"}, "",
			`deleting term "it's" (context "home page") deletes its translations too and cannot be undone; nothing was deleted. To delete, run:` +
				"\n  poeditor api terms delete 'it'\\''s' --context 'home page' --project 7717 --account client-a --yes"},
		{"from a file", []string{"-p", "web", "--from-file", "old.json"}, `[{"term":"a"},{"term":"b","context":"c"}]`,
			`deleting the 2 terms of old.json deletes their translations too and cannot be undone; nothing was deleted. To delete, run:` +
				"\n  poeditor api terms delete --from-file old.json --project web --yes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			env := testutil.NewEnv(t, fake) // not logged in
			if tt.file != "" {
				writeFile(t, env, "old.json", tt.file)
			}

			res := env.Run(append([]string{"api", "terms", "delete"}, tt.args...)...)

			if res.Err == nil || res.Err.Error() != tt.want {
				t.Errorf("error = %v\nwant    %s", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d requests sent without --yes", n)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		stdin      string
		data       string
		result     map[string]int
		wantStderr string
	}{
		{"one term", []string{"save", "-p", "web", "--context", "button", "--yes"}, "",
			`[{"term":"save","context":"button"}]`, map[string]int{"parsed": 1, "deleted": 1}, "Deleted 1 of 1 term from project web.\n"},
		{"from stdin, some not found", []string{"--from-file", "-", "-p", "web", "--yes"}, `[{"term":"a"},{"term":"b","context":"c"}]`,
			`[{"term":"a","context":""},{"term":"b","context":"c"}]`, map[string]int{"parsed": 2, "deleted": 1},
			"Deleted 1 of 2 terms from project web.\n1 term was not found: a term is matched by its text and its context (--context).\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("terms/delete", testutil.OK(map[string]any{"terms": tt.result}))
			env := writeEnv(t, fake)
			env.Stdin = tt.stdin

			res := env.Run(append([]string{"api", "terms", "delete"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "terms/delete" || req.Form.Get("id") != "7717" || req.Form.Has("yes") {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			assertData(t, req.Form.Get("data"), tt.data)
			if res.Stdout != "" || res.Stderr != tt.wantStderr {
				t.Errorf("stdout = %q, stderr = %q, want %q", res.Stdout, res.Stderr, tt.wantStderr)
			}
		})
	}
}

func TestDelete_json(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("terms/delete", testutil.OK(map[string]any{"terms": map[string]int{"parsed": 1, "deleted": 1}}))

	res := writeEnv(t, fake).Run("api", "terms", "delete", "save", "-p", "web", "--yes", "--json")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	testutil.Golden(t, res.Stdout)
}

func TestDelete_badInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		file string
		want string
	}{
		{"--project first", []string{"save", "--yes"}, "", `required flag "--project" not set`},
		{"no term", []string{"-p", "web", "--yes"}, "", "give a term, or --from-file (see --help)"},
		{"an empty term", []string{"", "-p", "web"}, "", "the term is empty"},
		{"a term and a file", []string{"save", "-p", "web", "--from-file", "t.json"}, `[{"term":"x"}]`, "not both"},
		{"--context with a file", []string{"-p", "web", "--from-file", "t.json", "--context", "c"}, `[{"term":"x"}]`, "--context: with --from-file"},
		{"a bad file, even with --yes", []string{"-p", "web", "--from-file", "t.json", "--yes"}, `[{"term":"x"},{}]`, `entry 2: no "term"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			env := writeEnv(t, fake)
			if tt.file != "" {
				writeFile(t, env, "t.json", tt.file)
			}

			res := env.Run(append([]string{"api", "terms", "delete"}, tt.args...)...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d API requests, want none", n)
			}
		})
	}
}

func TestDelete_apiFailure(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("terms/delete", testutil.Fail("4012", "Invalid project ID"))

	res := testutil.Run(t, fake, "api", "terms", "delete", "save", "-p", "7717", "--yes")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4012" {
		t.Fatalf("want API error 4012, got %v", res.Err)
	}
}

func TestDelete_notLoggedIn(t *testing.T) {
	res := testutil.Run(t, nil, "api", "terms", "delete", "save", "-p", "7717", "--yes")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("error = %v, want ErrNotLoggedIn", res.Err)
	}
}
