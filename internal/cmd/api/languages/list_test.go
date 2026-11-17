package languages_test

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

// languages/list as the API sends it (ApiV2_Languages_List): a float
// percentage, 100.0, a numeric string (the server's declared float|int|string),
// and a language nothing was translated in yet (int 0, updated "").
const languagesResult = `{"languages":[` +
	`{"name":"French","code":"fr","translations":10,"percentage":83.33,"updated":"2026-03-01T08:00:00+0000"},` +
	`{"name":"German","code":"de","translations":12,"percentage":100.0,"updated":"2026-02-11T10:15:00+0000"},` +
	`{"name":"Portuguese (BR)","code":"pt-br","translations":6,"percentage":"50","updated":"2026-01-14T09:12:44+0000"},` +
	`{"name":"Romanian","code":"ro","translations":0,"percentage":0,"updated":""}]}`

const emptyResult = `{"languages":[]}`

func TestList_output(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		result string
		tty    bool
	}{
		{"table", []string{"--project", "7717"}, languagesResult, true},
		{"table_piped", []string{"--project", "7717"}, languagesResult, false},
		{"table_empty", []string{"--project", "7717"}, emptyResult, true},
		{"table_empty_piped", []string{"--project", "7717"}, emptyResult, false},
		{"json", []string{"-p", "7717", "--json"}, languagesResult, true},
		{"json_empty", []string{"-p", "7717", "--json"}, emptyResult, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("languages/list", testutil.OK(json.RawMessage(tt.result)))
			env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
			env.TTY = tt.tty

			res := env.Run(append([]string{"api", "languages", "list"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "languages/list" || req.Form.Get("api_token") != testutil.TestToken || req.Form.Get("id") != "7717" {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			if res.Stderr != "" {
				t.Errorf("stderr = %q", res.Stderr)
			}
			testutil.Golden(t, res.Stdout)
		})
	}
}

// --id, the API docs' name, is a hidden synonym of --project.
func TestList_idSynonym(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/list", testutil.OK(json.RawMessage(emptyResult)))

	res := testutil.Run(t, fake, "api", "languages", "list", "--id", "327059")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if req := fake.OnlyRequest(t); req.Form.Get("id") != "327059" {
		t.Errorf("form = %v", req.Form)
	}
}

// --project takes an alias from .poeditor.yml, and its account.
func TestList_projectAlias(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/list", testutil.OK(json.RawMessage(emptyResult)))
	env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken).Login("client-a", "client-token")
	env.WriteRepoConfig(`projects:
  - id: 4321
    alias: client
    account: client-a
    files:
      - path: locales/{lang}.json
        format: key_value_json
`)

	res := env.Run("api", "languages", "list", "--project", "client")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	req := fake.OnlyRequest(t)
	if req.Form.Get("id") != "4321" || req.Form.Get("api_token") != "client-token" {
		t.Errorf("want project 4321 with client-a's token, got %v", req.Form)
	}
	if res.Stdout != "Project client has no languages.\n" {
		t.Errorf("stdout = %q", res.Stdout)
	}
}

// A percentage that is not a number fails the call, saying which field.
func TestList_badPercentage(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/list", testutil.OK(json.RawMessage(`{"languages":[{"code":"fr","percentage":"n/a"}]}`)))

	res := testutil.Run(t, fake, "api", "languages", "list", "-p", "7717")

	if res.Err == nil || !strings.Contains(res.Err.Error(), "percentage") {
		t.Errorf("error = %v, want it to name the percentage", res.Err)
	}
	if res.Stdout != "" {
		t.Errorf("stdout = %q", res.Stdout)
	}
}

func TestList_apiFailure(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/list", testutil.Fail("4012", "Invalid project ID"))

	res := testutil.Run(t, fake, "api", "languages", "list", "-p", "7717")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4012" {
		t.Fatalf("want API error 4012, got %v", res.Err)
	}
	if res.Stdout != "" {
		t.Errorf("stdout = %q", res.Stdout)
	}
}

func TestList_notLoggedIn(t *testing.T) {
	res := testutil.Run(t, nil, "api", "languages", "list", "-p", "7717")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("error = %v, want ErrNotLoggedIn", res.Err)
	}
}

func TestList_badArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"--project is required", []string{"api", "languages", "list"}, `required flag "--project" not set`},
		{"no arguments", []string{"api", "languages", "list", "-p", "7717", "extra"}, `unknown command "extra"`},
		{"an alias needs .poeditor.yml", []string{"api", "languages", "list", "-p", "web"}, "not a project id, and there is no .poeditor.yml"},
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
