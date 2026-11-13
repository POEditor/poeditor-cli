// Package projects holds `poeditor projects ...`.
package projects

import (
	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// NewCmdProjects returns the `projects` command group.
func NewCmdProjects(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects <command>",
		Short: "Work with POEditor projects",
	}
	cmd.AddCommand(NewCmdList(f))
	return cmd
}
