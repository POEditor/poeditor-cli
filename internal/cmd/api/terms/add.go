package terms

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdAdd returns `poeditor api terms add` (API v2 terms/add).
func NewCmdAdd(f *cmdutil.Factory) *cobra.Command {
	var (
		project  string
		term     api.NewTerm
		fromFile string
		asJSON   bool
	)

	cmd := &cobra.Command{
		Use:   "add [<term>]",
		Short: "Add terms to a project",
		Long: `Add a term to a project, or many from a JSON file (--from-file, - for stdin).
A term already in the project (same text and context) is left as it is.

The file is a list in the API's shape; only "term" is required:

  [
    {"term": "save", "context": "button", "reference": "src/save.js:12",
     "comment": "Max 10 characters", "tags": ["web", "v2"]},
    {"term": "%d files", "plural": "%d files"}
  ]

The counts go to stderr; --json also prints them on stdout.`,
		Example: `  poeditor api terms add save --project web --context button --tag web --tag v2
  poeditor api terms add "%d file" -p 7717 --plural "%d files"
  poeditor api terms add -p web --from-file new-terms.json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkInput(cmd, args, fromFile, 1, "a term", "context", "plural", "reference", "comment", "tag"); err != nil {
				return err
			}
			terms, err := newTerms(f, fromFile, args, term)
			if err != nil {
				return err
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			stats, err := p.Client.TermsAdd(cmd.Context(), p.ID, terms)
			if err != nil {
				return err
			}
			var c addCounts
			if t := stats.Terms; t != nil {
				c = addCounts{Parsed: t.Parsed, Added: t.Added}
			}
			note := ""
			if c.Parsed > c.Added {
				note = fmt.Sprintf("%s already in the project.", nTerms(c.Parsed-c.Added))
			}
			return report(f, p, asJSON, fmt.Sprintf("Added %d of %s to project %s.", c.Added, nTerms(c.Parsed), p.Label), note, c)
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().StringVar(&term.Context, "context", "", "The term's `context`")
	cmd.Flags().StringVar(&term.Plural, "plural", "", "The term's plural `form` (makes it a plural term)")
	cmd.Flags().StringVar(&term.Reference, "reference", "", "Where the term is used (e.g. src/save.js:12)")
	cmd.Flags().StringVar(&term.Comment, "comment", "", "A comment for translators")
	cmd.Flags().StringArrayVar(&term.Tags, "tag", nil, "A `tag` for the term (repeat for more)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "Add the terms of a JSON `file` (- for stdin) instead")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// addCounts are the counts of terms/add in --json.
type addCounts struct {
	Parsed int `json:"parsed"`
	Added  int `json:"added"`
}

// newTerms is the list to send: the file's, or the one term of the
// arguments and flags.
func newTerms(f *cmdutil.Factory, fromFile string, args []string, term api.NewTerm) ([]api.NewTerm, error) {
	if fromFile != "" {
		return cmdutil.ReadJSONList(f, fromFile, func(t api.NewTerm) error { return t.TermKey.Check() })
	}
	term.Term = args[0]
	if err := term.TermKey.Check(); err != nil {
		return nil, fmt.Errorf("the term is empty")
	}
	return []api.NewTerm{term}, nil
}
