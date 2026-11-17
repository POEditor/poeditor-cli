// Package push is `poeditor push`: upload the source-language files of
// .poeditor.yml (and, with --translations, every other language).
package push

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
	"github.com/POEditor/poeditor-cli/internal/transfer"
)

// Statuses.
const (
	Uploaded    transfer.Status = "uploaded"
	Missing     transfer.Status = "missing"
	WouldUpload transfer.Status = "would_upload"
)

// Result is one file in one language, as reported by --json.
type Result struct {
	transfer.FileResult
	transfer.UploadCounts
}

// upload is one planned upload.
type upload struct {
	transfer.Item
	client *api.Client
	opts   api.UploadOptions
}

// NewCmdPush returns `poeditor push`.
func NewCmdPush(f *cmdutil.Factory) *cobra.Command {
	var (
		project      string
		languages    []string
		translations bool
		dryRun       bool
		asJSON       bool
	)

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Upload the source files of " + repoconfig.FileName + " to POEditor",
		Long: `Upload the source files of ` + repoconfig.FileName + ` to POEditor.

By default only the source_language file of each file entry is uploaded, with
updating=terms_translations: new and changed terms plus their source text.
With --translations every other language is uploaded too, with
updating=translations, so a translation file never creates terms; whether
existing translations are overwritten follows push.overwrite. A translation
file that does not exist locally is skipped.

POEditor accepts one upload every 10 seconds (20 on free accounts), so push
waits between uploads and retries when the API says it is too early. Files are
uploaded one at a time and push stops at the first error. Nothing is uploaded
if a source file is missing or a format cannot be uploaded.`,
		Example: `  poeditor push
  poeditor push --project web
  poeditor push --translations --language fr
  poeditor push --dry-run --translations`,
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
			if len(languages) > 0 && !translations {
				return errors.New("--language only applies with --translations (without it, push uploads the source language only)")
			}

			plan, err := makePlan(cmd, f, cfg, projects, translations, languages)
			if err != nil {
				return err
			}

			results := []Result{}
			runErr := func() error {
				uploads := 0
				for _, u := range plan {
					res := Result{FileResult: u.Result("")}
					data, err := os.ReadFile(u.Path)
					if errors.Is(err, fs.ErrNotExist) {
						res.Status = Missing
						results = append(results, res)
						fmt.Fprintf(f.IO.ErrOut, "%-14s %s\n", "missing", u.Display)
						continue
					}
					if err != nil {
						return &transfer.ErrStopped{Display: u.Display, Err: err}
					}
					if dryRun {
						res.Status = WouldUpload
						results = append(results, res)
						fmt.Fprintf(f.IO.ErrOut, "%-14s %s (%s)  %s\n", "would upload", u.Display, u.Language, describe(u))
						continue
					}
					if uploads > 0 {
						if err := transfer.Wait(cmd.Context(), f, transfer.UploadDelay, "before the next upload"); err != nil {
							return err
						}
					}
					counts, err := transfer.Upload(cmd.Context(), f, u.client, u.Project.ID, filepath.Base(u.Path), data, u.opts)
					if err != nil {
						return &transfer.ErrStopped{Display: u.Display, Err: err}
					}
					uploads++
					res.Status = Uploaded
					res.UploadCounts = transfer.CountsOf(counts)
					results = append(results, res)
					fmt.Fprintf(f.IO.ErrOut, "%-14s %s (%s)  %s\n", "uploaded", u.Display, u.Language, transfer.CountsOf(counts))
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
	cmd.Flags().BoolVar(&translations, "translations", false, "Also upload every other language (updating=translations)")
	cmd.Flags().StringSliceVar(&languages, "language", nil, "With --translations: only these languages, the source one included only if listed (repeat or comma-separate)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "List what would be uploaded without uploading")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// makePlan lists the uploads and checks, before anything is sent, that each
// format can be uploaded and every source file exists.
func makePlan(cmd *cobra.Command, f *cmdutil.Factory, cfg *repoconfig.Config, projects []repoconfig.Project, translations bool, only []string) ([]upload, error) {
	var plan []upload
	for _, p := range projects {
		if p.SourceLanguage == "" {
			return nil, fmt.Errorf("project %s has no source_language in %s: push needs it to know which file holds the terms", p.Label(), cfg.Path())
		}
		for _, file := range p.Files {
			if format, ok := repoconfig.Formats[file.Format]; ok && !format.Import {
				return nil, fmt.Errorf("project %s: %s: format %s can be pulled but not pushed", p.Label(), file.Path, file.Format)
			}
		}
		client, err := transfer.Client(f, p)
		if err != nil {
			return nil, err
		}

		langs := []string{p.SourceLanguage}
		if translations {
			if langs, err = client.LanguageCodes(cmd.Context(), p.ID); err != nil {
				return nil, fmt.Errorf("project %s: %w", p.Label(), err)
			}
		}
		items, err := transfer.Items(cfg, p, langs, only)
		if err != nil {
			return nil, err
		}
		// a file's own `languages:` list wins over langs in Items; push keeps
		// only the languages it plans to send (the source one, by default)
		items = slices.DeleteFunc(items, func(it transfer.Item) bool { return !slices.Contains(langs, it.Language) })

		// source files first, so terms exist before their translations arrive
		slices.SortStableFunc(items, func(a, b transfer.Item) int {
			return boolInt(b.Language == p.SourceLanguage) - boolInt(a.Language == p.SourceLanguage)
		})
		for _, item := range items {
			source := item.Language == p.SourceLanguage
			if source {
				if _, err := os.Stat(item.Path); errors.Is(err, fs.ErrNotExist) {
					return nil, fmt.Errorf("%s: source file not found (project %s, source_language %s); nothing was uploaded", item.Display, p.Label(), p.SourceLanguage)
				}
			}
			plan = append(plan, upload{Item: item, client: client, opts: uploadOptions(item, source)})
		}
	}
	return plan, nil
}

// uploadOptions maps a file's `push:` block onto projects/upload. The
// source file creates terms (terms_translations, or push.updating); every
// other language only brings translations. sync_terms applies to the source
// upload only.
func uploadOptions(item transfer.Item, source bool) api.UploadOptions {
	opts := api.UploadOptions{Updating: "translations", Language: item.Language}
	if source {
		opts.Updating = "terms_translations"
	}
	push := item.File.Push
	if push == nil {
		return opts
	}
	if source && push.Updating != "" {
		opts.Updating = push.Updating
	}
	opts.Overwrite = push.Overwrite
	opts.SyncTerms = push.SyncTerms && source
	opts.ReadFromSource = push.ReadFromSource
	opts.FuzzyTrigger = push.FuzzyTrigger
	if t := push.Tags; t != nil {
		opts.Tags = api.UploadTags{All: t.All, New: t.New, Obsolete: t.Obsolete, OverwrittenTranslations: t.OverwrittenTranslations}
	}
	return opts
}

// describe is the --dry-run detail of an upload.
func describe(u upload) string {
	parts := []string{"updating=" + u.opts.Updating}
	if u.opts.Overwrite {
		parts = append(parts, "overwrite")
	}
	if u.opts.SyncTerms {
		parts = append(parts, "sync_terms")
	}
	if tags := u.opts.Tags.String(); tags != "" {
		parts = append(parts, "tags="+tags)
	}
	return strings.Join(parts, ", ")
}

func summary(results []Result, dryRun bool) string {
	var uploaded, missing, added, deleted int
	for _, r := range results {
		switch r.Status {
		case Uploaded, WouldUpload:
			uploaded++
		case Missing:
			missing++
		}
		if r.Terms != nil {
			added += r.Terms.Added
			deleted += r.Terms.Deleted
		}
	}
	files := "files"
	if uploaded == 1 {
		files = "file"
	}
	var line string
	if dryRun {
		line = fmt.Sprintf("%d %s would be uploaded (dry run: nothing was sent)", uploaded, files)
	} else {
		line = fmt.Sprintf("%d %s uploaded: %d terms added, %d deleted", uploaded, files, added, deleted)
	}
	if missing > 0 {
		line += fmt.Sprintf("; %d missing locally", missing)
	}
	return line
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
