package languages_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

// Without --yes nothing is sent, nor is an account needed; the error gives
// the line to run.
func TestDelete_needsYes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"by id", []string{"fr", "--project", "7717"},
			"poeditor api languages delete fr --project 7717 --yes"},
		{"--id and --account are kept", []string{"pt-br", "--id", "7717", "--account", "client a"},
			"poeditor api languages delete pt-br --project 7717 --account 'client a' --yes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)

			res := testutil.Run(t, fake, append([]string{"api", "languages", "delete"}, tt.args...)...)

			want := "deleting " + tt.args[0] + " deletes all its translations and cannot be undone; nothing was deleted. To delete it, run:\n  " + tt.want
			if res.Err == nil || res.Err.Error() != want {
				t.Errorf("error = %v\nwant    %s", res.Err, want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d requests sent without --yes", n)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/delete", testutil.OK(nil))

	res := testutil.Run(t, fake, "api", "languages", "delete", "fr", "--project", "7717", "--yes")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	req := fake.OnlyRequest(t)
	if req.Endpoint != "languages/delete" || req.Form.Get("id") != "7717" || req.Form.Get("language") != "fr" ||
		req.Form.Get("api_token") != testutil.TestToken || req.Form.Has("yes") {
		t.Errorf("request = %s %v", req.Endpoint, req.Form)
	}
	if res.Stdout != "" || res.Stderr != "Deleted fr from project 7717.\n" {
		t.Errorf("stdout = %q, stderr = %q", res.Stdout, res.Stderr)
	}
}

// A language the project does not have: the error lists the ones it has.
func TestDelete_notInProject(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/delete", testutil.Fail(api.CodeLanguageNotInProject, "Project not containing specified language"))
	fake.Stub("languages/list", testutil.OK(map[string]any{"languages": []map[string]any{{"code": "fr"}, {"code": "en"}}}))

	res := testutil.Run(t, fake, "api", "languages", "delete", "de", "-p", "7717", "--yes")

	if want := "de: project 7717 has no such language (it has: en, fr)"; res.Err == nil || res.Err.Error() != want {
		t.Errorf("error = %v, want %q", res.Err, want)
	}
	if res.Stderr != "" {
		t.Errorf("stderr = %q", res.Stderr)
	}
}

func TestDelete_apiFailure(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/delete", testutil.Fail("4012", "Invalid project ID"))

	res := testutil.Run(t, fake, "api", "languages", "delete", "fr", "-p", "7717", "--yes")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4012" {
		t.Fatalf("want API error 4012, got %v", res.Err)
	}
}

func TestDelete_notLoggedIn(t *testing.T) {
	res := testutil.Run(t, nil, "api", "languages", "delete", "fr", "-p", "7717", "--yes")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("error = %v, want ErrNotLoggedIn", res.Err)
	}
}

func TestDelete_badArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no code", []string{"api", "languages", "delete", "-p", "7717", "--yes"}, "accepts 1 arg(s), received 0"},
		{"two codes", []string{"api", "languages", "delete", "fr", "de", "-p", "7717", "--yes"}, "accepts 1 arg(s), received 2"},
		{"--project is required, before --yes", []string{"api", "languages", "delete", "fr"}, `required flag "--project" not set`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)

			res := testutil.Run(t, fake, tt.args...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d API requests, want none", n)
			}
		})
	}
}
