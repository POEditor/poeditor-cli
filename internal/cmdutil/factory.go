// Package cmdutil holds what every command needs: the streams, the config,
// the account resolution and a way to get an API client. Commands receive a
// *Factory and never read globals, so tests can hand them buffers, a temp
// config dir, an in-memory keyring and a fake API.
package cmdutil

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/term"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/build"
	"github.com/POEditor/poeditor-cli/internal/config"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
	"github.com/POEditor/poeditor-cli/internal/keyring"
	"github.com/POEditor/poeditor-cli/internal/prompt"
)

// ErrNotInteractive is returned by Prompt when nobody can answer questions.
var ErrNotInteractive = errors.New("this command asks questions and needs an interactive terminal")

// Factory is passed to every command constructor.
type Factory struct {
	IO *iostreams.IOStreams

	// Account is the --account flag (bound by the root command); "" = default.
	Account string
	// BaseURL is the API v2 root; "" = production. Real binary: POEDITOR_API_URL.
	BaseURL string
	// ConfigDir overrides the config directory; "" = config.Dir().
	ConfigDir string
	// WorkDir is where .poeditor.yml is searched from; "" = the current directory.
	WorkDir string

	Getenv  func(string) string
	Keyring keyring.Keyring
	// Prompter asks questions; nil when stdin/stderr are not a terminal.
	Prompter prompt.Prompter
	// Sleep waits d or until ctx ends (tests record the waits instead).
	Sleep func(ctx context.Context, d time.Duration) error

	cfg *config.Config
}

// New returns the factory used by the real binary.
func New() *Factory {
	f := &Factory{
		IO:      iostreams.System(os.Getenv),
		BaseURL: os.Getenv("POEDITOR_API_URL"),
		Getenv:  os.Getenv,
		Keyring: keyring.System(),
		Sleep:   Sleep,
	}
	if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stderr.Fd())) {
		f.Prompter = prompt.Huh{In: os.Stdin, Out: os.Stderr}
	}
	return f
}

// Sleep waits d, or returns early with ctx's error (Ctrl+C).
func Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Prompt returns the prompter, or ErrNotInteractive.
func (f *Factory) Prompt() (prompt.Prompter, error) {
	if f.Prompter == nil {
		return nil, ErrNotInteractive
	}
	return f.Prompter, nil
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

// WorkingDir is the directory commands work in.
func (f *Factory) WorkingDir() (string, error) {
	if f.WorkDir != "" {
		return f.WorkDir, nil
	}
	return os.Getwd()
}

// ResolveFor returns the account for a project of .poeditor.yml: --account
// wins, then the project's `account:`, then the default account.
func (f *Factory) ResolveFor(projectAccount string) (accounts.Resolved, error) {
	r, err := f.Resolver()
	if err != nil {
		return accounts.Resolved{}, err
	}
	if f.Account != "" {
		return r.Resolve(f.Account)
	}
	return r.Resolve(projectAccount)
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

// ReadFile reads a file named on the command line: relative to the working
// directory, or stdin for "-".
func (f *Factory) ReadFile(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(f.IO.In)
	}
	path, err := f.ResolvePath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// ResolvePath is a path named on the command line, relative to the working
// directory.
func (f *Factory) ResolvePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	dir, err := f.WorkingDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, path), nil
}
