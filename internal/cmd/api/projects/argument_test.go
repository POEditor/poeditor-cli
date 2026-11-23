package projects_test

import (
	"encoding/json"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/testutil"
)

// view, update and delete act on one project, so they take it as an
// argument too: `poeditor api projects view 7717`, as well as --project.
func TestProjectArgument(t *testing.T) {
	commands := []struct {
		verb, endpoint string
		extra          []string
		result         any
	}{
		{"view", "projects/view", nil, json.RawMessage(webProject)},
		{"update", "projects/update", []string{"--description", "x"}, json.RawMessage(webProject)},
		{"delete", "projects/delete", []string{"--yes"}, nil},
	}
	for _, c := range commands {
		t.Run(c.verb+" by id", func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub(c.endpoint, testutil.OK(c.result))

			res := env(t, fake).Run(append([]string{"api", "projects", c.verb, "7717"}, c.extra...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			if req := fake.OnlyRequest(t); req.Endpoint != c.endpoint || req.Form.Get("id") != "7717" {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
		})
		t.Run(c.verb+" by alias", func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub(c.endpoint, testutil.OK(c.result))

			res := env(t, fake).Run(append([]string{"api", "projects", c.verb, "web"}, c.extra...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			if req := fake.OnlyRequest(t); req.Form.Get("id") != "7717" {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
		})
		bad := []struct {
			name string
			args []string
			want string
		}{
			{"and --project", []string{"7717", "--project", "7717"}, "give the project once: as an argument or with --project, not both"},
			{"two", []string{"7717", "9001"}, "accepts at most 1 arg(s), received 2"},
			{"neither", nil, `required flag "--project" not set (the project's alias in .poeditor.yml, or its id)`},
		}
		for _, b := range bad {
			t.Run(c.verb+" "+b.name, func(t *testing.T) {
				fake := testutil.NewFakeAPI(t)

				res := env(t, fake).Run(append(append([]string{"api", "projects", c.verb}, b.args...), c.extra...)...)

				if res.Err == nil || res.Err.Error() != b.want {
					t.Errorf("error = %v, want %q", res.Err, b.want)
				}
				if n := len(fake.Requests()); n != 0 {
					t.Errorf("%d API requests, want none", n)
				}
			})
		}
	}
}

// Without --yes, the line to run names the project as it was given.
func TestDelete_argumentNeedsYes(t *testing.T) {
	fake := testutil.NewFakeAPI(t)

	res := env(t, fake).Run("api", "projects", "delete", "7717")

	want := "deleting project 7717 deletes all its terms and translations and cannot be undone; nothing was deleted. To delete it, run:\n  poeditor api projects delete --project 7717 --yes"
	if res.Err == nil || res.Err.Error() != want || len(fake.Requests()) != 0 {
		t.Errorf("error = %v\nwant    %s", res.Err, want)
	}
}
