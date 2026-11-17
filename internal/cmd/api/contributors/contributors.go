// Package contributors holds `poeditor api contributors ...`.
package contributors

import (
	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// NewCmdContributors returns the `contributors` command group.
func NewCmdContributors(f *cmdutil.Factory) *cobra.Command {
	cmd := cmdutil.NewGroup("contributors <command>", "Work with a project's contributors")
	cmd.AddCommand(NewCmdAdd(f), NewCmdList(f), NewCmdRemove(f))
	return cmd
}
