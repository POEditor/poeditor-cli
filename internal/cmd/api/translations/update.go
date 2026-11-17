package translations

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdUpdate returns `poeditor api translations update` (API v2
// translations/update).
func NewCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	var w writeFlags

	cmd := &cobra.Command{
		Use:   "update [<term> [<text>]]",
		Short: "Overwrite existing translations (never adds one)",
		Long: `Overwrite the translation of a term in one language, or many from a JSON file
(--from-file, - for stdin). Only existing translations change: a term with
none is skipped (to add one, use translations add; to write both at once,
languages update).

` + rowsHelp,
		Example: `  poeditor api translations update save Enregistrer --project web --language fr --context button
  poeditor api translations update -p web -l fr --from-file fixes.json`,
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
			stats, err := p.Client.TranslationsUpdate(cmd.Context(), p.ID, w.language, rows)
			if err != nil {
				return p.LanguageError(cmd.Context(), "--language "+w.language, err)
			}
			var c updateCounts
			if t := stats.Translations; t != nil {
				c = updateCounts{Parsed: t.Parsed, Updated: t.Updated}
			}
			note := ""
			if left := len(rows) - c.Updated; left > 0 {
				note = fmt.Sprintf("%d not updated: the term has no %s translation yet (translations add), or is not in the project.", left, w.language)
			}
			return report(f, p, w.language, w.asJSON, fmt.Sprintf("Updated %d of %s in project %s.", c.Updated, nTranslations(len(rows), w.language), p.Label), note, c)
		},
	}

	w.add(cmd, "Apply")
	return cmd
}

// updateCounts are the counts of translations/update in --json.
type updateCounts struct {
	Parsed  int `json:"parsed"`
	Updated int `json:"updated"`
}
