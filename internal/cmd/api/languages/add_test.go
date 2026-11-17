package languages_test

import (
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

// addStub answers languages/add as the server does: de is already in the
// project, xx is not a language, anything else is added.
func addStub(form url.Values) testutil.Stub {
	switch form.Get("language") {
	case "de":
		return testutil.Fail(api.CodeLanguageInProject, "Language already in the project")
	case "xx":
		return testutil.Fail(api.CodeUnknownLanguage, "Wrong language code")
	}
	return testutil.OK(nil)
}

func sentLanguages(fake *testutil.FakeAPI) []string {
	var codes []string
	for _, r := range fake.Requests() {
		codes = append(codes, r.Endpoint+" "+r.Form.Get("language"))
	}
	return codes
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantSent   []string
		wantStderr string
		wantErr    string
	}{
		{"one", []string{"fr", "--project", "7717"},
			[]string{"languages/add fr"}, "Added fr to project 7717.\n", ""},
		{"several, a repeat sent once", []string{"fr", "pt-br", "fr", "-p", "7717"},
			[]string{"languages/add fr", "languages/add pt-br"}, "Added fr to project 7717.\nAdded pt-br to project 7717.\n", ""},
		{"already there: a note, not an error", []string{"de", "fr", "-p", "7717"},
			[]string{"languages/add de", "languages/add fr"}, "Project 7717 already has de.\nAdded fr to project 7717.\n", ""},
		{"a wrong code stops; the ones before stay added", []string{"fr", "xx", "es", "-p", "7717"},
			[]string{"languages/add fr", "languages/add xx"}, "Added fr to project 7717.\n",
			"xx: not a language code POEditor knows (e.g. en, fr, pt-br; all of them: `poeditor api languages available`)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.StubFunc("languages/add", addStub)

			res := testutil.Run(t, fake, append([]string{"api", "languages", "add"}, tt.args...)...)

			if tt.wantErr == "" && res.Err != nil {
				t.Fatal(res.Err)
			}
			if tt.wantErr != "" && (res.Err == nil || res.Err.Error() != tt.wantErr) {
				t.Errorf("error = %v, want %q", res.Err, tt.wantErr)
			}
			if got := sentLanguages(fake); !reflect.DeepEqual(got, tt.wantSent) {
				t.Errorf("sent %q, want %q", got, tt.wantSent)
			}
			for _, r := range fake.Requests() {
				if r.Form.Get("id") != "7717" || r.Form.Get("api_token") != testutil.TestToken {
					t.Errorf("form = %v", r.Form)
				}
			}
			if res.Stdout != "" {
				t.Errorf("stdout = %q, want nothing (an action reports on stderr)", res.Stdout)
			}
			if res.Stderr != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", res.Stderr, tt.wantStderr)
			}
		})
	}
}

// --id is the hidden synonym of --project; an alias brings its account.
func TestAdd_projectFlags(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.StubFunc("languages/add", addStub)
	env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken).Login("client-a", "client-token")
	env.WriteRepoConfig(`projects:
  - id: 4321
    alias: client
    account: client-a
    files:
      - path: locales/{lang}.json
        format: key_value_json
`)

	res := env.Run("api", "languages", "add", "fr", "--project", "client")
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	res = env.Run("api", "languages", "add", "es", "--id", "4321")
	if res.Err != nil {
		t.Fatal(res.Err)
	}

	reqs := fake.Requests()
	if len(reqs) != 2 {
		t.Fatalf("%d requests", len(reqs))
	}
	for _, r := range reqs {
		if r.Form.Get("id") != "4321" || r.Form.Get("api_token") != "client-token" {
			t.Errorf("want project 4321 with client-a's token, got %v", r.Form)
		}
	}
	if res.Stderr != "Added es to project client.\n" {
		t.Errorf("stderr = %q", res.Stderr)
	}
}

func TestAdd_apiFailure(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/add", testutil.Fail("4012", "Invalid project ID"))

	res := testutil.Run(t, fake, "api", "languages", "add", "fr", "-p", "7717")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4012" {
		t.Fatalf("want API error 4012, got %v", res.Err)
	}
	if res.Stderr != "" {
		t.Errorf("stderr = %q", res.Stderr)
	}
}

func TestAdd_notLoggedIn(t *testing.T) {
	res := testutil.Run(t, nil, "api", "languages", "add", "fr", "-p", "7717")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("error = %v, want ErrNotLoggedIn", res.Err)
	}
}

func TestAdd_badArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no code", []string{"api", "languages", "add", "-p", "7717"}, "requires at least 1 arg(s)"},
		{"--project is required", []string{"api", "languages", "add", "fr"}, `required flag "--project" not set`},
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
