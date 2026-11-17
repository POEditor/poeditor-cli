// Package terms holds `poeditor api terms ...`.
package terms

import (
	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// NewCmdTerms returns the `terms` command group.
func NewCmdTerms(f *cmdutil.Factory) *cobra.Command {
	cmd := cmdutil.NewGroup("terms <command>", "Work with a project's terms")
	cmd.AddCommand(NewCmdAdd(f), NewCmdAddComment(f), NewCmdDelete(f), NewCmdList(f), NewCmdUpdate(f))
	return cmd
}
