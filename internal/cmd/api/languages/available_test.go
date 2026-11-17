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

// languages/available as the API sends it (ApiV2_Languages_Available), a
// custom language among them.
const availableResult = `{"languages":[` +
	`{"name":"English","code":"en"},` +
	`{"name":"Portuguese (BR)","code":"pt-br"},` +
	`{"name":"Klingon (custom)","code":"x-tlh"}]}`

func TestAvailable_output(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		result string
		tty    bool
	}{
		{"table", nil, availableResult, true},
		{"table_piped", nil, availableResult, false},
		{"table_empty", nil, emptyResult, true},
		{"table_empty_piped", nil, emptyResult, false},
		{"json", []string{"--json"}, availableResult, true},
		{"json_empty", []string{"--json"}, emptyResult, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("languages/available", testutil.OK(json.RawMessage(tt.result)))
			env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
			env.TTY = tt.tty

			res := env.Run(append([]string{"api", "languages", "available"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "languages/available" || req.Form.Get("api_token") != testutil.TestToken {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			if len(req.Form) != 1 {
				t.Errorf("form = %v, want only api_token", req.Form)
			}
			if res.Stderr != "" {
				t.Errorf("stderr = %q", res.Stderr)
			}
			testutil.Golden(t, res.Stdout)
		})
	}
}

// --account picks whose list it is (custom languages are per account).
func TestAvailable_account(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/available", testutil.OK(json.RawMessage(emptyResult)))
	env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken).Login("client-a", "client-token")

	res := env.Run("api", "languages", "available", "--account", "client-a")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if got := fake.OnlyRequest(t).Form.Get("api_token"); got != "client-token" {
		t.Errorf("token = %q, want client-a's", got)
	}
}

func TestAvailable_apiFailure(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/available", testutil.Fail("4011", "Invalid API Token"))

	res := testutil.Run(t, fake, "api", "languages", "available")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4011" {
		t.Fatalf("want API error 4011, got %v", res.Err)
	}
	if res.Stdout != "" {
		t.Errorf("stdout = %q", res.Stdout)
	}
}

func TestAvailable_notLoggedIn(t *testing.T) {
	res := testutil.Run(t, nil, "api", "languages", "available")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("error = %v, want ErrNotLoggedIn", res.Err)
	}
}

func TestAvailable_badArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no arguments", []string{"api", "languages", "available", "extra"}, `unknown command "extra"`},
		{"no --project", []string{"api", "languages", "available", "-p", "7717"}, "unknown shorthand flag: 'p'"},
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
