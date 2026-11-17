package contributors

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdRemove returns `poeditor api contributors remove` (API v2
// contributors/remove).
func NewCmdRemove(f *cmdutil.Factory) *cobra.Command {
	var (
		project string
		r       api.ContributorRemoval
		yes     bool
	)

	cmd := &cobra.Command{
		Use:   "remove <email>",
		Short: "Remove a contributor from a project, a language, or the proofreader right",
		Long: `Take someone's access to a project away: all of it (an administrator only
the project owner can remove), one language with --language, or only the
proofreader right with --proofreader.

Without --yes nothing is removed, and the command prints the line to run.`,
		Example: `  poeditor api contributors remove ana@example.com --project web --yes
  poeditor api contributors remove ana@example.com -p 7717 --language fr --yes
  poeditor api contributors remove ana@example.com -p web --proofreader --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]
			if project == "" {
				return projectflag.ErrMissing
			}
			if !strings.Contains(email, "@") {
				return fmt.Errorf("%q is not an email address", email)
			}
			if !yes {
				return fmt.Errorf("this removes %s; nothing was removed. To remove it, run:\n  %s %s",
					removal(email, project, r), cmd.CommandPath(), cmdutil.ShellJoin(confirmArgs(f, email, project, r)))
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			if err := p.Client.ContributorsRemove(cmd.Context(), p.ID, email, r); err != nil {
				return err
			}
			fmt.Fprintf(f.IO.ErrOut, "Removed %s.\n", removal(email, p.Label, r))
			return nil
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().StringVarP(&r.Language, "language", "l", "", "Only from this `language`")
	cmd.Flags().BoolVar(&r.Proofreader, "proofreader", false, "Only the proofreader right")
	cmd.Flags().BoolVar(&yes, "yes", false, "Remove it")
	cmd.MarkFlagsMutuallyExclusive("language", "proofreader")
	cmdutil.MarkAction(cmd)
	return cmd
}

// removal says what goes: "ana@example.com from project web", "… from fr in
// project web", "the proofreader right of … in project web".
func removal(email, project string, r api.ContributorRemoval) string {
	switch {
	case r.Proofreader:
		return fmt.Sprintf("the proofreader right of %s in project %s", email, project)
	case r.Language != "":
		return fmt.Sprintf("%s from %s in project %s", email, r.Language, project)
	}
	return fmt.Sprintf("%s from project %s", email, project)
}

// confirmArgs is the command line that removes for real.
func confirmArgs(f *cmdutil.Factory, email, project string, r api.ContributorRemoval) []string {
	out := []string{email, "--project", project}
	switch {
	case r.Proofreader:
		out = append(out, "--proofreader")
	case r.Language != "":
		out = append(out, "--language", r.Language)
	}
	if f.Account != "" {
		out = append(out, "--account", f.Account)
	}
	return append(out, "--yes")
}
