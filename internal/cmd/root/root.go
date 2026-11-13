// Package root builds the `poeditor` command tree from the modules.
package root

import (
	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/build"
	apicmd "github.com/POEditor/poeditor-cli/internal/cmd/api"
	"github.com/POEditor/poeditor-cli/internal/cmd/auth"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// topLevel is the module whose commands sit directly under `poeditor`.
var topLevel = apicmd.Module

// modules are attached as `poeditor <module.Name> <command>`.
var modules = []cmdutil.Module{
	auth.Module,
}

// NewCmdRoot returns the root command with every module attached.
// It panics on a command-name clash: that is a programming error, and the
// test suite builds this tree, so it can never reach a release.
func NewCmdRoot(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "poeditor <command> <subcommand> [flags]",
		Short:         "POEditor CLI",
		Long:          "Work with POEditor projects, terms and translations from the command line.",
		Version:       build.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetIn(f.IO.In)
	cmd.SetOut(f.IO.Out)
	cmd.SetErr(f.IO.ErrOut)
	cmd.CompletionOptions.HiddenDefaultCmd = true
	cmd.PersistentFlags().StringVar(&f.Account, "account", "", "Account to use (default: your default account)")

	must(cmdutil.AddUnique(cmd, topLevel.Commands(f)...))
	for _, m := range modules {
		group := &cobra.Command{Use: m.Name + " <command>", Short: m.Short}
		must(cmdutil.AddUnique(group, m.Commands(f)...))
		must(cmdutil.AddUnique(cmd, group))
	}
	return cmd
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
