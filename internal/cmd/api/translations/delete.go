package translations

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdDelete returns `poeditor api translations delete` (API v2
// translations/delete).
func NewCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var (
		project  string
		language string
		context  string
		fromFile string
		yes      bool
		asJSON   bool
	)

	cmd := &cobra.Command{
		Use:   "delete [<term>]",
		Short: "Delete translations in one language (the terms stay)",
		Long: `Delete the translation of a term in one language, or of many terms listed in
a JSON file (--from-file, - for stdin). The terms and their other languages
stay.

  [{"term": "save", "context": "button"}, {"term": "old_key"}]

This cannot be undone: without --yes nothing is deleted, and the command
prints the line to run. The counts go to stderr; --json also prints them on
stdout.`,
		Example: `  poeditor api translations delete save --project web --language fr --context button --yes
  poeditor api translations delete -p 7717 -l fr --from-file stale.json --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return projectflag.ErrMissing
			}
			if err := cmdutil.CheckBulkInput(cmd, args, fromFile, 1, "a term", "context"); err != nil {
				return err
			}
			keys, err := termKeys(f, fromFile, args, context)
			if err != nil {
				return err
			}
			if !yes {
				return fmt.Errorf("deleting the %s translation of %s cannot be undone; nothing was deleted. To delete, run:\n  %s %s",
					language, what(keys, fromFile), cmd.CommandPath(), cmdutil.ShellJoin(confirmArgs(f, args, language, context, fromFile, project)))
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			stats, err := p.Client.TranslationsDelete(cmd.Context(), p.ID, language, keys)
			if err != nil {
				return p.LanguageError(cmd.Context(), "--language "+language, err)
			}
			var c deleteCounts
			if t := stats.Translations; t != nil {
				c = deleteCounts{Parsed: t.Parsed, Deleted: t.Deleted}
			}
			note := ""
			if left := len(keys) - c.Deleted; left > 0 {
				note = fmt.Sprintf("%d not deleted: the term has no %s translation, or is not in the project.", left, language)
			}
			return report(f, p, language, asJSON, fmt.Sprintf("Deleted %d of %s from project %s.", c.Deleted, nTranslations(len(keys), language), p.Label), note, c)
		},
	}

	projectflag.Add(cmd, &project)
	addLanguageFlag(cmd, &language)
	cmd.Flags().StringVar(&context, "context", "", "The term's `context` (with its text, it finds the term)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "Delete the translations of the terms listed in a JSON `file` (- for stdin) instead")
	cmd.Flags().BoolVar(&yes, "yes", false, "Delete them: this cannot be undone")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// deleteCounts are the counts of translations/delete in --json.
type deleteCounts struct {
	Parsed  int `json:"parsed"`
	Deleted int `json:"deleted"`
}

func termKeys(f *cmdutil.Factory, fromFile string, args []string, context string) ([]api.TermKey, error) {
	if fromFile != "" {
		return cmdutil.ReadJSONList(f, fromFile, api.TermKey.Check)
	}
	key := api.TermKey{Term: args[0], Context: context}
	if err := key.Check(); err != nil {
		return nil, fmt.Errorf("the term is empty")
	}
	return []api.TermKey{key}, nil
}

// what names the terms in the --yes message.
func what(keys []api.TermKey, fromFile string) string {
	if fromFile != "" {
		n := "1 term"
		if len(keys) != 1 {
			n = fmt.Sprintf("%d terms", len(keys))
		}
		return fmt.Sprintf("the %s of %s", n, fromFile)
	}
	k := keys[0]
	if k.Context != "" {
		return fmt.Sprintf("term %s (context %s)", strconv.Quote(k.Term), strconv.Quote(k.Context))
	}
	return "term " + strconv.Quote(k.Term)
}

// confirmArgs is the command line that deletes for real: the user's input
// plus --yes.
func confirmArgs(f *cmdutil.Factory, args []string, language, context, fromFile, project string) []string {
	var out []string
	if fromFile != "" {
		out = []string{"--from-file", fromFile}
	} else {
		out = append(out, args[0])
		if context != "" {
			out = append(out, "--context", context)
		}
	}
	out = append(out, "--project", project, "--language", language)
	if f.Account != "" {
		out = append(out, "--account", f.Account)
	}
	return append(out, "--yes")
}
