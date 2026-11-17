package contributors_test

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
		want       map[string]string // form, besides id, name, email ("" = not sent)
		wantStderr string
	}{
		{"contributor", []string{"--project", "7717", "--language", "fr"},
			map[string]string{"language": "fr", "proofreader": "", "admin": ""}, "Added ana@example.com to project 7717 as fr contributor.\n"},
		{"proofreader", []string{"--id", "7717", "-l", "fr", "--proofreader"},
			map[string]string{"language": "fr", "proofreader": "1", "admin": ""}, "Added ana@example.com to project 7717 as fr proofreader.\n"},
		{"administrator", []string{"-p", "7717", "--admin"},
			map[string]string{"language": "", "proofreader": "", "admin": "1"}, "Added ana@example.com to project 7717 as administrator.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("contributors/add", testutil.OK(nil))

			res := testutil.Run(t, fake, append([]string{"api", "contributors", "add", "ana@example.com", "--name", "Ana Pop"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "contributors/add" || req.Form.Get("id") != "7717" || req.Form.Get("name") != "Ana Pop" ||
				req.Form.Get("email") != "ana@example.com" || req.Form.Get("api_token") != testutil.TestToken {
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

func TestAdd_badInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no language or admin", []string{"ana@example.com", "--name", "Ana", "-p", "7717"}, "give --language (a contributor of that language) or --admin"},
		{"language and admin", []string{"ana@example.com", "--name", "Ana", "-p", "7717", "-l", "fr", "--admin"}, "none of the others can be"},
		{"proofreader and admin", []string{"ana@example.com", "--name", "Ana", "-p", "7717", "--proofreader", "--admin"}, "none of the others can be"},
		{"no name", []string{"ana@example.com", "-p", "7717", "-l", "fr"}, `required flag(s) "name" not set`},
		{"a blank name", []string{"ana@example.com", "--name", " ", "-p", "7717", "-l", "fr"}, "--name is empty"},
		{"not an email", []string{"ana", "--name", "Ana", "-p", "7717", "-l", "fr"}, `"ana" is not an email address`},
		{"no email", []string{"--name", "Ana", "-p", "7717", "-l", "fr"}, "accepts 1 arg(s), received 0"},
		{"no project", []string{"ana@example.com", "--name", "Ana", "-l", "fr"}, `required flag "--project" not set`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)

			res := testutil.Run(t, fake, append([]string{"api", "contributors", "add"}, tt.args...)...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d API requests, want none", n)
			}
		})
	}
}

func TestAdd_apiErrors(t *testing.T) {
	t.Run("the API's reason", func(t *testing.T) {
		fake := testutil.NewFakeAPI(t)
		fake.Stub("contributors/add", testutil.Fail("4040", "Only the project owner can add an administrator."))

		res := testutil.Run(t, fake, "api", "contributors", "add", "bo@example.com", "--name", "Bo", "-p", "7717", "--admin")

		var apiErr *api.Error
		if !errors.As(res.Err, &apiErr) || apiErr.Message != "Only the project owner can add an administrator." || res.Stderr != "" {
			t.Errorf("error = %v, stderr = %q", res.Err, res.Stderr)
		}
	})
	t.Run("a wrong language", func(t *testing.T) {
		fake := testutil.NewFakeAPI(t)
		fake.Stub("contributors/add", testutil.Fail(api.CodeUnknownLanguage, "Wrong language code"))

		res := testutil.Run(t, fake, "api", "contributors", "add", "ana@example.com", "--name", "Ana", "-p", "7717", "-l", "frr")

		if res.Err == nil || !strings.HasPrefix(res.Err.Error(), "--language frr: not a language code POEditor knows") {
			t.Errorf("error = %v", res.Err)
		}
	})
	t.Run("not logged in", func(t *testing.T) {
		res := testutil.Run(t, nil, "api", "contributors", "add", "ana@example.com", "--name", "Ana", "-p", "7717", "-l", "fr")

		if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
			t.Errorf("error = %v", res.Err)
		}
	})
}
