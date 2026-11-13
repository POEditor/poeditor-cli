// Package cmdutil holds what every command needs: the streams, the config,
// the account resolution and a way to get an API client. Commands receive a
// *Factory and never read globals, so tests can hand them buffers, a temp
// config dir, an in-memory keyring and a fake API.
package cmdutil

import (
	"os"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/build"
	"github.com/POEditor/poeditor-cli/internal/config"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
	"github.com/POEditor/poeditor-cli/internal/keyring"
)

// Factory is passed to every command constructor.
type Factory struct {
	IO *iostreams.IOStreams

	// Account is the --account flag (bound by the root command); "" = default.
	Account string
	// BaseURL is the API v2 root; "" = production. Real binary: POEDITOR_API_URL.
	BaseURL string
	// ConfigDir overrides the config directory; "" = config.Dir().
	ConfigDir string

	Getenv  func(string) string
	Keyring keyring.Keyring

	cfg *config.Config
}

// New returns the factory used by the real binary.
func New() *Factory {
	return &Factory{
		IO:      iostreams.System(os.Getenv),
		BaseURL: os.Getenv("POEDITOR_API_URL"),
		Getenv:  os.Getenv,
		Keyring: keyring.System(),
	}
}

// Config loads the user config once per run.
func (f *Factory) Config() (*config.Config, error) {
	if f.cfg != nil {
		return f.cfg, nil
	}
	dir := f.ConfigDir
	if dir == "" {
		var err error
		if dir, err = config.Dir(f.Getenv); err != nil {
			return nil, err
		}
	}
	cfg, err := config.Load(dir)
	if err != nil {
		return nil, err
	}
	f.cfg = cfg
	return cfg, nil
}

// Resolver returns the account resolver over the config, keyring and env.
func (f *Factory) Resolver() (*accounts.Resolver, error) {
	cfg, err := f.Config()
	if err != nil {
		return nil, err
	}
	return &accounts.Resolver{Config: cfg, Keyring: f.Keyring, Getenv: f.Getenv}, nil
}

// Resolve returns the account this run uses (--account, else the default).
func (f *Factory) Resolve() (accounts.Resolved, error) {
	r, err := f.Resolver()
	if err != nil {
		return accounts.Resolved{}, err
	}
	return r.Resolve(f.Account)
}

// APIClient returns a client authenticated as the resolved account.
func (f *Factory) APIClient() (*api.Client, error) {
	res, err := f.Resolve()
	if err != nil {
		return nil, err
	}
	return f.APIClientWithToken(res.Token), nil
}

// APIClientWithToken returns a client for an explicit token (auth login).
func (f *Factory) APIClientWithToken(token string) *api.Client {
	return api.NewClient(f.BaseURL, token, "poeditor-cli/"+build.Version)
}
