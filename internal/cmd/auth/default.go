package auth

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// NewCmdDefault returns `poeditor auth default <account>`.
func NewCmdDefault(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "default <account>",
		Short: "Set the account used when no --account is given",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}
			name := args[0]
			if !cfg.HasAccount(name) {
				if len(cfg.Accounts) == 0 {
					return fmt.Errorf("account %q isn't set up, and there are no accounts yet. Run `poeditor auth login --account %s`", name, name)
				}
				return fmt.Errorf("account %q isn't set up (accounts: %s)", name, strings.Join(cfg.AccountNames(), ", "))
			}
			cfg.DefaultAccount = name
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("cannot save %s: %w", cfg.Path(), err)
			}
			fmt.Fprintf(f.IO.ErrOut, "Default account: %q.\n", name)
			return nil
		},
	}
}
