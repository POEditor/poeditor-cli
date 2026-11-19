package projects

import (
	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdView returns `poeditor api projects view` (API v2 projects/view).
func NewCmdView(f *cmdutil.Factory) *cobra.Command {
	var (
		project string
		asJSON  bool
	)

	cmd := &cobra.Command{
		Use:   "view",
		Short: "Show a project's details",
		Long: `Show a project: name, description, reference and fallback languages, public
and open flags, how many terms it has, when terms were last added and updated
(T ADDED, T UPDATED) and when the project was created. One row, as in
` + "`poeditor api projects list`" + `; piped, one tab-separated line.`,
		Example: `  poeditor api projects view --project web
  poeditor api projects view -p 7717 --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			details, err := p.Client.ProjectsView(cmd.Context(), p.ID)
			if err != nil {
				return err
			}
			if asJSON {
				return cmdutil.PrintJSON(f.IO, toDetailsJSON(details))
			}
			return printDetails(f, details)
		},
	}

	projectflag.Add(cmd, &project)
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}
