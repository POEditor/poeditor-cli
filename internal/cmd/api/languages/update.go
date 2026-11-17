package languages

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// NewCmdUpdate returns `poeditor api languages update` (API v2
// languages/update).
func NewCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	var (
		project      string
		fromFile     string
		fuzzyTrigger bool
		asJSON       bool
	)

	cmd := &cobra.Command{
		Use:   "update <code>",
		Short: "Write many translations of one language from a JSON file",
		Long: `Write the translations of one language, read from a JSON file (or stdin with
--from-file -). Existing translations are overwritten; terms that are not in
the project are skipped. The file is a list of terms and their translation,
in the API's shape:

  [
    {"term": "save", "context": "button",
     "translation": {"content": "Enregistrer", "fuzzy": 0}},
    {"term": "files", "context": "",
     "translation": {"content": {"one": "1 fichier", "other": "%d fichiers"}}}
  ]

"context" may be left out (no context), "fuzzy" too (0). A plural term's
content is an object of its forms. The counts go to stderr; --json also
prints them on stdout.

To translate files of .poeditor.yml, use ` + "`poeditor push`" + `.`,
		Example: `  poeditor api languages update fr --project web --from-file fr.json
  jq '…' export.json | poeditor api languages update fr -p 7717 --from-file -
  poeditor api languages update fr -p web --from-file fr.json --fuzzy-trigger`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]
			rows, err := readRows(f, fromFile)
			if err != nil {
				return err
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			stats, err := p.Client.LanguagesUpdate(cmd.Context(), p.ID, code, rows, api.LanguageUpdateOptions{FuzzyTrigger: fuzzyTrigger})
			if err != nil {
				return p.LanguageError(cmd.Context(), code, err)
			}
			result := updateResult(p, code, stats)
			c := result.Translations
			fmt.Fprintf(f.IO.ErrOut, "Updated %s in project %s: %d parsed, %d added, %d updated.\n",
				code, p.Label, c.Parsed, c.Added, c.Updated)
			if asJSON {
				return cmdutil.PrintJSON(f.IO, result)
			}
			return nil
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().StringVar(&fromFile, "from-file", "", "The translations: a JSON `file` (- for stdin) (required)")
	_ = cmd.MarkFlagRequired("from-file")
	cmd.Flags().BoolVar(&fuzzyTrigger, "fuzzy-trigger", false, "Mark the other languages' translations fuzzy where a translation is added or changed")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// UpdateResult is what --json prints: our own shape, like push's.
type UpdateResult struct {
	Project      repoconfig.ProjectRef `json:"project"`
	Language     string                `json:"language"`
	Translations TranslationCounts     `json:"translations"`
}

// TranslationCounts are the translation counts of the update.
type TranslationCounts struct {
	Parsed  int `json:"parsed"`
	Added   int `json:"added"`
	Updated int `json:"updated"`
}

func updateResult(p projectflag.Project, code string, s api.Stats) UpdateResult {
	r := UpdateResult{Project: p.Ref, Language: code}
	if t := s.Translations; t != nil {
		r.Translations = TranslationCounts{Parsed: t.Parsed, Added: t.Added, Updated: t.Updated}
	}
	return r
}

// readRows reads and checks --from-file before any API call. The server
// would skip a row without a term silently.
func readRows(f *cmdutil.Factory, path string) ([]api.TranslationRow, error) {
	return cmdutil.ReadJSONList(f, path, func(r api.TranslationRow) error { return r.TermKey.Check() })
}
