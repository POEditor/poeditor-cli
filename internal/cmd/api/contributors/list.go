package contributors

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdList returns `poeditor api contributors list` (API v2
// contributors/list).
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	var (
		project  string
		language string
		asJSON   bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the contributors of a project, or of all your projects",
		Long: `List who has access to a project: administrators, and contributors with their
languages. Without --project, the contributors of every project of the
account; --language (with --project) keeps those of one language.

Each line is one person in one project. ROLE is administrator, proofreader
or contributor (proofreader only when the project has proofreading on).`,
		Example: `  poeditor api contributors list --project web
  poeditor api contributors list -p 7717 --language fr
  poeditor api contributors list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if language != "" && project == "" {
				return errors.New("--language needs --project: it lists the contributors of one language of a project")
			}
			var (
				client *api.Client
				p      projectflag.Project
				err    error
			)
			if project != "" {
				if p, err = projectflag.Resolve(f, project); err != nil {
					return err
				}
				client = p.Client
			} else if client, err = f.APIClient(); err != nil {
				return err
			}
			list, err := client.ContributorsList(cmd.Context(), p.ID, language)
			if err != nil {
				if project != "" && language != "" {
					return p.LanguageError(cmd.Context(), "--language "+language, err)
				}
				return err
			}
			if asJSON {
				return cmdutil.PrintJSON(f.IO, contributorsJSON(list))
			}
			return printContributors(f, p, list)
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().Lookup("project").Usage = "The project: its alias in .poeditor.yml, or its id (default: every project)"
	cmd.Flags().StringVarP(&language, "language", "l", "", "Only the contributors of this `language` (needs --project)")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

func printContributors(f *cmdutil.Factory, p projectflag.Project, list []api.Contributor) error {
	if len(list) == 0 {
		if f.IO.IsStdoutTTY() {
			if p.ID != 0 {
				fmt.Fprintf(f.IO.Out, "Project %s has no contributors.\n", p.Label)
			} else {
				fmt.Fprintln(f.IO.Out, "No contributors.")
			}
		}
		return nil
	}
	oneProject := p.ID != 0
	header := []string{"NAME", "EMAIL", "ROLE", "LANGUAGES"}
	if !oneProject {
		header = append([]string{"ID", "PROJECT"}, header...)
	}
	table := cmdutil.NewTable(f.IO, header...)
	for _, c := range list {
		for _, perm := range c.Permissions {
			row := []string{c.Name, c.Email, role(perm), cmdutil.ListCell(f.IO, perm.Languages)}
			if !oneProject {
				row = append([]string{strconv.Itoa(int(perm.Project.ID)), perm.Project.Name}, row...)
			}
			table.AddRow(row...)
		}
	}
	return table.Render()
}

// role is a permission in one word: administrator, proofreader or
// contributor.
func role(perm api.Permission) string {
	switch {
	case perm.Type == api.ContributorAdmin:
		return "administrator"
	case perm.Proofreader:
		return "proofreader"
	}
	return "contributor"
}

// contributorJSON is one contributor in --json, as our own type so an API
// change never changes our output by accident.
type contributorJSON struct {
	Name        string           `json:"name"`
	Email       string           `json:"email"`
	Permissions []permissionJSON `json:"permissions"`
}

type permissionJSON struct {
	Project     projectJSON `json:"project"`
	Type        string      `json:"type"`
	Proofreader bool        `json:"proofreader"`
	Languages   []string    `json:"languages"`
}

type projectJSON struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func contributorsJSON(list []api.Contributor) []contributorJSON {
	out := make([]contributorJSON, len(list))
	for i, c := range list {
		out[i] = contributorJSON{Name: c.Name, Email: c.Email, Permissions: []permissionJSON{}}
		for _, perm := range c.Permissions {
			langs := perm.Languages
			if langs == nil {
				langs = []string{}
			}
			out[i].Permissions = append(out[i].Permissions, permissionJSON{
				Project: projectJSON{ID: int(perm.Project.ID), Name: perm.Project.Name},
				Type:    perm.Type, Proofreader: perm.Proofreader, Languages: langs,
			})
		}
	}
	return out
}
