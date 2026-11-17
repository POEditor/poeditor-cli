package projectflag_test

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

const repoConfig = `projects:
  - id: 7717
    alias: web
    files:
      - path: locales/{lang}.json
        format: key_value_json
  - id: 4321
    alias: client
    account: client-a
    files:
      - path: client/{lang}.po
        format: po
`

// factory is what a command gets, over env's config, keyring and work dir.
func factory(env *testutil.Env, account string) *cmdutil.Factory {
	ios, _, _, _ := iostreams.Test()
	return &cmdutil.Factory{
		IO:        ios,
		BaseURL:   "http://127.0.0.1:1/",
		ConfigDir: env.ConfigDir,
		WorkDir:   env.WorkDir,
		Getenv:    func(string) string { return "" },
		Keyring:   env.Keyring,
		Account:   account,
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		config      bool // a .poeditor.yml in the work dir
		value       string
		account     string // --account
		wantID      int
		wantLabel   string
		wantAccount string
	}{
		{"alias from the file", true, "web", "", 7717, "web", "default"},
		{"id from the file is labelled by its alias", true, "7717", "", 7717, "web", "default"},
		{"the project's account", true, "client", "", 4321, "client", "client-a"},
		{"--account wins over the project's account", true, "client", "default", 4321, "client", "default"},
		{"an id not in the file", true, "999", "", 999, "999", "default"},
		{"an id without a file", false, "999", "", 999, "999", "default"},
		{"--account with an id", false, "999", "client-a", 999, "999", "client-a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := testutil.NewEnv(t, nil).Login("default", "tok-default").Login("client-a", "tok-client")
			if tt.config {
				env.WriteRepoConfig(repoConfig)
			}

			p, err := projectflag.Resolve(factory(env, tt.account), tt.value)

			if err != nil {
				t.Fatal(err)
			}
			if p.ID != tt.wantID || p.Label != tt.wantLabel || p.Account != tt.wantAccount || p.Client == nil {
				t.Errorf("got id %d, label %q, account %q; want %d, %q, %q", p.ID, p.Label, p.Account, tt.wantID, tt.wantLabel, tt.wantAccount)
			}
			wantRef := repoconfig.ProjectRef{ID: tt.wantID}
			if tt.wantLabel != strconv.Itoa(tt.wantID) {
				wantRef.Alias = tt.wantLabel
			}
			if p.Ref != wantRef {
				t.Errorf("ref = %+v, want %+v", p.Ref, wantRef)
			}
		})
	}
}

// A .poeditor.yml that can't be read does not block a plain id, which does
// not need it: a warning, then the default account. An alias still needs it.
func TestResolve_unreadableConfig(t *testing.T) {
	broken := "projects:\n  - id: 7717\n    name: web\n    files: []\n"

	t.Run("an id goes on, with a warning", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).Login("default", "tok")
		env.WriteRepoConfig(broken)
		f := factory(env, "")

		p, err := projectflag.Resolve(f, "327059")

		if err != nil {
			t.Fatal(err)
		}
		if p.ID != 327059 || p.Account != "default" {
			t.Errorf("got %+v", p)
		}
		stderr := f.IO.ErrOut.(*bytes.Buffer).String()
		if !strings.HasPrefix(stderr, "warning: ") || !strings.Contains(stderr, "ignored") || !strings.Contains(stderr, `was renamed to "alias"`) {
			t.Errorf("stderr = %q", stderr)
		}
	})

	t.Run("an alias needs the file", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).Login("default", "tok")
		env.WriteRepoConfig(broken)

		_, err := projectflag.Resolve(factory(env, ""), "web")

		if err == nil || !strings.Contains(err.Error(), `was renamed to "alias"`) {
			t.Errorf("err = %v", err)
		}
	})
}

func TestResolve_errors(t *testing.T) {
	tests := []struct {
		name   string
		config bool
		value  string
		login  bool
		want   string
	}{
		{"unknown alias lists the file's projects", true, "mobile", true, `no project "mobile"`},
		{"an alias needs the file", false, "web", true, "not a project id, and there is no .poeditor.yml"},
		{"zero is not an id", false, "0", true, "not a project id"},
		{"not logged in", false, "999", false, accounts.ErrNotLoggedIn.Error()},
		{"not given", false, "", true, "required flag \"--project\" not set"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := testutil.NewEnv(t, nil)
			if tt.login {
				env.Login("default", "tok")
			}
			if tt.config {
				env.WriteRepoConfig(repoConfig)
			}

			_, err := projectflag.Resolve(factory(env, ""), tt.value)

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}
