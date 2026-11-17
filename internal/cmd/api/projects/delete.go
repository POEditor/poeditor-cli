package projects

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdDelete returns `poeditor api projects delete` (API v2
// projects/delete).
func NewCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var (
		project string
		yes     bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a project, with all its terms and translations",
		Long: `Delete a project the account owns: its terms, translations, languages and
contributors go with it, and this cannot be undone. Without --yes nothing is
deleted, and the command prints the line to run.`,
		Example: `  poeditor api projects delete --project 7717 --yes`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if project == "" {
				return projectflag.ErrMissing
			}
			if !yes {
				args := []string{"--project", project}
				if f.Account != "" {
					args = append(args, "--account", f.Account)
				}
				return fmt.Errorf("deleting project %s deletes all its terms and translations and cannot be undone; nothing was deleted. To delete it, run:\n  %s %s",
					project, cmd.CommandPath(), cmdutil.ShellJoin(append(args, "--yes")))
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			if err := p.Client.ProjectsDelete(cmd.Context(), p.ID); err != nil {
				return err
			}
			fmt.Fprintf(f.IO.ErrOut, "Deleted project %s.\n", p.Label)
			return nil
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().BoolVar(&yes, "yes", false, "Delete it: everything in it goes, and this cannot be undone")
	cmdutil.MarkAction(cmd)
	return cmd
}
