// Package translations holds `poeditor api translations ...`.
package translations

import (
	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// NewCmdTranslations returns the `translations` command group.
func NewCmdTranslations(f *cmdutil.Factory) *cobra.Command {
	cmd := cmdutil.NewGroup("translations <command>", "Work with a project's translations")
	cmd.AddCommand(NewCmdAdd(f), NewCmdAutomatic(f), NewCmdDelete(f), NewCmdUpdate(f))
	return cmd
}
