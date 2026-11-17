package translations

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// Code shared by add, update and delete: one term's translation from the
// arguments and flags, or many from --from-file, in one --language.

// writeFlags are the flags add and update share.
type writeFlags struct {
	project     string
	language    string
	context     string
	pluralForms []string
	fromFile    string
	asJSON      bool
}

func (w *writeFlags) add(cmd *cobra.Command, verb string) {
	projectflag.Add(cmd, &w.project)
	addLanguageFlag(cmd, &w.language)
	cmd.Flags().StringVar(&w.context, "context", "", "The term's `context` (with its text, it finds the term)")
	cmd.Flags().StringArrayVar(&w.pluralForms, "plural-form", nil, "A plural term's `form=text` (repeat for each form, e.g. one=\"1 file\")")
	cmd.Flags().StringVar(&w.fromFile, "from-file", "", verb+" the translations of a JSON `file` (- for stdin) instead")
	cmdutil.AddJSONFlag(cmd, &w.asJSON)
}

func addLanguageFlag(cmd *cobra.Command, language *string) {
	cmd.Flags().StringVarP(language, "language", "l", "", "The `language` (its code, e.g. fr) (required)")
	_ = cmd.MarkFlagRequired("language")
}

// rows is the list to send: the file's, or the one translation of the
// arguments (<term> <text>, or <term> with --plural-form).
func (w *writeFlags) rows(f *cmdutil.Factory, cmd *cobra.Command, args []string) ([]api.TranslationRow, error) {
	nArgs := 2
	if len(w.pluralForms) > 0 {
		nArgs = 1
	}
	if w.fromFile == "" && len(args) == 2 && len(w.pluralForms) > 0 {
		return nil, errors.New("give the text or --plural-form, not both")
	}
	if err := cmdutil.CheckBulkInput(cmd, args, w.fromFile, nArgs, "a term and its translation (or --plural-form for a plural term)", "context", "plural-form"); err != nil {
		return nil, err
	}
	if w.fromFile != "" {
		return cmdutil.ReadJSONList(f, w.fromFile, checkRow)
	}
	row := api.TranslationRow{TermKey: api.TermKey{Term: args[0], Context: w.context}}
	if len(args) == 2 {
		row.Translation.Content.Text = args[1]
	} else {
		forms, err := pluralForms(w.pluralForms)
		if err != nil {
			return nil, err
		}
		row.Translation.Content.Forms = forms
	}
	if err := checkRow(row); err != nil {
		return nil, fmt.Errorf("the translation: %w", err)
	}
	return []api.TranslationRow{row}, nil
}

// checkRow refuses what translations/add and update would skip or ignore.
func checkRow(r api.TranslationRow) error {
	if err := r.TermKey.Check(); err != nil {
		return err
	}
	if r.Translation.Content.IsEmpty() {
		return errors.New(`no "content" (to remove a translation: translations delete)`)
	}
	if r.Translation.Fuzzy != 0 {
		return errors.New(`"fuzzy": translations add and update ignore it; languages update sets it`)
	}
	return nil
}

// pluralForms reads the --plural-form values, in their order.
func pluralForms(values []string) ([]api.PluralForm, error) {
	var forms []api.PluralForm
	seen := map[string]bool{}
	for _, v := range values {
		form, text, ok := strings.Cut(v, "=")
		form = strings.TrimSpace(form)
		if !ok || form == "" {
			return nil, fmt.Errorf("--plural-form %q: use form=text (e.g. one=\"1 file\")", v)
		}
		if seen[form] {
			return nil, fmt.Errorf("--plural-form %s: given twice", form)
		}
		seen[form] = true
		forms = append(forms, api.PluralForm{Form: form, Text: text})
	}
	return forms, nil
}

// writeResult is what --json prints: our own shape, like push's.
type writeResult struct {
	Project      repoconfig.ProjectRef `json:"project"`
	Language     string                `json:"language"`
	Translations any                   `json:"translations"`
}

// report prints the summary on stderr, and the note when some were left
// out; with --json, the counts on stdout.
func report(f *cmdutil.Factory, p projectflag.Project, language string, asJSON bool, summary, note string, counts any) error {
	fmt.Fprintln(f.IO.ErrOut, summary)
	if note != "" {
		fmt.Fprintln(f.IO.ErrOut, note)
	}
	if asJSON {
		return cmdutil.PrintJSON(f.IO, writeResult{Project: p.Ref, Language: language, Translations: counts})
	}
	return nil
}

// nTranslations is "1 fr translation", "2 fr translations".
func nTranslations(n int, language string) string {
	if n == 1 {
		return "1 " + language + " translation"
	}
	return fmt.Sprintf("%d %s translations", n, language)
}
