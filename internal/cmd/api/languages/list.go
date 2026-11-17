package languages

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdList returns `poeditor api languages list` (API v2 languages/list).
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	var (
		project string
		asJSON  bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List a project's languages and how far each is translated",
		Long: `List the languages of a project: code, name, how many terms are translated,
the progress, and when a translation was last added.

Piped, each language is one tab-separated line (the progress as a bare
number); --json gives everything.`,
		Example: `  poeditor api languages list --project web
  poeditor api languages list -p 7717 --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			langs, err := p.Client.LanguagesList(cmd.Context(), p.ID)
			if err != nil {
				return err
			}
			if asJSON {
				return cmdutil.PrintJSON(f.IO, languagesJSON(langs))
			}
			return printLanguages(f, p, langs)
		},
	}

	projectflag.Add(cmd, &project)
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

func printLanguages(f *cmdutil.Factory, p projectflag.Project, langs []api.Language) error {
	if len(langs) == 0 {
		if f.IO.IsStdoutTTY() {
			fmt.Fprintf(f.IO.Out, "Project %s has no languages.\n", p.Label)
		}
		return nil
	}
	table := cmdutil.NewTable(f.IO, "CODE", "NAME", "TRANSLATIONS", "PROGRESS", "UPDATED")
	for _, l := range langs {
		table.AddRow(l.Code, l.Name, strconv.Itoa(l.Translations),
			cmdutil.PercentCell(f.IO, l.Percentage), cmdutil.TimeCell(f.IO, l.Updated))
	}
	return table.Render()
}

// languageJSON is one language in --json: the API's fields, as our own type
// so an API change never changes our output by accident.
type languageJSON struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Translations int      `json:"translations"`
	Percentage   float64  `json:"percentage"`
	Updated      api.Time `json:"updated"`
}

func languagesJSON(langs []api.Language) []languageJSON {
	out := make([]languageJSON, len(langs))
	for i, l := range langs {
		out[i] = languageJSON{
			Code: l.Code, Name: l.Name, Translations: l.Translations,
			Percentage: float64(l.Percentage), Updated: l.Updated,
		}
	}
	return out
}
