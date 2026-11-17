package projects

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
	"github.com/POEditor/poeditor-cli/internal/transfer"
)

// NewCmdUpload returns `poeditor api projects upload` (API v2
// projects/upload).
func NewCmdUpload(f *cmdutil.Factory) *cobra.Command {
	var (
		project string
		opts    api.UploadOptions
		yes     bool
		asJSON  bool
	)

	cmd := &cobra.Command{
		Use:   "upload <file>",
		Short: "Upload a file of terms and/or translations to a project",
		Long: `Upload one file to a project. --updating says what it brings: terms (new terms
only), terms_translations (terms and the --language translations) or
translations (translations only, never new terms). POEditor reads the format
from the file.

--overwrite replaces existing translations; --sync-terms DELETES every term
that is not in the file, with its translations, so it needs --yes. Tags go to
all the file's terms (--tag-all), the new ones (--tag-new), the ones missing
from the file (--tag-obsolete) or those whose translation was overwritten
(--tag-overwritten).

POEditor accepts one upload every 10 seconds (20 on free accounts): when it
says it is too early, the command waits and retries. To upload the files of
` + repoconfig.FileName + `, use ` + "`poeditor push`" + `.`,
		Example: `  poeditor api projects upload en.json --project web --updating terms_translations --language en
  poeditor api projects upload fr.po -p 7717 --updating translations -l fr --overwrite
  poeditor api projects upload en.json -p web --updating terms --sync-terms --tag-new v2 --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return projectflag.ErrMissing
			}
			if err := checkUpload(opts); err != nil {
				return err
			}
			name := args[0]
			if name == "-" {
				return errors.New("upload a file by its name: POEditor reads the format from it (- is not supported)")
			}
			data, err := f.ReadFile(name)
			if err != nil {
				return err
			}
			if opts.SyncTerms && !yes {
				confirm := append([]string{name, "--project", project}, uploadArgs(opts)...)
				if f.Account != "" {
					confirm = append(confirm, "--account", f.Account)
				}
				return fmt.Errorf("--sync-terms deletes every term of project %s that is not in %s, with its translations; nothing was sent. To upload, run:\n  %s %s",
					project, name, cmd.CommandPath(), cmdutil.ShellJoin(append(confirm, "--yes")))
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			stats, err := transfer.Upload(cmd.Context(), f, p.Client, p.ID, filepath.Base(name), data, opts)
			if err != nil {
				if opts.Language != "" {
					return p.LanguageError(cmd.Context(), "--language "+opts.Language, err)
				}
				return err
			}
			counts := transfer.CountsOf(stats)
			fmt.Fprintf(f.IO.ErrOut, "Uploaded %s to project %s: %s\n", name, p.Label, counts)
			if asJSON {
				return cmdutil.PrintJSON(f.IO, uploadResult{Project: p.Ref, File: name, Language: opts.Language, UploadCounts: counts})
			}
			return nil
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().StringVar(&opts.Updating, "updating", "", "What the file brings (`mode`: "+strings.Join(repoconfig.UpdatingModes, ", ")+"; required)")
	cmd.Flags().StringVarP(&opts.Language, "language", "l", "", "The `language` of the file's translations (required unless --updating terms)")
	cmd.Flags().BoolVar(&opts.Overwrite, "overwrite", false, "Replace existing translations")
	cmd.Flags().BoolVar(&opts.SyncTerms, "sync-terms", false, "Delete the terms that are not in the file (needs --yes)")
	cmd.Flags().BoolVar(&opts.ReadFromSource, "read-from-source", false, "Read the translations from the file's source text (xliff)")
	cmd.Flags().BoolVar(&opts.FuzzyTrigger, "fuzzy-trigger", false, "Mark the other languages' translations fuzzy where the source changed")
	cmd.Flags().StringArrayVar(&opts.Tags.All, "tag-all", nil, "Tag every term of the file with `tag` (repeat)")
	cmd.Flags().StringArrayVar(&opts.Tags.New, "tag-new", nil, "Tag the new terms with `tag` (repeat)")
	cmd.Flags().StringArrayVar(&opts.Tags.Obsolete, "tag-obsolete", nil, "Tag the project's terms missing from the file with `tag` (repeat)")
	cmd.Flags().StringArrayVar(&opts.Tags.OverwrittenTranslations, "tag-overwritten", nil, "Tag the terms whose translation was overwritten with `tag` (repeat)")
	cmd.Flags().BoolVar(&yes, "yes", false, "With --sync-terms: upload, deleting the terms not in the file")
	_ = cmd.MarkFlagRequired("updating")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// uploadResult is what --json prints: our own shape, the counts as push's.
type uploadResult struct {
	Project  repoconfig.ProjectRef `json:"project"`
	File     string                `json:"file"`
	Language string                `json:"language,omitempty"`
	transfer.UploadCounts
}

// checkUpload checks the flags before the file is read or anything is sent.
func checkUpload(o api.UploadOptions) error {
	if !slices.Contains(repoconfig.UpdatingModes, o.Updating) {
		return fmt.Errorf("--updating %q: use %s", o.Updating, strings.Join(repoconfig.UpdatingModes, ", "))
	}
	if o.Updating != "terms" && o.Language == "" {
		return fmt.Errorf("--updating %s needs --language: the language of the file's translations", o.Updating)
	}
	return nil
}

// uploadArgs are the flags of opts, for the line that repeats the command.
func uploadArgs(o api.UploadOptions) []string {
	args := []string{"--updating", o.Updating}
	if o.Language != "" {
		args = append(args, "--language", o.Language)
	}
	for _, b := range []struct {
		flag string
		on   bool
	}{{"--overwrite", o.Overwrite}, {"--sync-terms", o.SyncTerms}, {"--read-from-source", o.ReadFromSource}, {"--fuzzy-trigger", o.FuzzyTrigger}} {
		if b.on {
			args = append(args, b.flag)
		}
	}
	for _, g := range []struct {
		flag string
		tags []string
	}{{"--tag-all", o.Tags.All}, {"--tag-new", o.Tags.New}, {"--tag-obsolete", o.Tags.Obsolete}, {"--tag-overwritten", o.Tags.OverwrittenTranslations}} {
		for _, t := range g.tags {
			args = append(args, g.flag, t)
		}
	}
	return args
}
