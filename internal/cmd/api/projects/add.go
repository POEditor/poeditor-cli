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
		description string
		asJSON      bool
	)

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a project",
		Long: `Create a project owned by the account (--account, else the default one). It
starts with no languages: add them with ` + "`poeditor api languages add`" + `.

The new project's id goes to stderr; --json prints the whole project.`,
		Example: `  poeditor api projects add "Mobile app"
  poeditor api projects add Web --description "The marketing site" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
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

	cmd.Flags().StringVar(&description, "description", "", "The project's `description`")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}
