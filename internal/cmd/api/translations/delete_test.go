package translations_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestDelete_needsYes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		file string
		want string
	}{
		{"one term", []string{"save", "--project", "web", "--language", "ro", "--context", "button"}, "",
			`deleting the ro translation of term "save" (context "button") cannot be undone; nothing was deleted. To delete, run:` +
				"\n  poeditor api translations delete save --context button --project web --language ro --yes"},
		{"from a file, with an account", []string{"--id", "7717", "-l", "ro", "--from-file", "stale.json", "--account", "client-a"}, `[{"term":"a"}]`,
			`deleting the ro translation of the 1 term of stale.json cannot be undone; nothing was deleted. To delete, run:` +
				"\n  poeditor api translations delete --from-file stale.json --project 7717 --language ro --account client-a --yes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			e := testutil.NewEnv(t, fake) // not logged in
			if tt.file != "" {
				writeFile(t, e, "stale.json", tt.file)
			}

			res := e.Run(append([]string{"api", "translations", "delete"}, tt.args...)...)

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
		{"one term", []string{"save", "-p", "web", "-l", "ro", "--context", "button", "--yes"}, "",
			`[{"term":"save","context":"button"}]`, map[string]int{"parsed": 1, "deleted": 1}, "Deleted 1 of 1 ro translation from project web.\n"},
		{"from stdin, one without translation", []string{"-p", "web", "-l", "ro", "--from-file", "-", "--yes"}, `[{"term":"a"},{"term":"b"}]`,
			`[{"term":"a","context":""},{"term":"b","context":""}]`, map[string]int{"parsed": 2, "deleted": 1},
			"Deleted 1 of 2 ro translations from project web.\n1 not deleted: the term has no ro translation, or is not in the project.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("translations/delete", testutil.OK(map[string]any{"translations": tt.result}))
			e := env(t, fake)
			e.Stdin = tt.stdin

			res := e.Run(append([]string{"api", "translations", "delete"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "translations/delete" || req.Form.Get("id") != "7717" || req.Form.Get("language") != "ro" || req.Form.Has("yes") {
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
	fake.Stub("translations/delete", testutil.OK(map[string]any{"translations": map[string]int{"parsed": 1, "deleted": 1}}))

	res := env(t, fake).Run("api", "translations", "delete", "save", "-p", "web", "-l", "ro", "--yes", "--json")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	testutil.Golden(t, res.Stdout)
}

func TestDelete_badInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"--project first", []string{"save", "-l", "ro"}, `required flag "--project" not set`},
		{"--language is required", []string{"save", "-p", "web", "--yes"}, `required flag(s) "language" not set`},
		{"no term", []string{"-p", "web", "-l", "ro"}, "give a term, or --from-file"},
		{"an empty term", []string{" ", "-p", "web", "-l", "ro"}, "the term is empty"},
		{"--context with a file", []string{"-p", "web", "-l", "ro", "--from-file", "-", "--context", "c"}, "--context: with --from-file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)

			res := env(t, fake).Run(append([]string{"api", "translations", "delete"}, tt.args...)...)

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
	fake.Stub("translations/delete", testutil.Fail(api.CodeUnknownLanguage, "Wrong language code"))

	res := testutil.Run(t, fake, "api", "translations", "delete", "save", "-p", "7717", "-l", "xx", "--yes")

	if res.Err == nil || !strings.HasPrefix(res.Err.Error(), "--language xx: not a language code POEditor knows") {
		t.Errorf("error = %v", res.Err)
	}
}

func TestDelete_notLoggedIn(t *testing.T) {
	res := testutil.Run(t, nil, "api", "translations", "delete", "save", "-p", "7717", "-l", "ro", "--yes")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("error = %v, want ErrNotLoggedIn", res.Err)
	}
}
