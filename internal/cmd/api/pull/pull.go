// Package pull is `poeditor pull`: download translations into the files
// listed in .poeditor.yml.
package pull

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
	"github.com/POEditor/poeditor-cli/internal/transfer"
)

// Result is one file in one language, as reported by --json.
type Result struct {
	Project  transfer.ProjectRef `json:"project"`
	File     string              `json:"file"`
	Language string              `json:"language"`
	Status   string              `json:"status"`
}

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

			results := []Result{}
			runErr := func() error {
				for _, p := range projects {
					client, err := transfer.Client(f, p)
					if err != nil {
						return err
					}
					langs, err := transfer.ProjectLanguages(cmd.Context(), client, p.ID)
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
						if dryRun && status != transfer.Unchanged {
							status = "would_" + strings.TrimSuffix(status, "d") // would_create, would_update
						}
						results = append(results, Result{Project: transfer.Ref(p), File: item.Display, Language: item.Language, Status: status})
						fmt.Fprintf(f.IO.ErrOut, "%-14s %s\n", strings.ReplaceAll(status, "_", " "), item.Display)
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
func pullOne(cmd *cobra.Command, client *api.Client, item transfer.Item, dryRun bool) (string, error) {
	params := ExportParams(item)
	var result struct {
		URL string `json:"url"`
	}
	if err := client.Post(cmd.Context(), "projects/export", params, &result); err != nil {
		return "", err
	}
	if result.URL == "" {
		return "", fmt.Errorf("the API returned no download link")
	}
	data, err := client.Download(cmd.Context(), result.URL)
	if err != nil {
		return "", err
	}
	return transfer.WriteFile(item.Path, data, dryRun)
}

// ExportParams are the projects/export parameters for one item.
func ExportParams(item transfer.Item) url.Values {
	params := url.Values{
		"id":       {strconv.Itoa(item.Project.ID)},
		"language": {item.Language},
		"type":     {item.File.Format},
	}
	if p := item.File.Pull; p != nil {
		if len(p.Filters) > 0 {
			params.Set("filters", transfer.JSONList(p.Filters))
		}
		if len(p.Tags) > 0 {
			params.Set("tags", transfer.JSONList(p.Tags))
			if p.TagsMatch != "" {
				params.Set("tags_match", p.TagsMatch)
			}
		}
		if p.Order != "" {
			params.Set("order", p.Order)
		}
		if p.FallbackLanguage != "" {
			params.Set("fallback_language", p.FallbackLanguage)
		}
	}
	return params
}

func summary(results []Result, dryRun bool) string {
	counts := map[string]int{}
	for _, r := range results {
		counts[r.Status]++
	}
	files := "files"
	if len(results) == 1 {
		files = "file"
	}
	if dryRun {
		return fmt.Sprintf("%d %s: %d would be created, %d would be updated, %d unchanged (dry run: nothing was written)",
			len(results), files, counts["would_create"], counts["would_update"], counts[transfer.Unchanged])
	}
	return fmt.Sprintf("%d %s: %d created, %d updated, %d unchanged",
		len(results), files, counts[transfer.Created], counts[transfer.Updated], counts[transfer.Unchanged])
}
