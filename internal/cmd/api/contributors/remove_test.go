package contributors_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

// Without --yes nothing is sent, nor is an account needed.
func TestRemove_needsYes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"everything", []string{"--project", "web"},
			"this removes ana@example.com from project web; nothing was removed. To remove it, run:\n  poeditor api contributors remove ana@example.com --project web --yes"},
		{"one language", []string{"--id", "7717", "-l", "fr", "--account", "client-a"},
			"this removes ana@example.com from fr in project 7717; nothing was removed. To remove it, run:\n  poeditor api contributors remove ana@example.com --project 7717 --language fr --account client-a --yes"},
		{"the proofreader right", []string{"-p", "web", "--proofreader"},
			"this removes the proofreader right of ana@example.com in project web; nothing was removed. To remove it, run:\n  poeditor api contributors remove ana@example.com --project web --proofreader --yes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)

			res := testutil.NewEnv(t, fake).Run(append([]string{"api", "contributors", "remove", "ana@example.com"}, tt.args...)...)

			if res.Err == nil || res.Err.Error() != tt.want {
				t.Errorf("error = %v\nwant    %s", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d requests sent without --yes", n)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		want       map[string]string
		wantStderr string
	}{
		{"everything", nil, map[string]string{"language": "", "proofreader": ""}, "Removed ana@example.com from project 7717.\n"},
		{"one language", []string{"-l", "fr"}, map[string]string{"language": "fr", "proofreader": ""}, "Removed ana@example.com from fr in project 7717.\n"},
		{"the proofreader right", []string{"--proofreader"}, map[string]string{"language": "", "proofreader": "1"}, "Removed the proofreader right of ana@example.com in project 7717.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("contributors/remove", testutil.OK(nil))

			res := testutil.Run(t, fake, append([]string{"api", "contributors", "remove", "ana@example.com", "-p", "7717", "--yes"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "contributors/remove" || req.Form.Get("id") != "7717" || req.Form.Get("email") != "ana@example.com" || req.Form.Has("yes") {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			for k, want := range tt.want {
				if got := req.Form.Get(k); got != want || (want == "" && req.Form.Has(k)) {
					t.Errorf("%s = %q, want %q", k, got, want)
				}
			}
			if res.Stdout != "" || res.Stderr != tt.wantStderr {
				t.Errorf("stdout = %q, stderr = %q, want %q", res.Stdout, res.Stderr, tt.wantStderr)
			}
		})
	}
}

func TestRemove_badInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"--project first", []string{"ana@example.com", "--yes"}, `required flag "--project" not set`},
		{"not an email", []string{"ana", "-p", "web", "--yes"}, `"ana" is not an email address`},
		{"language and proofreader", []string{"ana@example.com", "-p", "web", "-l", "fr", "--proofreader"}, "none of the others can be"},
		{"no email", []string{"-p", "web"}, "accepts 1 arg(s), received 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)

			res := testutil.Run(t, fake, append([]string{"api", "contributors", "remove"}, tt.args...)...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d API requests, want none", n)
			}
		})
	}
}

func TestRemove_apiErrors(t *testing.T) {
	t.Run("the API's reason", func(t *testing.T) {
		fake := testutil.NewFakeAPI(t)
		fake.Stub("contributors/remove", testutil.Fail("4040", "The contributor is not assigned to this project."))

		res := testutil.Run(t, fake, "api", "contributors", "remove", "ana@example.com", "-p", "7717", "--yes")

		var apiErr *api.Error
		if !errors.As(res.Err, &apiErr) || apiErr.Message != "The contributor is not assigned to this project." || res.Stderr != "" {
			t.Errorf("error = %v, stderr = %q", res.Err, res.Stderr)
		}
	})
	t.Run("not logged in", func(t *testing.T) {
		res := testutil.Run(t, nil, "api", "contributors", "remove", "ana@example.com", "-p", "7717", "--yes")

		if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
			t.Errorf("error = %v", res.Err)
		}
	})
}
