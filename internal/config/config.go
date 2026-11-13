// Package config is the per-user config file, <config dir>/config.yml:
//
//	default_account: personal
//	accounts:
//	  personal: {}
//	  client-a:
//	    token: ...   # only when the OS keyring is unavailable
//
// The config dir is $POEDITOR_CONFIG_DIR, else <os.UserConfigDir()>/poeditor
// (~/.config/poeditor, ~/Library/Application Support/poeditor, %AppData%\poeditor).
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"
)

// FileName is the config file inside the config dir.
const FileName = "config.yml"

// Config is the parsed config file.
type Config struct {
	DefaultAccount string              `yaml:"default_account,omitempty"`
	Accounts       map[string]*Account `yaml:"accounts,omitempty"`

	path string
}

// Account is one named account. Token is set only when the keyring could not
// be used; normally the token lives in the keyring.
type Account struct {
	Token string `yaml:"token,omitempty"`
}

// Dir returns the config directory, honouring POEDITOR_CONFIG_DIR.
func Dir(getenv func(string) string) (string, error) {
	if dir := getenv("POEDITOR_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot find the user config directory: %w", err)
	}
	return filepath.Join(base, "poeditor"), nil
}

// Load reads <dir>/config.yml; a missing file is an empty config.
func Load(dir string) (*Config, error) {
	cfg := &Config{path: filepath.Join(dir, FileName)}
	data, err := os.ReadFile(cfg.path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("%s is not valid YAML: %w", cfg.path, err)
	}
	return cfg, nil
}

// Path is the file the config is read from and saved to.
func (c *Config) Path() string { return c.path }

// Save writes the file (0600: it may hold a fallback token), creating the dir.
func (c *Config) Save() error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return err
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

// AccountNames returns the account names, sorted.
func (c *Config) AccountNames() []string {
	names := make([]string, 0, len(c.Accounts))
	for name := range c.Accounts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// HasAccount reports whether name is configured.
func (c *Config) HasAccount(name string) bool {
	_, ok := c.Accounts[name]
	return ok
}
