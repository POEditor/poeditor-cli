package auth

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/keyring"
)

// NewCmdLogout returns `poeditor auth logout`.
func NewCmdLogout(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove an account and its stored token",
		Long: `Remove an account and its stored token.

With a single account, no --account is needed. With several, name the one to
remove with --account. If it was the default, the first remaining account (in
alphabetical order) becomes the default.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}

			name := f.Account
			if name == "" {
				switch len(cfg.Accounts) {
				case 0:
					return errors.New("not logged in to any account")
				case 1:
					name = cfg.AccountNames()[0]
				default:
					return fmt.Errorf("you have several accounts (%s): say which one with --account <name>", strings.Join(cfg.AccountNames(), ", "))
				}
			}
			if !cfg.HasAccount(name) {
				return fmt.Errorf("account %q isn't set up", name)
			}

			keyringErr := f.Keyring.Delete(name)
			// not in the keyring, or kept in the config file because there is no
			// keyring: removing the account entry below removes the token
			if errors.Is(keyringErr, keyring.ErrNotFound) || cfg.Accounts[name].Token != "" {
				keyringErr = nil
			}

			delete(cfg.Accounts, name)
			newDefault := ""
			if cfg.DefaultAccount == name {
				if remaining := cfg.AccountNames(); len(remaining) > 0 {
					newDefault = remaining[0]
				}
				cfg.DefaultAccount = newDefault
			}
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("cannot save %s: %w", cfg.Path(), err)
			}

			fmt.Fprintf(f.IO.ErrOut, "Logged out of %q.\n", name)
			if newDefault != "" {
				fmt.Fprintf(f.IO.ErrOut, "%q is now your default account.\n", newDefault)
			}
			if keyringErr != nil {
				fmt.Fprintf(f.IO.ErrOut, "Warning: could not remove the token from the system keyring (%v); delete the %q entry for service %q by hand.\n",
					keyringErr, name, keyring.Service)
			}
			return nil
		},
	}
}
