package languages

import (
	"errors"
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdAdd returns `poeditor api languages add` (API v2 languages/add, once
// per language).
func NewCmdAdd(f *cmdutil.Factory) *cobra.Command {
	var project string

	cmd := &cobra.Command{
		Use:   "add <code>...",
		Short: "Add languages to a project",
		Long: `Add one or more languages to a project, by code (every code:
` + "`poeditor api languages available`" + `). A language the project already has is
left as it is, with a note. The first code POEditor refuses stops the command;
the languages before it stay added.`,
		Example: `  poeditor api languages add fr --project web
  poeditor api languages add de es pt-br -p 7717`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, codes []string) error {
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			for _, code := range unique(codes) {
				err := p.Client.LanguagesAdd(cmd.Context(), p.ID, code)
				var apiErr *api.Error
				switch {
				case errors.As(err, &apiErr) && apiErr.Code == api.CodeLanguageInProject:
					fmt.Fprintf(f.IO.ErrOut, "Project %s already has %s.\n", p.Label, code)
				case err != nil:
					return p.LanguageError(cmd.Context(), code, err)
				default:
					fmt.Fprintf(f.IO.ErrOut, "Added %s to project %s.\n", code, p.Label)
				}
			}
			return nil
		},
	}

	projectflag.Add(cmd, &project)
	cmdutil.MarkAction(cmd)
	return cmd
}

// unique keeps the first of each code, in order: "add fr fr" adds fr once.
func unique(codes []string) []string {
	var out []string
	for _, c := range codes {
		if !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	return out
}
