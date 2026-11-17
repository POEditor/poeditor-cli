package terms

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdList returns `poeditor api terms list` (API v2 terms/list).
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	var (
		project  string
		language string
		asJSON   bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List a project's terms, with their translations in a language",
		Long: `List every term of a project, with all its fields. With --language, each
term's translation in that language is added: on a terminal its status
(fuzzy, proofread, last update) on one line and the text below it, one line
per plural form.

On a terminal, long texts wrap within their column. Piped, each term is one
tab-separated line (a plural translation as a JSON object); --json gives
everything, in the API's shape.`,
		Example: `  poeditor api terms list --project web
  poeditor api terms list -p 7717 --language fr
  poeditor api terms list -p web -l fr --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			terms, err := p.Client.TermsList(cmd.Context(), p.ID, language)
			if err != nil {
				return p.LanguageError(cmd.Context(), "--language "+language, err)
			}
			if asJSON {
				return cmdutil.PrintJSON(f.IO, termsJSON(terms))
			}
			return printTerms(f, p, terms, language != "")
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().StringVarP(&language, "language", "l", "", "Add each term's translation in this `language` (its code, e.g. fr)")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

func printTerms(f *cmdutil.Factory, p projectflag.Project, terms []api.Term, withTranslation bool) error {
	if len(terms) == 0 {
		if f.IO.IsStdoutTTY() {
			fmt.Fprintf(f.IO.Out, "Project %s has no terms.\n", p.Label)
		}
		return nil
	}

	header := []string{"TERM", "CONTEXT"}
	if withTranslation {
		header = append(header, "TRANSLATION")
	}
	header = append(header, "PLURAL", "REFERENCE", "TAGS", "COMMENT", "CREATED", "UPDATED")
	table := cmdutil.NewTable(f.IO, header...)
	if withTranslation {
		// the status line stays whole; the other columns wrap more instead
		widest := 0
		for _, t := range terms {
			if t.Translation != nil && !t.Translation.IsEmpty() {
				widest = max(widest, len(statusLine(t.Translation))) // ASCII only
			}
		}
		table.SetMinWidth(2, widest)
	}
	for _, t := range terms {
		row := []string{t.Term, t.Context}
		if withTranslation {
			row = append(row, translationCell(f.IO, t.Translation))
		}
		row = append(row, t.Plural, t.Reference, cmdutil.ListCell(f.IO, t.Tags), t.Comment,
			cmdutil.TimeCell(f.IO, t.Created), cmdutil.TimeCell(f.IO, t.Updated))
		table.AddRow(row...)
	}
	return table.Render()
}

// translationCell is a translation as a table cell; empty when the term is
// not translated.
//   - terminal: the status on the first line, the text below it (a plural
//     translation one "form: text" line per form):
//     FUZZY: 0  PROOFREAD: 1  UPDATED: 2026-03-01 08:00:00
//     Salvează
//   - piped: the text only (a plural one as its JSON object); the status is
//     in --json.
func translationCell(ios *iostreams.IOStreams, tr *api.Translation) string {
	if tr == nil || tr.IsEmpty() {
		return ""
	}
	if !ios.IsStdoutTTY() {
		if tr.Content.IsPlural() {
			text, _ := tr.Content.MarshalJSON() // always encodes
			return string(text)
		}
		return tr.Content.Text
	}

	lines := []string{statusLine(tr)}
	if tr.Content.IsPlural() {
		for _, form := range tr.Content.Forms {
			lines = append(lines, form.Form+": "+form.Text)
		}
	} else {
		lines = append(lines, tr.Content.Text)
	}
	return strings.Join(lines, "\n")
}

// statusLine is a translation's state on one line:
// FUZZY: 0  PROOFREAD: 1  UPDATED: 2026-03-01 08:00:00 (PROOFREAD left out
// when proofreading is off).
func statusLine(tr *api.Translation) string {
	status := []string{"FUZZY: " + strconv.Itoa(tr.Fuzzy)}
	if tr.Proofread != nil {
		status = append(status, "PROOFREAD: "+strconv.Itoa(*tr.Proofread))
	}
	if !tr.Updated.IsZero() {
		status = append(status, "UPDATED: "+cmdutil.TimeLine(tr.Updated))
	}
	return strings.Join(status, "  ")
}

// termJSON is one term in --json: the API's fields, as our own type so an
// API change never changes our output by accident.
type termJSON struct {
	Term        string           `json:"term"`
	Context     string           `json:"context"`
	Plural      string           `json:"plural"`
	Reference   string           `json:"reference"`
	Comment     string           `json:"comment"`
	Tags        []string         `json:"tags"`
	Created     api.Time         `json:"created"`
	Updated     api.Time         `json:"updated"`
	Translation *translationJSON `json:"translation,omitempty"`
}

type translationJSON struct {
	Content   api.Content `json:"content"`
	Fuzzy     int         `json:"fuzzy"`
	Proofread *int        `json:"proofread,omitempty"` // left out when proofreading is off
	Updated   api.Time    `json:"updated"`
}

func termsJSON(terms []api.Term) []termJSON {
	out := make([]termJSON, len(terms))
	for i, t := range terms {
		tags := t.Tags
		if tags == nil {
			tags = []string{}
		}
		out[i] = termJSON{
			Term: t.Term, Context: t.Context, Plural: t.Plural, Reference: t.Reference,
			Comment: t.Comment, Tags: tags, Created: t.Created, Updated: t.Updated,
		}
		if tr := t.Translation; tr != nil {
			out[i].Translation = &translationJSON{Content: tr.Content, Fuzzy: tr.Fuzzy, Proofread: tr.Proofread, Updated: tr.Updated}
		}
	}
	return out
}
