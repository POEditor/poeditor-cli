// Package cmdutil holds what every command needs: the streams and a way to get
// an API client. Commands receive a *Factory and never read globals, so tests
// can hand them buffers and a fake API.
package cmdutil

import (
	"errors"
	"os"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/build"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

// ErrNoToken is returned when no API token is configured.
var ErrNoToken = errors.New("no API token: set the POEDITOR_TOKEN environment variable")

// Factory is passed to every command constructor.
type Factory struct {
	IO        *iostreams.IOStreams
	APIClient func() (*api.Client, error)
}

// New returns the factory used by the real binary.
// Token: POEDITOR_TOKEN (keyring + named accounts come later, see docs/design-decisions.md).
// Base URL: POEDITOR_API_URL, defaulting to production.
func New() *Factory {
	return &Factory{
		IO: iostreams.System(),
		APIClient: func() (*api.Client, error) {
			token := os.Getenv("POEDITOR_TOKEN")
			if token == "" {
				return nil, ErrNoToken
			}
			return api.NewClient(os.Getenv("POEDITOR_API_URL"), token, "poeditor-cli/"+build.Version), nil
		},
	}
}
