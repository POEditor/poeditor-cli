package auth_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/keyring"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestMain(m *testing.M) { testutil.Main(m) }

func projects(n int) map[string]any {
	list := make([]map[string]any, n)
	for i := range list {
		list[i] = map[string]any{"id": i + 1, "name": "p", "public": 0, "open": 0, "created": ""}
	}
	return map[string]any{"projects": list}
}

func assertKeyring(t *testing.T, env *testutil.Env, account, want string) {
	t.Helper()
	got, err := env.Keyring.Get(account)
	if want == "" {
		if !errors.Is(err, keyring.ErrNotFound) {
			t.Errorf("keyring[%s] = %q, %v; want no entry", account, got, err)
		}
		return
	}
	if err != nil || got != want {
		t.Errorf("keyring[%s] = %q, %v; want %q", account, got, err, want)
	}
}

func TestLogin_firstAccountWithPrompt(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/list", testutil.OK(projects(2)))
	env := testutil.NewEnv(t, fake)
	env.Stdin = "  tok-1  \n"

	res := env.Run("auth", "login")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if got := fake.OnlyRequest(t).Form.Get("api_token"); got != "tok-1" {
		t.Errorf("validated token = %q, want tok-1 (trimmed)", got)
	}
	want := "Paste your API token (create one at https://poeditor.com/account/api): " +
		"Logged in to account \"default\" (2 projects).\n" +
		"Token stored in the system keyring.\n" +
		"\"default\" is your default account.\n"
	if res.Stderr != want {
		t.Errorf("stderr:\n%s\nwant:\n%s", res.Stderr, want)
	}
	if res.Stdout != "" {
		t.Errorf("stdout = %q", res.Stdout)
	}
	assertKeyring(t, env, "default", "tok-1")
	cfg := env.Config()
	if cfg.DefaultAccount != "default" || cfg.Accounts["default"] == nil || cfg.Accounts["default"].Token != "" {
		t.Errorf("config = %+v", cfg)
	}
}

func TestLogin_withTokenAndAccount(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/list", testutil.OK(projects(1)))
	env := testutil.NewEnv(t, fake)
	env.Stdin = "tok-ci"

	res := env.Run("auth", "login", "--account", "ci", "--with-token")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if strings.Contains(res.Stderr, "Paste your API token") {
		t.Error("--with-token must not prompt")
	}
	if !strings.HasPrefix(res.Stderr, "Logged in to account \"ci\" (1 project).\n") {
		t.Errorf("stderr = %q", res.Stderr)
	}
	assertKeyring(t, env, "ci", "tok-ci")
	if cfg := env.Config(); cfg.DefaultAccount != "ci" {
		t.Errorf("default = %q, want ci", cfg.DefaultAccount)
	}
}

func TestLogin_secondAccountNeedsAName(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	env := testutil.NewEnv(t, fake).Login("personal", "p")
	env.Stdin = "tok-2\n"

	res := env.Run("auth", "login")

	want := "you already have an account (personal): use --account <name> to add another one, or --account personal to replace its token"
	if res.Err == nil || res.Err.Error() != want {
		t.Fatalf("error = %v, want %q", res.Err, want)
	}
	if n := len(fake.Requests()); n != 0 {
		t.Errorf("no API call expected, got %d", n)
	}
	assertKeyring(t, env, "personal", "p")
}

func TestLogin_addSecondAccountKeepsDefault(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/list", testutil.OK(projects(3)))
	env := testutil.NewEnv(t, fake).Login("personal", "p")
	env.Stdin = "c\n"

	res := env.Run("auth", "login", "--account", "client-a")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if strings.Contains(res.Stderr, "default account") {
		t.Errorf("default must not change: %q", res.Stderr)
	}
	cfg := env.Config()
	if cfg.DefaultAccount != "personal" || !cfg.HasAccount("client-a") {
		t.Errorf("config = %+v", cfg)
	}
	assertKeyring(t, env, "client-a", "c")
	assertKeyring(t, env, "personal", "p")
}

func TestLogin_replaceToken(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/list", testutil.OK(projects(0)))
	env := testutil.NewEnv(t, fake).Login("personal", "old")
	env.Stdin = "new\n"

	res := env.Run("auth", "login", "--account", "personal")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Stderr, "Updated the token of account \"personal\" (0 projects).") {
		t.Errorf("stderr = %q", res.Stderr)
	}
	assertKeyring(t, env, "personal", "new")
}

func TestLogin_rejectedTokenSavesNothing(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/list", testutil.Fail("4011", "Invalid API Token"))
	env := testutil.NewEnv(t, fake)
	env.Stdin = "bad\n"

	res := env.Run("auth", "login")

	want := "token not accepted, nothing was saved: API error 4011: Invalid API Token"
	if res.Err == nil || res.Err.Error() != want {
		t.Fatalf("error = %v, want %q", res.Err, want)
	}
	assertKeyring(t, env, "default", "")
	if cfg := env.Config(); len(cfg.Accounts) != 0 || cfg.DefaultAccount != "" {
		t.Errorf("config should stay empty: %+v", cfg)
	}
}

func TestLogin_badInputMakesNoAPICall(t *testing.T) {
	tests := []struct {
		name, stdin string
		args        []string
		wantErr     string
	}{
		{"empty token", "\n", []string{"auth", "login"}, "no token given"},
		{"empty stdin with --with-token", "", []string{"auth", "login", "--with-token"}, "no token given"},
		{"invalid account name", "tok\n", []string{"auth", "login", "--account", "Client A"}, `invalid account name "Client A"`},
		{"extra argument", "tok\n", []string{"auth", "login", "extra"}, `unknown command "extra"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			env := testutil.NewEnv(t, fake)
			env.Stdin = tt.stdin

			res := env.Run(tt.args...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %q", res.Err, tt.wantErr)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("no API call expected, got %d", n)
			}
		})
	}
}

func TestLogin_noKeyringFallsBackToConfigFile(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/list", testutil.OK(projects(1)))
	env := testutil.NewEnv(t, fake)
	env.Keyring.Unavailable = true
	env.Stdin = "tok\n"

	res := env.Run("auth", "login")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	cfg := env.Config()
	if acc := cfg.Accounts["default"]; acc == nil || acc.Token != "tok" {
		t.Fatalf("token not in config file: %+v", cfg.Accounts)
	}
	want := "Warning: no system keyring available (keyring unavailable); the token is stored in plain text in " + cfg.Path() + ".\n"
	if !strings.Contains(res.Stderr, want) {
		t.Errorf("stderr = %q, want it to contain %q", res.Stderr, want)
	}
}
