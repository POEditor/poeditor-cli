package translations

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

const rowsHelp = `The file is a list in the API's shape; "context" may be left out (no
context), and a plural term's content is an object of its forms:

  [
    {"term": "save", "context": "button", "translation": {"content": "Enregistrer"}},
    {"term": "%d files", "translation": {"content": {"one": "1 fichier", "other": "%d fichiers"}}}
  ]

The counts go to stderr; --json also prints them on stdout.`

// NewCmdAdd returns `poeditor api translations add` (API v2
// translations/add).
func NewCmdAdd(f *cmdutil.Factory) *cobra.Command {
	var w writeFlags

	cmd := &cobra.Command{
		Use:   "add [<term> [<text>]]",
		Short: "Add translations where a term has none (never overwrites)",
		Long: `Add the translation of a term in one language, or many from a JSON file
(--from-file, - for stdin). Only empty translations are filled: a term that
already has one keeps it (to overwrite, use translations update).

` + rowsHelp,
		Example: `  poeditor api translations add save Enregistrer --project web --language fr --context button
  poeditor api translations add "%d files" -p 7717 -l fr --plural-form one="1 fichier" --plural-form other="%d fichiers"
  poeditor api translations add -p web -l fr --from-file fr.json`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			rows, err := w.rows(f, cmd, args)
			if err != nil {
				return err
			}
			p, err := projectflag.Resolve(f, w.project)
			if err != nil {
				return err
			}
			stats, err := p.Client.TranslationsAdd(cmd.Context(), p.ID, w.language, rows)
			if err != nil {
				return p.LanguageError(cmd.Context(), "--language "+w.language, err)
			}
			var c addCounts
			if t := stats.Translations; t != nil {
				c = addCounts{Parsed: t.Parsed, Added: t.Added}
			}
			note := ""
			if left := len(rows) - c.Added; left > 0 {
				note = fmt.Sprintf("%d not added: the term already has a %s translation (translations update overwrites it), or is not in the project.", left, w.language)
			}
			return report(f, p, w.language, w.asJSON, fmt.Sprintf("Added %d of %s to project %s.", c.Added, nTranslations(len(rows), w.language), p.Label), note, c)
		},
	}

	w.add(cmd, "Add")
	return cmd
}

// addCounts are the counts of translations/add in --json.
type addCounts struct {
	Parsed int `json:"parsed"`
	Added  int `json:"added"`
}
