// Package pull is `poeditor pull`: download translations into the files
// listed in .poeditor.yml.
package pull

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
	"github.com/POEditor/poeditor-cli/internal/transfer"
)

// NewCmdPull returns `poeditor pull`.
func NewCmdPull(f *cmdutil.Factory) *cobra.Command {
	var (
		project   string
		languages []string
		dryRun    bool
		asJSON    bool
	)

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Download translations into the files of " + repoconfig.FileName,
		Long: `Download translations into the files of ` + repoconfig.FileName + `.

For every project, file and language in the config, the file is exported from
POEditor (with the file's pull: options) and written to its path. POEditor wins:
local changes are overwritten. Files whose content did not change are left
untouched. Languages are the file's languages: list, or every language of the
project.

Files are downloaded one at a time, and pull stops at the first error; files
written before it stay written.`,
		Example: `  poeditor pull
  poeditor pull --project web --language fr --language de
  poeditor pull --dry-run`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := transfer.LoadConfig(f)
			if err != nil {
				return err
			}
			projects, err := cfg.Select(project)
			if err != nil {
				return err
			}

			results := []transfer.FileResult{}
			runErr := func() error {
				for _, p := range projects {
					client, err := transfer.Client(f, p)
					if err != nil {
						return err
					}
					langs, err := client.LanguageCodes(cmd.Context(), p.ID)
					if err != nil {
						return fmt.Errorf("project %s: %w", p.Label(), err)
					}
					items, err := transfer.Items(cfg, p, langs, languages)
					if err != nil {
						return err
					}
					for _, item := range items {
						status, err := pullOne(cmd, client, item, dryRun)
						if err != nil {
							return &transfer.ErrStopped{Display: item.Display, Err: err}
						}
						results = append(results, item.Result(status))
						fmt.Fprintf(f.IO.ErrOut, "%-14s %s\n", strings.ReplaceAll(string(status), "_", " "), item.Display)
					}
				}
				return nil
			}()

			if asJSON {
				if err := cmdutil.PrintJSON(f.IO, results); err != nil {
					return err
				}
			}
			if runErr != nil {
				return runErr
			}
			fmt.Fprintln(f.IO.ErrOut, summary(results, dryRun))
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Only this project (its alias or id)")
	cmd.Flags().StringSliceVar(&languages, "language", nil, "Only these languages (repeat or comma-separate)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change without writing anything")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// pullOne exports one file in one language and writes it.
func pullOne(cmd *cobra.Command, client *api.Client, item transfer.Item, dryRun bool) (transfer.Status, error) {
	data, err := transfer.Export(cmd.Context(), client, item.Project.ID, exportOptions(item))
	if err != nil {
		return "", err
	}
	return transfer.WriteFile(item.Path, data, dryRun)
}

// exportOptions maps a file's `pull:` block onto projects/export.
func exportOptions(item transfer.Item) api.ExportOptions {
	opts := api.ExportOptions{Language: item.Language, Type: item.File.Format}
	if p := item.File.Pull; p != nil {
		opts.Filters = p.Filters
		opts.Tags = p.Tags
		opts.TagsMatch = p.TagsMatch
		opts.Order = p.Order
		opts.FallbackLanguage = p.FallbackLanguage
	}
	return opts
}

func summary(results []transfer.FileResult, dryRun bool) string {
	counts := map[transfer.Status]int{}
	for _, r := range results {
		counts[r.Status]++
	}
	files := "files"
	if len(results) == 1 {
		files = "file"
	}
	if dryRun {
		return fmt.Sprintf("%d %s: %d would be created, %d would be updated, %d unchanged (dry run: nothing was written)",
			len(results), files, counts[transfer.WouldCreate], counts[transfer.WouldUpdate], counts[transfer.Unchanged])
	}
	return fmt.Sprintf("%d %s: %d created, %d updated, %d unchanged",
		len(results), files, counts[transfer.Created], counts[transfer.Updated], counts[transfer.Unchanged])
}
