package projects_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestView_output(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		result string
		tty    bool
		width  int // 0: not fitted
	}{
		{"terminal", []string{"--project", "web"}, webProject, true, 0},
		{"terminal_narrow", []string{"--project", "web"}, webProject, true, 100},
		{"terminal_new", []string{"--id", "9001"}, newProject, true, 0},
		{"piped", []string{"-p", "web"}, webProject, false, 0},
		{"json", []string{"-p", "web", "--json"}, webProject, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("projects/view", testutil.OK(json.RawMessage(tt.result)))
			e := env(t, fake)
			e.TTY = tt.tty
			e.Width = tt.width

			res := e.Run(append([]string{"api", "projects", "view"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "projects/view" || req.Form.Get("api_token") != testutil.TestToken || (req.Form.Get("id") != "7717" && req.Form.Get("id") != "9001") {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			testutil.Golden(t, res.Stdout)
		})
	}
}

func TestView_errors(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/view", testutil.Fail("4012", "Invalid project ID"))

	res := testutil.Run(t, fake, "api", "projects", "view", "-p", "1")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4012" || res.Stdout != "" {
		t.Errorf("error = %v, stdout = %q", res.Err, res.Stdout)
	}
	if res := testutil.Run(t, nil, "api", "projects", "view", "-p", "1"); !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("not logged in: %v", res.Err)
	}
	empty := testutil.NewFakeAPI(t)
	if res := testutil.Run(t, empty, "api", "projects", "view"); res.Err == nil || len(empty.Requests()) != 0 {
		t.Errorf("no --project: %v", res.Err)
	}
}
