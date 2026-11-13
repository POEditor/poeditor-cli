package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/config"
)

func TestLoad_missingFileIsEmpty(t *testing.T) {
	cfg, err := config.Load(t.TempDir())

	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultAccount != "" || len(cfg.Accounts) != 0 {
		t.Errorf("want empty config, got %+v", cfg)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "poeditor")
	cfg, _ := config.Load(dir)
	cfg.DefaultAccount = "personal"
	cfg.Accounts = map[string]*config.Account{"personal": {}, "client-a": {Token: "fallback"}}

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(dir)

	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultAccount != "personal" || loaded.Accounts["client-a"].Token != "fallback" {
		t.Errorf("round trip lost data: %+v", loaded)
	}
	if got := loaded.AccountNames(); !reflect.DeepEqual(got, []string{"client-a", "personal"}) {
		t.Errorf("AccountNames = %v", got)
	}
	if !loaded.HasAccount("personal") || loaded.HasAccount("nope") {
		t.Error("HasAccount wrong")
	}
	raw, err := os.ReadFile(loaded.Path())
	if err != nil {
		t.Fatal(err)
	}
	if want := "accounts:\n  client-a:\n    token: fallback\n  personal: {}\n"; !strings.Contains(string(raw), want) {
		t.Errorf("file not 2-space indented:\n%s", raw)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(loaded.Path())
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("config file mode = %o, want 600", perm)
		}
	}
}

func TestLoad_invalidYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte("accounts: [unclosed"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(dir)

	if err == nil || !strings.Contains(err.Error(), "is not valid YAML") {
		t.Fatalf("got %v", err)
	}
}

func TestDir(t *testing.T) {
	env := map[string]string{"POEDITOR_CONFIG_DIR": "/custom"}
	if dir, _ := config.Dir(func(k string) string { return env[k] }); dir != "/custom" {
		t.Errorf("override ignored: %q", dir)
	}
	dir, err := config.Dir(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "poeditor" {
		t.Errorf("default dir = %q", dir)
	}
}
