// Package auth is the auth module: `poeditor auth login|list|default|logout`.
// It manages named accounts in the user config and their tokens in the OS keyring.
package auth

import (
	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// Module is the auth module, attached as `poeditor auth`.
var Module = cmdutil.Module{
	Name:  "auth",
	Short: "Log in to POEditor accounts and manage their API tokens",
	Commands: func(f *cmdutil.Factory) []*cobra.Command {
		return []*cobra.Command{
			NewCmdLogin(f),
			NewCmdList(f),
			NewCmdDefault(f),
			NewCmdLogout(f),
		}
	},
}
