package languages

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdDelete returns `poeditor api languages delete` (API v2
// languages/delete).
func NewCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var (
		project string
		yes     bool
	)

	cmd := &cobra.Command{
		Use:   "delete <code>",
		Short: "Delete a language from a project, with all its translations",
		Long: `Delete a language from a project. All its translations are deleted with it,
and this cannot be undone: without --yes nothing is deleted, and the command
prints the line to run.`,
		Example: `  poeditor api languages delete fr --project web --yes`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]
			if project == "" {
				return projectflag.ErrMissing
			}
			if !yes {
				return fmt.Errorf("deleting %s deletes all its translations and cannot be undone; nothing was deleted. To delete it, run:\n  %s %s",
					code, cmd.CommandPath(), cmdutil.ShellJoin(confirmArgs(f, code, project)))
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			if err := p.Client.LanguagesDelete(cmd.Context(), p.ID, code); err != nil {
				return p.LanguageError(cmd.Context(), code, err)
			}
			fmt.Fprintf(f.IO.ErrOut, "Deleted %s from project %s.\n", code, p.Label)
			return nil
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().BoolVar(&yes, "yes", false, "Delete it: the language's translations go with it, and this cannot be undone")
	cmdutil.MarkAction(cmd)
	return cmd
}

// confirmArgs is the command line that deletes for real: the user's
// arguments plus --yes.
func confirmArgs(f *cmdutil.Factory, code, project string) []string {
	args := []string{code, "--project", project}
	if f.Account != "" {
		args = append(args, "--account", f.Account)
	}
	return append(args, "--yes")
}
