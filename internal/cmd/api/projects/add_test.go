package projects_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		args []string
		form map[string]string // "" = not sent
		json bool
	}{
		{"name only", []string{"Mobile app"}, map[string]string{"name": "Mobile app", "description": ""}, false},
		{"description, json", []string{" Mobile app ", "--description", "The app", "--json"}, map[string]string{"name": "Mobile app", "description": "The app"}, true},
		{"--name", []string{"--name", " Mobile app "}, map[string]string{"name": "Mobile app", "description": ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("projects/add", testutil.OK(json.RawMessage(newProject)))

			res := testutil.Run(t, fake, append([]string{"api", "projects", "add"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "projects/add" || req.Form.Has("id") || req.Form.Get("api_token") != testutil.TestToken {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			for k, want := range tt.form {
				if got := req.Form.Get(k); got != want || (want == "" && req.Form.Has(k)) {
					t.Errorf("%s = %q, want %q", k, got, want)
				}
			}
			if res.Stderr != "Created project Mobile app (id 9001).\n" {
				t.Errorf("stderr = %q", res.Stderr)
			}
			if !tt.json && res.Stdout != "" {
				t.Errorf("stdout = %q", res.Stdout)
			}
			if tt.json {
				testutil.Golden(t, res.Stdout)
			}
		})
	}
}

func TestAdd_errors(t *testing.T) {
	empty := testutil.NewFakeAPI(t)
	for _, args := range [][]string{{"api", "projects", "add"}, {"api", "projects", "add", " "}, {"api", "projects", "add", "a", "b"},
		{"api", "projects", "add", "--name", " "}, {"api", "projects", "add", "a", "--name", "b"}} {
		if res := testutil.Run(t, empty, args...); res.Err == nil {
			t.Errorf("%v: no error", args)
		}
	}
	both := testutil.Run(t, empty, "api", "projects", "add", "a", "--name", "b")
	if both.Err == nil || both.Err.Error() != "give the project name once: as an argument or with --name, not both" {
		t.Errorf("name twice: %v", both.Err)
	}
	none := testutil.Run(t, empty, "api", "projects", "add")
	if none.Err == nil || none.Err.Error() != "the project name is missing: poeditor api projects add <name> (or --name <name>)" {
		t.Errorf("no name: %v", none.Err)
	}
	if n := len(empty.Requests()); n != 0 {
		t.Errorf("%d requests for bad arguments", n)
	}

	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/add", testutil.Fail("4040", "The name is too long"))
	res := testutil.Run(t, fake, "api", "projects", "add", "x")
	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Message != "The name is too long" || res.Stderr != "" {
		t.Errorf("error = %v, stderr = %q", res.Err, res.Stderr)
	}
	if res := testutil.Run(t, nil, "api", "projects", "add", "x"); !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("not logged in: %v", res.Err)
	}
}
