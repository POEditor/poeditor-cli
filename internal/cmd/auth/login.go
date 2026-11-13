package auth

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/config"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

// TokenPageURL is where users create API tokens.
const TokenPageURL = "https://poeditor.com/account/api"

// NewCmdLogin returns `poeditor auth login`.
func NewCmdLogin(f *cmdutil.Factory) *cobra.Command {
	var withToken bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store an API token for an account",
		Long: `Store an API token for an account.

The token is checked against the API before anything is saved, then kept in the
system keyring. Your first account is called "default" unless you name it with
--account; once you have one, name every further account with --account.
Logging in again to an existing account replaces its token.

Tokens are created at ` + TokenPageURL + `.`,
		Example: `  poeditor auth login
  poeditor auth login --account client-a
  echo "$TOKEN" | poeditor auth login --account ci --with-token`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}

			name := f.Account
			if name == "" {
				if len(cfg.Accounts) > 0 {
					return fmt.Errorf("you already have an account (%s): use --account <name> to add another one, or --account %s to replace its token",
						strings.Join(cfg.AccountNames(), ", "), cfg.AccountNames()[0])
				}
				name = "default"
			}
			if err := accounts.ValidateName(name); err != nil {
				return err
			}

			token, err := readToken(f.IO, withToken)
			if err != nil {
				return err
			}

			var result struct {
				Projects []struct{} `json:"projects"`
			}
			if err := f.APIClientWithToken(token).Post(cmd.Context(), "projects/list", nil, &result); err != nil {
				return fmt.Errorf("token not accepted, nothing was saved: %w", err)
			}

			existed := cfg.HasAccount(name)
			account := &config.Account{}
			keyringErr := f.Keyring.Set(name, token)
			if keyringErr != nil {
				account.Token = token
			}
			if cfg.Accounts == nil {
				cfg.Accounts = map[string]*config.Account{}
			}
			cfg.Accounts[name] = account
			madeDefault := cfg.DefaultAccount == ""
			if madeDefault {
				cfg.DefaultAccount = name
			}
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("cannot save %s: %w", cfg.Path(), err)
			}

			verb := "Logged in to"
			if existed {
				verb = "Updated the token of"
			}
			fmt.Fprintf(f.IO.ErrOut, "%s account %q (%s).\n", verb, name, projectCount(len(result.Projects)))
			if keyringErr != nil {
				fmt.Fprintf(f.IO.ErrOut, "Warning: no system keyring available (%v); the token is stored in plain text in %s.\n", keyringErr, cfg.Path())
			} else {
				fmt.Fprintln(f.IO.ErrOut, "Token stored in the system keyring.")
			}
			if madeDefault {
				fmt.Fprintf(f.IO.ErrOut, "%q is your default account.\n", name)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&withToken, "with-token", false, "Read the token from standard input (no prompt)")
	cmdutil.MarkAction(cmd)
	return cmd
}

func readToken(ios *iostreams.IOStreams, fromStdin bool) (string, error) {
	var token string
	if fromStdin {
		data, err := io.ReadAll(ios.In)
		if err != nil {
			return "", err
		}
		token = string(data)
	} else {
		fmt.Fprintf(ios.ErrOut, "Paste your API token (create one at %s): ", TokenPageURL)
		if file, ok := ios.In.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
			data, err := term.ReadPassword(int(file.Fd()))
			fmt.Fprintln(ios.ErrOut)
			if err != nil {
				return "", err
			}
			token = string(data)
		} else {
			line, err := bufio.NewReader(ios.In).ReadString('\n')
			if err != nil && !errors.Is(err, io.EOF) {
				return "", err
			}
			token = line
		}
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("no token given")
	}
	return token, nil
}

func projectCount(n int) string {
	if n == 1 {
		return "1 project"
	}
	return fmt.Sprintf("%d projects", n)
}
