package auth_test

import (
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/config"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestList(t *testing.T) {
	t.Run("no accounts", func(t *testing.T) {
		res := testutil.NewEnv(t, nil).Run("auth", "list")

		if res.Err != nil || res.Stdout != "No accounts. Run `poeditor auth login`.\n" {
			t.Fatalf("got %q, %v", res.Stdout, res.Err)
		}
	})

	t.Run("no accounts, token in environment", func(t *testing.T) {
		env := testutil.NewEnv(t, nil)
		env.Vars["POEDITOR_TOKEN"] = "x"

		res := env.Run("auth", "list")

		if res.Err != nil || res.Stdout != "No accounts; commands use POEDITOR_TOKEN from the environment.\n" {
			t.Fatalf("got %q, %v", res.Stdout, res.Err)
		}
	})

	t.Run("every token source", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).Login("personal", "p").Login("ci", "c")
		env.Vars["POEDITOR_TOKEN_CI"] = "from-env"
		cfg := env.Config()
		cfg.Accounts["offline"] = &config.Account{Token: "in-file"}
		cfg.Accounts["broken"] = &config.Account{}
		if err := cfg.Save(); err != nil {
			t.Fatal(err)
		}

		res := env.Run("auth", "list")

		if res.Err != nil {
			t.Fatal(res.Err)
		}
		testutil.Golden(t, res.Stdout)
	})
}

func TestDefault(t *testing.T) {
	t.Run("switches", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).Login("personal", "p").Login("client-a", "c")

		res := env.Run("auth", "default", "client-a")

		if res.Err != nil {
			t.Fatal(res.Err)
		}
		if res.Stderr != "Default account: \"client-a\".\n" {
			t.Errorf("stderr = %q", res.Stderr)
		}
		if got := env.Config().DefaultAccount; got != "client-a" {
			t.Errorf("default = %q", got)
		}
	})

	tests := []struct {
		name     string
		accounts []string
		args     []string
		wantErr  string
	}{
		{"unknown account", []string{"personal", "ci"}, []string{"auth", "default", "nope"}, `account "nope" isn't set up (accounts: ci, personal)`},
		{"no accounts yet", nil, []string{"auth", "default", "nope"}, "account \"nope\" isn't set up, and there are no accounts yet. Run `poeditor auth login --account nope`"},
		{"missing argument", []string{"personal"}, []string{"auth", "default"}, "accepts 1 arg(s), received 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := testutil.NewEnv(t, nil)
			for _, name := range tt.accounts {
				env.Login(name, "t")
			}
			before := env.Config().DefaultAccount

			res := env.Run(tt.args...)

			if res.Err == nil || res.Err.Error() != tt.wantErr {
				t.Fatalf("error = %v, want %q", res.Err, tt.wantErr)
			}
			if got := env.Config().DefaultAccount; got != before {
				t.Errorf("default changed to %q", got)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	t.Run("only account, no flag", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).Login("personal", "p")

		res := env.Run("auth", "logout")

		if res.Err != nil {
			t.Fatal(res.Err)
		}
		if res.Stderr != "Logged out of \"personal\".\n" {
			t.Errorf("stderr = %q", res.Stderr)
		}
		assertKeyring(t, env, "personal", "")
		if cfg := env.Config(); len(cfg.Accounts) != 0 || cfg.DefaultAccount != "" {
			t.Errorf("config = %+v", cfg)
		}
	})

	t.Run("default account moves to the first remaining", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).Login("personal", "p").Login("zeta", "z").Login("acme", "a")

		res := env.Run("auth", "logout", "--account", "personal")

		if res.Err != nil {
			t.Fatal(res.Err)
		}
		want := "Logged out of \"personal\".\n\"acme\" is now your default account.\n"
		if res.Stderr != want {
			t.Errorf("stderr = %q, want %q", res.Stderr, want)
		}
		if got := env.Config().DefaultAccount; got != "acme" {
			t.Errorf("default = %q", got)
		}
		assertKeyring(t, env, "zeta", "z")
	})

	t.Run("token kept in the config file (no keyring)", func(t *testing.T) {
		env := testutil.NewEnv(t, nil)
		cfg := env.Config()
		cfg.DefaultAccount = "offline"
		cfg.Accounts = map[string]*config.Account{"offline": {Token: "in-file"}}
		if err := cfg.Save(); err != nil {
			t.Fatal(err)
		}
		env.Keyring.Unavailable = true

		res := env.Run("auth", "logout")

		if res.Err != nil {
			t.Fatal(res.Err)
		}
		if strings.Contains(res.Stderr, "Warning") {
			t.Errorf("no warning expected, the token was in the file: %q", res.Stderr)
		}
		if env.Config().HasAccount("offline") {
			t.Error("account still in config")
		}
	})

	t.Run("keyring token that cannot be removed", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).Login("personal", "p")
		env.Keyring.Unavailable = true

		res := env.Run("auth", "logout")

		if res.Err != nil {
			t.Fatal(res.Err)
		}
		if !strings.Contains(res.Stderr, `Warning: could not remove the token from the system keyring (keyring unavailable); delete the "personal" entry for service "poeditor-cli" by hand.`) {
			t.Errorf("stderr = %q", res.Stderr)
		}
	})

	errs := []struct {
		name     string
		accounts []string
		args     []string
		wantErr  string
	}{
		{"not logged in", nil, []string{"auth", "logout"}, "not logged in to any account"},
		{"several accounts, no flag", []string{"personal", "ci"}, []string{"auth", "logout"}, "you have several accounts (ci, personal): say which one with --account <name>"},
		{"unknown account", []string{"personal"}, []string{"auth", "logout", "--account", "nope"}, `account "nope" isn't set up`},
	}
	for _, tt := range errs {
		t.Run(tt.name, func(t *testing.T) {
			env := testutil.NewEnv(t, nil)
			for _, name := range tt.accounts {
				env.Login(name, "t")
			}

			res := env.Run(tt.args...)

			if res.Err == nil || res.Err.Error() != tt.wantErr {
				t.Fatalf("error = %v, want %q", res.Err, tt.wantErr)
			}
			if got := len(env.Config().Accounts); got != len(tt.accounts) {
				t.Errorf("%d accounts left, want %d", got, len(tt.accounts))
			}
		})
	}
}
