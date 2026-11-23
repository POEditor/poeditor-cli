package projects

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// NewCmdAdd returns `poeditor api projects add` (API v2 projects/add).
func NewCmdAdd(f *cmdutil.Factory) *cobra.Command {
	var (
		nameFlag    string
		description string
		asJSON      bool
	)

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a project",
		Long: `Create a project owned by the account (--account, else the default one). It
starts with no languages: add them with ` + "`poeditor api languages add`" + `.

The name is the argument, or --name (as in ` + "`projects update`" + `); not both.

The new project's id goes to stderr; --json prints the whole project.`,
		Example: `  poeditor api projects add "Mobile app"
  poeditor api projects add --name Web --description "The marketing site" --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameGiven := cmd.Flags().Changed("name")
			if len(args) == 1 && nameGiven {
				return errors.New("give the project name once: as an argument or with --name, not both")
			}
			if len(args) == 0 && !nameGiven {
				return errors.New("the project name is missing: poeditor api projects add <name> (or --name <name>)")
			}
			name := nameFlag
			if len(args) == 1 {
				name = args[0]
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return errors.New("the project name is empty")
			}
			client, err := f.APIClient()
			if err != nil {
				return err
			}
			p, err := client.ProjectsAdd(cmd.Context(), name, description)
			if err != nil {
				return err
			}
			fmt.Fprintf(f.IO.ErrOut, "Created project %s (id %d).\n", p.Name, p.ID)
			if asJSON {
				return cmdutil.PrintJSON(f.IO, toDetailsJSON(p))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&nameFlag, "name", "", "The project's `name` (instead of the argument)")
	cmd.Flags().StringVar(&description, "description", "", "The project's `description`")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}
