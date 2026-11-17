// Package languages holds `poeditor api languages ...`.
package languages

import (
	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// NewCmdLanguages returns the `languages` command group.
func NewCmdLanguages(f *cmdutil.Factory) *cobra.Command {
	cmd := cmdutil.NewGroup("languages <command>", "Work with a project's languages")
	cmd.AddCommand(NewCmdAdd(f), NewCmdAvailable(f), NewCmdDelete(f), NewCmdList(f), NewCmdUpdate(f))
	return cmd
}
