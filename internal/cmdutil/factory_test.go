package cmdutil_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/config"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
	"github.com/POEditor/poeditor-cli/internal/keyring"
)

// overrideURL is any non-production API URL, used to check the base-URL override.
const overrideURL = "https://api.example.test/v2/"

func factory(t *testing.T, env map[string]string) *cmdutil.Factory {
	t.Helper()
	return &cmdutil.Factory{
		ConfigDir: t.TempDir(),
		Getenv:    func(k string) string { return env[k] },
		Keyring:   &keyring.Memory{},
	}
}

func TestAPIClient_notLoggedIn(t *testing.T) {
	_, err := factory(t, nil).APIClient()

	if !errors.Is(err, accounts.ErrNotLoggedIn) {
		t.Fatalf("want ErrNotLoggedIn, got %v", err)
	}
}

func TestAPIClient_usesResolvedAccount(t *testing.T) {
	f := factory(t, nil)
	cfg, _ := f.Config()
	cfg.DefaultAccount = "personal"
	cfg.Accounts = map[string]*config.Account{"personal": {}, "client-a": {}}
	_ = f.Keyring.Set("personal", "p-token")
	_ = f.Keyring.Set("client-a", "c-token")

	tests := []struct{ account, want string }{{"", "p-token"}, {"client-a", "c-token"}}
	for _, tt := range tests {
		f.Account = tt.account
		client, err := f.APIClient()
		if err != nil {
			t.Fatal(err)
		}
		if client.Token != tt.want {
			t.Errorf("--account=%q: token %q, want %q", tt.account, client.Token, tt.want)
		}
	}
}

func TestAPIClientWithToken_baseURL(t *testing.T) {
	tests := []struct{ name, baseURL, want string }{
		{"production by default", "", api.DefaultBaseURL},
		{"override", overrideURL, overrideURL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := factory(t, nil)
			f.BaseURL = tt.baseURL

			client := f.APIClientWithToken("abc")

			if client.Token != "abc" || client.BaseURL != tt.want {
				t.Errorf("token=%q url=%q", client.Token, client.BaseURL)
			}
		})
	}
}

func TestNew_readsAPIURLFromEnv(t *testing.T) {
	t.Setenv("POEDITOR_API_URL", overrideURL)

	if got := cmdutil.New().BaseURL; got != overrideURL {
		t.Errorf("BaseURL = %q", got)
	}
}

func TestConfig_loadedOnce(t *testing.T) {
	f := factory(t, nil)
	a, _ := f.Config()
	b, _ := f.Config()
	if a != b {
		t.Error("Config() should return the same instance")
	}
}

func TestResolveFor(t *testing.T) {
	f := factory(t, nil)
	cfg, _ := f.Config()
	cfg.DefaultAccount = "personal"
	cfg.Accounts = map[string]*config.Account{"personal": {}, "client-a": {}, "client-b": {}}
	for _, name := range []string{"personal", "client-a", "client-b"} {
		_ = f.Keyring.Set(name, name+"-tok")
	}

	tests := []struct{ flag, project, want string }{
		{"", "", "personal"},
		{"", "client-a", "client-a"},
		{"client-b", "client-a", "client-b"},
	}
	for _, tt := range tests {
		f.Account = tt.flag
		got, err := f.ResolveFor(tt.project)
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != tt.want {
			t.Errorf("--account=%q project=%q: got %q, want %q", tt.flag, tt.project, got.Name, tt.want)
		}
	}
}

func TestWorkingDir(t *testing.T) {
	f := factory(t, nil)
	f.WorkDir = "/some/repo"
	if got, _ := f.WorkingDir(); got != "/some/repo" {
		t.Errorf("WorkingDir = %q", got)
	}
}

func TestPrompt_notInteractive(t *testing.T) {
	if _, err := factory(t, nil).Prompt(); !errors.Is(err, cmdutil.ErrNotInteractive) {
		t.Fatalf("got %v", err)
	}
}

func TestSleep(t *testing.T) {
	if err := cmdutil.Sleep(context.Background(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if err := cmdutil.Sleep(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Error("a cancelled sleep must return at once")
	}
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte("from file"), 0o644); err != nil {
		t.Fatal(err)
	}
	ios, stdin, _, _ := iostreams.Test()
	stdin.WriteString("from stdin")
	f := &cmdutil.Factory{IO: ios, WorkDir: dir}

	for path, want := range map[string]string{
		"a.json":                     "from file", // relative to the working directory
		filepath.Join(dir, "a.json"): "from file",
		"-":                          "from stdin",
	} {
		got, err := f.ReadFile(path)
		if err != nil || string(got) != want {
			t.Errorf("ReadFile(%q) = %q, %v; want %q", path, got, err, want)
		}
	}
	if _, err := f.ReadFile("missing.json"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing file: err = %v", err)
	}
}

func TestResolvePath(t *testing.T) {
	dir := t.TempDir()
	f := &cmdutil.Factory{WorkDir: dir}
	abs := filepath.Join(dir, "x", "y.po")

	for in, want := range map[string]string{"locales/fr.po": filepath.Join(dir, "locales", "fr.po"), abs: abs} {
		if got, err := f.ResolvePath(in); err != nil || got != want {
			t.Errorf("ResolvePath(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}
