package terms

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// updateFlags are the flags that change a term; with --from-file each entry
// has them instead.
var updateFlags = []string{"new-term", "new-context", "plural", "reference", "tag"}

// NewCmdUpdate returns `poeditor api terms update` (API v2 terms/update).
func NewCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	var (
		project      string
		context      string
		newTerm      string
		newContext   string
		plural       string
		reference    string
		tags         []string
		fuzzyTrigger bool
		fromFile     string
		asJSON       bool
	)

	cmd := &cobra.Command{
		Use:   "update [<term>]",
		Short: "Change terms: text, context, plural, reference, tags",
		Long: `Change a term of a project, found by its text and --context, or many from a
JSON file (--from-file, - for stdin). Only what you give changes: --plural ""
or --reference "" clears that field; --tag replaces all the term's tags.

The file is a list in the API's shape. "term" and "context" find the term;
the other fields are optional, and "tags": [] removes every tag:

  [
    {"term": "save", "context": "button", "new_term": "Save",
     "reference": "src/save.js:14", "tags": ["web"]},
    {"term": "old_key", "new_term": "new_key", "new_context": "home"}
  ]

--fuzzy-trigger marks the translations of every updated term fuzzy, so they
get reviewed. The counts go to stderr; --json also prints them on stdout.`,
		Example: `  poeditor api terms update save --project web --context button --new-term Save
  poeditor api terms update welcome -p 7717 --tag web --tag v2
  poeditor api terms update -p web --from-file renames.json --fuzzy-trigger`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.CheckBulkInput(cmd, args, fromFile, 1, "a term", append([]string{"context"}, updateFlags...)...); err != nil {
				return err
			}
			var updates []api.TermUpdate
			var err error
			if fromFile != "" {
				updates, err = cmdutil.ReadJSONList(f, fromFile, api.TermUpdate.Check)
			} else {
				changed := func(name string, v *string) *string {
					if cmd.Flags().Changed(name) {
						return v
					}
					return nil
				}
				u := api.TermUpdate{
					TermKey:    api.TermKey{Term: args[0], Context: context},
					NewTerm:    changed("new-term", &newTerm),
					NewContext: changed("new-context", &newContext),
					Plural:     changed("plural", &plural),
					Reference:  changed("reference", &reference),
				}
				if cmd.Flags().Changed("tag") {
					u.Tags = &tags
				}
				if u.NewTerm == nil && u.NewContext == nil && u.Plural == nil && u.Reference == nil && u.Tags == nil && !fuzzyTrigger {
					return errors.New("nothing to change: give --new-term, --new-context, --plural, --reference, --tag or --fuzzy-trigger")
				}
				if err := u.Check(); err != nil {
					return fmt.Errorf("the term: %w", err)
				}
				updates = []api.TermUpdate{u}
			}
			if err != nil {
				return err
			}

			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			stats, err := p.Client.TermsUpdate(cmd.Context(), p.ID, updates, api.TermsUpdateOptions{FuzzyTrigger: fuzzyTrigger})
			if err != nil {
				return err
			}
			var c updateCounts
			if t := stats.Terms; t != nil {
				c = updateCounts{Parsed: t.Parsed, Updated: t.Updated}
			}
			return report(f, p, asJSON, fmt.Sprintf("Updated %d of %s in project %s.", c.Updated, nTerms(c.Parsed), p.Label), notFound(c.Parsed-c.Updated), c)
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().StringVar(&context, "context", "", "The term's `context` (with its text, it finds the term)")
	cmd.Flags().StringVar(&newTerm, "new-term", "", "Change the term's `text`")
	cmd.Flags().StringVar(&newContext, "new-context", "", "Change the term's `context`")
	cmd.Flags().StringVar(&plural, "plural", "", "Set the plural `form` (\"\" makes it a normal term)")
	cmd.Flags().StringVar(&reference, "reference", "", "Set where the term is used (\"\" clears it)")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Replace the term's tags with this `tag` (repeat for more)")
	cmd.Flags().BoolVar(&fuzzyTrigger, "fuzzy-trigger", false, "Mark the updated terms' translations fuzzy")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "Apply the updates of a JSON `file` (- for stdin) instead")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// updateCounts are the counts of terms/update in --json.
type updateCounts struct {
	Parsed  int `json:"parsed"`
	Updated int `json:"updated"`
}
