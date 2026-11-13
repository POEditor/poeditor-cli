// Package root builds the `poeditor` command tree.
package root

import (
	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/build"
	"github.com/POEditor/poeditor-cli/internal/cmd/projects"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// NewCmdRoot returns the root command with every subcommand attached.
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

	cmd.AddCommand(projects.NewCmdProjects(f))
	return cmd
}
