// Package accounts decides which account and token a command uses.
//
// Account: --account if given, else the config's default_account.
// Token for that account, first found wins:
//  1. POEDITOR_TOKEN_<ACCOUNT> (client-a → POEDITOR_TOKEN_CLIENT_A)
//  2. POEDITOR_TOKEN, for the default account only
//  3. the OS keyring
//  4. the config file (fallback when the keyring is unavailable)
//
// With no account configured at all, POEDITOR_TOKEN alone is enough (CI).
package accounts

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/POEditor/poeditor-cli/internal/config"
	"github.com/POEditor/poeditor-cli/internal/keyring"
)

// ErrNotLoggedIn means there is no account and no POEDITOR_TOKEN.
var ErrNotLoggedIn = errors.New("not logged in: run `poeditor auth login`, or set POEDITOR_TOKEN")

// Token sources, as shown by `auth list`.
const (
	SourceEnv     = "environment"
	SourceKeyring = "keyring"
	SourceConfig  = "config file"
)

var validName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,39}$`)

// ValidateName checks an account name: lower-case letters, digits, - and _,
// so it maps to one unambiguous environment variable.
func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid account name %q: use lower-case letters, digits, - and _ (max 40)", name)
	}
	return nil
}

// EnvVar is the environment variable holding name's token.
func EnvVar(name string) string {
	return "POEDITOR_TOKEN_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}

// Resolver resolves accounts against one config, keyring and environment.
type Resolver struct {
	Config  *config.Config
	Keyring keyring.Keyring
	Getenv  func(string) string
}

// Resolved is the account a command runs as.
type Resolved struct {
	Name   string // "" when running on a bare POEDITOR_TOKEN
	Token  string
	Source string // SourceEnv, SourceKeyring or SourceConfig
}

// Resolve picks the account (requested, or the default) and its token.
func (r *Resolver) Resolve(requested string) (Resolved, error) {
	name := requested
	if name == "" {
		name = r.Config.DefaultAccount
	}
	if name == "" {
		if token := r.Getenv("POEDITOR_TOKEN"); token != "" {
			return Resolved{Token: token, Source: SourceEnv}, nil
		}
		return Resolved{}, ErrNotLoggedIn
	}

	token, source := r.Token(name)
	if token != "" {
		return Resolved{Name: name, Token: token, Source: source}, nil
	}
	if !r.Config.HasAccount(name) {
		return Resolved{}, fmt.Errorf("account %q isn't set up. Run `poeditor auth login --account %s`", name, name)
	}
	return Resolved{}, fmt.Errorf("account %q has no token stored. Run `poeditor auth login --account %s`", name, name)
}

// Token returns name's token and where it came from, or "" if there is none.
func (r *Resolver) Token(name string) (token, source string) {
	if token := r.Getenv(EnvVar(name)); token != "" {
		return token, SourceEnv
	}
	if name == r.Config.DefaultAccount {
		if token := r.Getenv("POEDITOR_TOKEN"); token != "" {
			return token, SourceEnv
		}
	}
	if token, err := r.Keyring.Get(name); err == nil && token != "" {
		return token, SourceKeyring
	}
	if acc := r.Config.Accounts[name]; acc != nil && acc.Token != "" {
		return acc.Token, SourceConfig
	}
	return "", ""
}
