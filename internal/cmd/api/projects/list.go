package projects

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// Project is one entry of the projects/list result.
type Project struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Public  int    `json:"public"`
	Open    int    `json:"open"`
	Created string `json:"created"`
}

// NewCmdList returns `poeditor projects list` (API v2 projects/list).
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the projects your token can access",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			account, err := f.Resolve()
			if err != nil {
				return err
			}
			client := f.APIClientWithToken(account.Token)

			var result struct {
				Projects []Project `json:"projects"`
			}
			if err := client.Post(cmd.Context(), "projects/list", nil, &result); err != nil {
				return err
			}

			if err := printProjects(f, result.Projects, asJSON); err != nil {
				return err
			}
			return accountHint(f, account.Name)
		},
	}

	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

func printProjects(f *cmdutil.Factory, projects []Project, asJSON bool) error {
	if asJSON {
		if projects == nil {
			projects = []Project{}
		}
		return cmdutil.PrintJSON(f.IO, projects)
	}

	if len(projects) == 0 {
		if f.IO.IsStdoutTTY() {
			fmt.Fprintln(f.IO.Out, "No projects found.")
		}
		return nil
	}
	table := cmdutil.NewTable(f.IO, "ID", "NAME", "PUBLIC", "OPEN", "CREATED")
	for _, p := range projects {
		table.AddRow(strconv.Itoa(p.ID), p.Name, yesNo(p.Public), yesNo(p.Open), p.Created)
	}
	return table.Render()
}

// accountHint tells users with several accounts whose projects they see. It
// goes to stderr so --json output and pipes stay clean.
func accountHint(f *cmdutil.Factory, current string) error {
	cfg, err := f.Config()
	if err != nil || current == "" || len(cfg.Accounts) < 2 {
		return err
	}
	var others []string
	for _, name := range cfg.AccountNames() {
		if name != current {
			others = append(others, name)
		}
	}
	use := "--account <name>"
	if len(others) == 1 {
		use = "--account " + others[0]
	}
	fmt.Fprintf(f.IO.ErrOut, "\nAccount: %s (also logged in: %s — use %s)\n", current, strings.Join(others, ", "), use)
	return nil
}

func yesNo(v int) string {
	if v != 0 {
		return "yes"
	}
	return "no"
}
