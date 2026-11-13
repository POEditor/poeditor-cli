package accounts_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/config"
	"github.com/POEditor/poeditor-cli/internal/keyring"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name       string
		requested  string
		def        string
		accounts   []string
		keyring    map[string]string
		fileTokens map[string]string
		env        map[string]string
		wantName   string
		wantToken  string
		wantSource string
		wantErr    string
	}{
		{name: "nothing configured", wantErr: accounts.ErrNotLoggedIn.Error()},
		{name: "bare POEDITOR_TOKEN, no accounts", env: map[string]string{"POEDITOR_TOKEN": "env"},
			wantToken: "env", wantSource: accounts.SourceEnv},
		{name: "default account from keyring", def: "personal", accounts: []string{"personal"},
			keyring: map[string]string{"personal": "kr"}, wantName: "personal", wantToken: "kr", wantSource: accounts.SourceKeyring},
		{name: "--account picks another", requested: "client-a", def: "personal", accounts: []string{"personal", "client-a"},
			keyring: map[string]string{"personal": "p", "client-a": "c"}, wantName: "client-a", wantToken: "c", wantSource: accounts.SourceKeyring},
		{name: "per-account env beats keyring", requested: "client-a", def: "personal", accounts: []string{"personal", "client-a"},
			keyring: map[string]string{"client-a": "kr"}, env: map[string]string{"POEDITOR_TOKEN_CLIENT_A": "env"},
			wantName: "client-a", wantToken: "env", wantSource: accounts.SourceEnv},
		{name: "POEDITOR_TOKEN overrides the default account", def: "personal", accounts: []string{"personal"},
			keyring: map[string]string{"personal": "kr"}, env: map[string]string{"POEDITOR_TOKEN": "env"},
			wantName: "personal", wantToken: "env", wantSource: accounts.SourceEnv},
		{name: "POEDITOR_TOKEN does not apply to other accounts", requested: "client-a", def: "personal", accounts: []string{"personal", "client-a"},
			keyring: map[string]string{"client-a": "kr"}, env: map[string]string{"POEDITOR_TOKEN": "env"},
			wantName: "client-a", wantToken: "kr", wantSource: accounts.SourceKeyring},
		{name: "config file fallback", def: "personal", accounts: []string{"personal"},
			fileTokens: map[string]string{"personal": "file"}, wantName: "personal", wantToken: "file", wantSource: accounts.SourceConfig},
		{name: "env-only account (CI) needs no config", requested: "client-a",
			env: map[string]string{"POEDITOR_TOKEN_CLIENT_A": "env"}, wantName: "client-a", wantToken: "env", wantSource: accounts.SourceEnv},
		{name: "unknown account", requested: "client-a", def: "personal", accounts: []string{"personal"},
			wantErr: "account \"client-a\" isn't set up. Run `poeditor auth login --account client-a`"},
		{name: "account without token", def: "personal", accounts: []string{"personal"},
			wantErr: "account \"personal\" has no token stored. Run `poeditor auth login --account personal`"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{DefaultAccount: tt.def, Accounts: map[string]*config.Account{}}
			for _, name := range tt.accounts {
				cfg.Accounts[name] = &config.Account{Token: tt.fileTokens[name]}
			}
			kr := &keyring.Memory{}
			for name, token := range tt.keyring {
				_ = kr.Set(name, token)
			}
			r := &accounts.Resolver{Config: cfg, Keyring: kr, Getenv: func(k string) string { return tt.env[k] }}

			got, err := r.Resolve(tt.requested)

			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Name != tt.wantName || got.Token != tt.wantToken || got.Source != tt.wantSource {
				t.Errorf("got %+v, want name=%q token=%q source=%q", got, tt.wantName, tt.wantToken, tt.wantSource)
			}
		})
	}
}

func TestResolve_unavailableKeyringFallsBackToFile(t *testing.T) {
	cfg := &config.Config{DefaultAccount: "p", Accounts: map[string]*config.Account{"p": {Token: "file"}}}
	r := &accounts.Resolver{Config: cfg, Keyring: &keyring.Memory{Unavailable: true}, Getenv: func(string) string { return "" }}

	got, err := r.Resolve("")

	if err != nil || got.Token != "file" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestValidateName(t *testing.T) {
	for _, ok := range []string{"default", "client-a", "acme_2"} {
		if err := accounts.ValidateName(ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "Client", "-x", "a b", "a/b", strings.Repeat("a", 41)} {
		if err := accounts.ValidateName(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestEnvVar(t *testing.T) {
	if got := accounts.EnvVar("client-a"); got != "POEDITOR_TOKEN_CLIENT_A" {
		t.Errorf("EnvVar = %q", got)
	}
}

func TestErrNotLoggedInIsSentinel(t *testing.T) {
	r := &accounts.Resolver{Config: &config.Config{}, Keyring: &keyring.Memory{}, Getenv: func(string) string { return "" }}
	if _, err := r.Resolve(""); !errors.Is(err, accounts.ErrNotLoggedIn) {
		t.Fatalf("got %v", err)
	}
}
