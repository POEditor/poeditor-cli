package projects

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
	"github.com/POEditor/poeditor-cli/internal/transfer"
)

// NewCmdExport returns `poeditor api projects export` (API v2
// projects/export).
func NewCmdExport(f *cmdutil.Factory) *cobra.Command {
	var (
		project string
		opts    api.ExportOptions
		output  string
		asJSON  bool
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Download one language of a project as a file",
		Long: `Export one language of a project in a file format, and print the file — or
write it with --output (atomically; a file that would not change is left
alone). With --json, the export is described instead: its download link and,
with --output, the file written.

--format is the API's name (po, key_value_json, apple_strings, xliff…).
--filter keeps some terms: ` + strings.Join(repoconfig.PullFilters, ", ") + `.

To download every file of ` + repoconfig.FileName + `, use ` + "`poeditor pull`" + `.`,
		Example: `  poeditor api projects export --project web --language fr --format po > fr.po
  poeditor api projects export -p 7717 -l de --format key_value_json -o locales/de.json
  poeditor api projects export -p web -l fr --format po --filter translated --tag v2 --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := checkExport(opts); err != nil {
				return err
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			link, err := p.Client.ProjectsExport(cmd.Context(), p.ID, opts)
			if err != nil {
				return p.LanguageError(cmd.Context(), "--language "+opts.Language, err)
			}
			result := exportResult{Project: p.Ref, Language: opts.Language, Format: opts.Type, URL: link}
			if output == "" && asJSON {
				return cmdutil.PrintJSON(f.IO, result)
			}
			data, err := p.Client.Download(cmd.Context(), link)
			if err != nil {
				return err
			}
			if output == "" {
				_, err := f.IO.Out.Write(data)
				return err
			}
			path, err := f.ResolvePath(output)
			if err != nil {
				return err
			}
			status, err := transfer.WriteFile(path, data, false)
			if err != nil {
				return err
			}
			fmt.Fprintf(f.IO.ErrOut, "%s %s\n", status, output)
			if asJSON {
				result.File, result.Status = output, status
				return cmdutil.PrintJSON(f.IO, result)
			}
			return nil
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().StringVarP(&opts.Language, "language", "l", "", "The `language` to export (its code, e.g. fr) (required)")
	cmd.Flags().StringVar(&opts.Type, "format", "", "The file `format`, e.g. po, key_value_json (required)")
	cmd.Flags().StringArrayVar(&opts.Filters, "filter", nil, "Only the terms that are `filter` (repeat: all must match)")
	cmd.Flags().StringArrayVar(&opts.Tags, "tag", nil, "Only the terms with this `tag` (repeat for more)")
	cmd.Flags().StringVar(&opts.TagsMatch, "tags-match", "", "How --tag matches (`mode`: "+strings.Join(api.TagsMatchModes, ", ")+"; the API's default: any)")
	cmd.Flags().StringVar(&opts.Order, "order", "", "Sort by `terms`, alphabetically (the only order)")
	cmd.Flags().StringVar(&opts.FallbackLanguage, "fallback-language", "", "Fill untranslated terms from this `language`")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Write the file to `path` instead of stdout")
	_ = cmd.MarkFlagRequired("language")
	_ = cmd.MarkFlagRequired("format")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// exportResult is what --json prints: our own shape.
type exportResult struct {
	Project  repoconfig.ProjectRef `json:"project"`
	Language string                `json:"language"`
	Format   string                `json:"format"`
	URL      string                `json:"url"`
	File     string                `json:"file,omitempty"`   // with --output
	Status   transfer.Status       `json:"status,omitempty"` // with --output: created, updated, unchanged
}

// checkExport refuses values the API would drop silently.
func checkExport(o api.ExportOptions) error {
	for _, filter := range o.Filters {
		if !slices.Contains(repoconfig.PullFilters, filter) {
			return fmt.Errorf("--filter %q: use %s", filter, strings.Join(repoconfig.PullFilters, ", "))
		}
	}
	if o.TagsMatch != "" {
		if !slices.Contains(api.TagsMatchModes, o.TagsMatch) {
			return fmt.Errorf("--tags-match %q: use %s", o.TagsMatch, strings.Join(api.TagsMatchModes, ", "))
		}
		if len(o.Tags) == 0 {
			return fmt.Errorf("--tags-match needs --tag")
		}
	}
	if o.Order != "" && o.Order != "terms" {
		return fmt.Errorf("--order %q: the only order is terms", o.Order)
	}
	return nil
}
