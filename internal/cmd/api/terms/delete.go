package terms

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdDelete returns `poeditor api terms delete` (API v2 terms/delete).
func NewCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var (
		project  string
		context  string
		fromFile string
		yes      bool
		asJSON   bool
	)

	cmd := &cobra.Command{
		Use:   "delete [<term>]",
		Short: "Delete terms from a project, with their translations",
		Long: `Delete a term of a project, found by its text and --context, or many from a
JSON file (--from-file, - for stdin) listing them:

  [{"term": "save", "context": "button"}, {"term": "old_key"}]

The terms' translations are deleted with them, and this cannot be undone:
without --yes nothing is deleted, and the command prints the line to run.
The counts go to stderr; --json also prints them on stdout.`,
		Example: `  poeditor api terms delete save --project web --context button --yes
  poeditor api terms delete -p 7717 --from-file obsolete.json --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return projectflag.ErrMissing
			}
			if err := checkInput(cmd, args, fromFile, 1, "a term", "context"); err != nil {
				return err
			}
			keys, err := termKeys(f, fromFile, args, context)
			if err != nil {
				return err
			}
			if !yes {
				their := "their"
				if fromFile == "" {
					their = "its"
				}
				return fmt.Errorf("deleting %s deletes %s translations too and cannot be undone; nothing was deleted. To delete, run:\n  %s %s",
					what(keys, fromFile), their, cmd.CommandPath(), cmdutil.ShellJoin(confirmArgs(f, args, context, fromFile, project)))
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			stats, err := p.Client.TermsDelete(cmd.Context(), p.ID, keys)
			if err != nil {
				return err
			}
			var c deleteCounts
			if t := stats.Terms; t != nil {
				c = deleteCounts{Parsed: t.Parsed, Deleted: t.Deleted}
			}
			return report(f, p, asJSON, fmt.Sprintf("Deleted %d of %s from project %s.", c.Deleted, nTerms(c.Parsed), p.Label), notFound(c.Parsed-c.Deleted), c)
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().StringVar(&context, "context", "", "The term's `context` (with its text, it finds the term)")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "Delete the terms listed in a JSON `file` (- for stdin) instead")
	cmd.Flags().BoolVar(&yes, "yes", false, "Delete them: their translations go with them, and this cannot be undone")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// deleteCounts are the counts of terms/delete in --json.
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

// what names the terms to delete in the --yes message.
func what(keys []api.TermKey, fromFile string) string {
	if fromFile != "" {
		return fmt.Sprintf("the %s of %s", nTerms(len(keys)), fromFile)
	}
	k := keys[0]
	if k.Context != "" {
		return fmt.Sprintf("term %s (context %s)", strconv.Quote(k.Term), strconv.Quote(k.Context))
	}
	return "term " + strconv.Quote(k.Term)
}

// confirmArgs is the command line that deletes for real: the user's input
// plus --yes.
func confirmArgs(f *cmdutil.Factory, args []string, context, fromFile, project string) []string {
	var out []string
	if fromFile != "" {
		out = []string{"--from-file", fromFile}
	} else {
		out = append(out, args[0])
		if context != "" {
			out = append(out, "--context", context)
		}
	}
	out = append(out, "--project", project)
	if f.Account != "" {
		out = append(out, "--account", f.Account)
	}
	return append(out, "--yes")
}
