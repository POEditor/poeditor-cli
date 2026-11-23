package projects_test

import (
	"errors"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestDelete_needsYes(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"--project", "web"}, "deleting project web deletes all its terms and translations and cannot be undone; nothing was deleted. To delete it, run:\n  poeditor api projects delete --project web --yes"},
		{[]string{"--id", "7717", "--account", "client a"}, "deleting project 7717 deletes all its terms and translations and cannot be undone; nothing was deleted. To delete it, run:\n  poeditor api projects delete --project 7717 --account 'client a' --yes"},
	}
	for _, tt := range tests {
		fake := testutil.NewFakeAPI(t)

		res := testutil.NewEnv(t, fake).Run(append([]string{"api", "projects", "delete"}, tt.args...)...)

		if res.Err == nil || res.Err.Error() != tt.want {
			t.Errorf("error = %v\nwant    %s", res.Err, tt.want)
		}
		if n := len(fake.Requests()); n != 0 {
			t.Errorf("%d requests sent without --yes", n)
		}
	}
}

func TestDelete(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/delete", testutil.OK(nil))

	res := env(t, fake).Run("api", "projects", "delete", "-p", "web", "--yes")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if req := fake.OnlyRequest(t); req.Endpoint != "projects/delete" || req.Form.Get("id") != "7717" || req.Form.Has("yes") {
		t.Errorf("request = %s %v", req.Endpoint, req.Form)
	}
	if res.Stdout != "" || res.Stderr != "Deleted project web.\n" {
		t.Errorf("stdout = %q, stderr = %q", res.Stdout, res.Stderr)
	}
}

func TestDelete_errors(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/delete", testutil.Fail("4040", "You must be the owner of the project"))

	res := testutil.Run(t, fake, "api", "projects", "delete", "-p", "7717", "--yes")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Message != "You must be the owner of the project" || res.Stderr != "" {
		t.Errorf("error = %v, stderr = %q", res.Err, res.Stderr)
	}
	if res := testutil.Run(t, nil, "api", "projects", "delete", "-p", "7717", "--yes"); !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("not logged in: %v", res.Err)
	}
	empty := testutil.NewFakeAPI(t)
	for _, args := range [][]string{{"api", "projects", "delete", "--yes"}, {"api", "projects", "delete", "7717", "9001", "--yes"}} {
		if res := testutil.Run(t, empty, args...); res.Err == nil {
			t.Errorf("%v: no error", args)
		}
	}
	if len(empty.Requests()) != 0 {
		t.Error("bad arguments sent a request")
	}
}
