package projects

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// NewCmdSync returns `poeditor api projects sync` (API v2 projects/sync).
func NewCmdSync(f *cmdutil.Factory) *cobra.Command {
	var (
		project string
		yes     bool
		asJSON  bool
	)

	cmd := &cobra.Command{
		Use:   "sync <file.json>",
		Short: "Make a project's terms exactly those of a JSON file (deletes the others)",
		Long: `Make a project's terms exactly the list in a JSON file (- for stdin): new terms
are added, existing ones get the file's reference, plural, tags and comment,
and EVERY OTHER TERM IS DELETED, with its translations. Without --yes nothing
is sent, and the command prints the line to run.

The file is a list in the API's shape; only "term" is required, and every
entry must have one (POEditor would otherwise delete every term):

  [
    {"term": "save", "context": "button", "reference": "src/save.js:12", "tags": ["web"]},
    {"term": "%d files", "plural": "%d files"}
  ]

To add terms without deleting any, use ` + "`poeditor api terms add`" + `. The counts go to
stderr; --json also prints them on stdout.`,
		Example: `  poeditor api projects sync terms.json --project web --yes`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return projectflag.ErrMissing
			}
			path := args[0]
			terms, err := cmdutil.ReadJSONFile(f, "", path, func(t api.NewTerm) error { return t.TermKey.Check() })
			if err != nil {
				return err
			}
			if !yes {
				confirm := []string{path, "--project", project}
				if f.Account != "" {
					confirm = append(confirm, "--account", f.Account)
				}
				return fmt.Errorf("syncing project %s with the %d terms of %s deletes every other term, with its translations; nothing was sent. To sync, run:\n  %s %s",
					project, len(terms), path, cmd.CommandPath(), cmdutil.ShellJoin(append(confirm, "--yes")))
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			stats, err := p.Client.ProjectsSync(cmd.Context(), p.ID, terms)
			if err != nil {
				return err
			}
			var c syncCounts
			if t := stats.Terms; t != nil {
				c = syncCounts{Parsed: t.Parsed, Added: t.Added, Updated: t.Updated, Deleted: t.Deleted}
			}
			fmt.Fprintf(f.IO.ErrOut, "Synced project %s: %d terms parsed, %d added, %d updated, %d deleted.\n", p.Label, c.Parsed, c.Added, c.Updated, c.Deleted)
			if asJSON {
				return cmdutil.PrintJSON(f.IO, syncResult{Project: p.Ref, Terms: c})
			}
			return nil
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().BoolVar(&yes, "yes", false, "Sync: every term not in the file is deleted")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// syncResult is what --json prints: our own shape.
type syncResult struct {
	Project repoconfig.ProjectRef `json:"project"`
	Terms   syncCounts            `json:"terms"`
}

type syncCounts struct {
	Parsed  int `json:"parsed"`
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
}
