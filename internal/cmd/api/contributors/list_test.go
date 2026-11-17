package contributors_test

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

// contributors/list as the API sends it (ApiV2_Contributors_List): project
// ids as strings, an administrator without languages, a proofreader, and one
// person in two projects.
const (
	allProjects = `{"contributors":[` +
		`{"name":"Ana Pop","email":"ana@example.com","permissions":[` +
		`{"project":{"id":"7717","name":"Web"},"type":"contributor","proofreader":true,"languages":["fr","de"]},` +
		`{"project":{"id":"4321","name":"Mobile app"},"type":"contributor","proofreader":false,"languages":["fr"]}]},` +
		`{"name":"Bo","email":"bo@example.com","permissions":[{"project":{"id":"7717","name":"Web"},"type":"administrator","proofreader":true}]}]}`
	oneProject = `{"contributors":[` +
		`{"name":"Ana Pop","email":"ana@example.com","permissions":[{"project":{"id":"7717","name":"Web"},"type":"contributor","proofreader":false,"languages":["fr"]}]},` +
		`{"name":"Bo","email":"bo@example.com","permissions":[{"project":{"id":"7717","name":"Web"},"type":"administrator","proofreader":false}]}]}`
	noContributors = `{"contributors":[]}`
)

func TestList_output(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		result   string
		tty      bool
		wantForm map[string]string // id, language ("" = not sent)
	}{
		{"all", nil, allProjects, true, map[string]string{"id": "", "language": ""}},
		{"all_piped", nil, allProjects, false, map[string]string{"id": "", "language": ""}},
		{"all_empty", nil, noContributors, true, map[string]string{"id": "", "language": ""}},
		{"project", []string{"--project", "7717"}, oneProject, true, map[string]string{"id": "7717", "language": ""}},
		{"project_language", []string{"--id", "7717", "--language", "fr"}, oneProject, true, map[string]string{"id": "7717", "language": "fr"}},
		{"project_empty", []string{"-p", "7717"}, noContributors, true, map[string]string{"id": "7717"}},
		{"project_empty_piped", []string{"-p", "7717"}, noContributors, false, map[string]string{"id": "7717"}},
		{"json", []string{"--json"}, allProjects, true, map[string]string{"id": ""}},
		{"json_empty", []string{"-p", "7717", "-l", "fr", "--json"}, noContributors, true, map[string]string{"id": "7717", "language": "fr"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("contributors/list", testutil.OK(json.RawMessage(tt.result)))
			env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
			env.TTY = tt.tty

			res := env.Run(append([]string{"api", "contributors", "list"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Form.Get("api_token") != testutil.TestToken {
				t.Errorf("form = %v", req.Form)
			}
			for k, want := range tt.wantForm {
				if got := req.Form.Get(k); got != want || (want == "" && req.Form.Has(k)) {
					t.Errorf("%s = %q (sent %v), want %q", k, got, req.Form.Has(k), want)
				}
			}
			if res.Stderr != "" {
				t.Errorf("stderr = %q", res.Stderr)
			}
			testutil.Golden(t, res.Stdout)
		})
	}
}

func TestList_errors(t *testing.T) {
	t.Run("--language needs --project, before any call", func(t *testing.T) {
		fake := testutil.NewFakeAPI(t)

		res := testutil.Run(t, fake, "api", "contributors", "list", "--language", "fr")

		if res.Err == nil || !strings.HasPrefix(res.Err.Error(), "--language needs --project") {
			t.Errorf("error = %v", res.Err)
		}
		if n := len(fake.Requests()); n != 0 {
			t.Errorf("%d requests", n)
		}
	})
	t.Run("a language the project lacks", func(t *testing.T) {
		fake := testutil.NewFakeAPI(t)
		fake.Stub("contributors/list", testutil.Fail(api.CodeLanguageNotInProject, "Project not containing specified language"))
		fake.Stub("languages/list", testutil.OK(map[string]any{"languages": []map[string]any{{"code": "de"}}}))

		res := testutil.Run(t, fake, "api", "contributors", "list", "-p", "7717", "-l", "fr")

		if want := "--language fr: project 7717 has no such language (it has: de)"; res.Err == nil || res.Err.Error() != want {
			t.Errorf("error = %v, want %q", res.Err, want)
		}
	})
	t.Run("API failure", func(t *testing.T) {
		fake := testutil.NewFakeAPI(t)
		fake.Stub("contributors/list", testutil.Fail("4011", "Invalid API Token"))

		res := testutil.Run(t, fake, "api", "contributors", "list")

		var apiErr *api.Error
		if !errors.As(res.Err, &apiErr) || apiErr.Code != "4011" || res.Stdout != "" {
			t.Errorf("error = %v, stdout = %q", res.Err, res.Stdout)
		}
	})
	t.Run("not logged in", func(t *testing.T) {
		for _, args := range [][]string{{"api", "contributors", "list"}, {"api", "contributors", "list", "-p", "7717"}} {
			if res := testutil.Run(t, nil, args...); !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
				t.Errorf("%v: error = %v", args, res.Err)
			}
		}
	})
	t.Run("an argument", func(t *testing.T) {
		fake := testutil.NewFakeAPI(t)
		if res := testutil.Run(t, fake, "api", "contributors", "list", "web"); res.Err == nil || len(fake.Requests()) != 0 {
			t.Errorf("error = %v", res.Err)
		}
	})
}
