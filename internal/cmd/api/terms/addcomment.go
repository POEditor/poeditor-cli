package terms

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdAddComment returns `poeditor api terms add-comment` (API v2
// terms/add_comment).
func NewCmdAddComment(f *cmdutil.Factory) *cobra.Command {
	var (
		project  string
		context  string
		pinned   bool
		fromFile string
		asJSON   bool
	)

	cmd := &cobra.Command{
		Use:   "add-comment [<term> <text>]",
		Short: "Comment on terms",
		Long: `Add a comment to a term of a project, found by its text and --context, or
to many from a JSON file (--from-file, - for stdin). The comment is the
term's, for every language.

  [
    {"term": "save", "context": "button", "comment": "Max 10 characters", "pinned": 1},
    {"term": "welcome", "comment": "Shown once, on first launch"}
  ]

The counts go to stderr; --json also prints them on stdout.`,
		Example: `  poeditor api terms add-comment save "Max 10 characters" --project web --context button
  poeditor api terms add-comment welcome "Keep it short" -p 7717 --pinned
  poeditor api terms add-comment -p web --from-file comments.json`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.CheckBulkInput(cmd, args, fromFile, 2, "a term and the comment", "context", "pinned"); err != nil {
				return err
			}
			var comments []api.TermComment
			var err error
			if fromFile != "" {
				comments, err = cmdutil.ReadJSONList(f, fromFile, api.TermComment.Check)
				if err != nil {
					return err
				}
			} else {
				c := api.TermComment{TermKey: api.TermKey{Term: args[0], Context: context}, Comment: args[1]}
				if pinned {
					c.Pinned = 1
				}
				if err := c.Check(); err != nil {
					return fmt.Errorf("the comment: %w", err)
				}
				comments = []api.TermComment{c}
			}

			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			stats, err := p.Client.TermsAddComment(cmd.Context(), p.ID, comments)
			if err != nil {
				return err
			}
			var c commentCounts
			if t := stats.Terms; t != nil {
				c = commentCounts{Parsed: t.Parsed, WithAddedComment: t.WithAddedComment}
			}
			return report(f, p, asJSON, fmt.Sprintf("Commented on %d of %s in project %s.", c.WithAddedComment, nTerms(c.Parsed), p.Label), notFound(c.Parsed-c.WithAddedComment), c)
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().StringVar(&context, "context", "", "The term's `context` (with its text, it finds the term)")
	cmd.Flags().BoolVar(&pinned, "pinned", false, "Pin the comment")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "Add the comments of a JSON `file` (- for stdin) instead")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// commentCounts are the counts of terms/add_comment in --json.
type commentCounts struct {
	Parsed           int `json:"parsed"`
	WithAddedComment int `json:"with_added_comment"`
}
