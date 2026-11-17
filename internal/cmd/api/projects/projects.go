// Package projects holds `poeditor api projects ...`.
package projects

import (
	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// NewCmdProjects returns the `projects` command group.
func NewCmdProjects(f *cmdutil.Factory) *cobra.Command {
	cmd := cmdutil.NewGroup("projects <command>", "Work with POEditor projects")
	cmd.AddCommand(NewCmdAdd(f), NewCmdDelete(f), NewCmdExport(f), NewCmdList(f), NewCmdSync(f), NewCmdUpdate(f), NewCmdUpload(f), NewCmdView(f))
	return cmd
}
