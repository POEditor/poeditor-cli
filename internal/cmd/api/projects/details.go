package projects

import (
	"strconv"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// detailsJSON is a project in --json (view, add, update): the API's fields and
// values (public, open 0/1), as our own type so an API change never changes our
// output by accident.
type detailsJSON struct {
	ID                int      `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Public            int      `json:"public"` // 0/1, as the API sends it
	Open              int      `json:"open"`   // 0/1
	ReferenceLanguage string   `json:"reference_language"`
	FallbackLanguage  string   `json:"fallback_language"`
	Terms             int      `json:"terms"`
	TermsLastAdded    api.Time `json:"terms_last_added"`
	TermsLastUpdated  api.Time `json:"terms_last_updated"`
	Created           api.Time `json:"created"`
}

func toDetailsJSON(p api.ProjectDetails) detailsJSON {
	return detailsJSON{
		ID: p.ID, Name: p.Name, Description: p.Description, Public: p.Public, Open: p.Open,
		ReferenceLanguage: p.ReferenceLanguage, FallbackLanguage: p.FallbackLanguage, Terms: p.Terms,
		TermsLastAdded: p.TermsLastAdded, TermsLastUpdated: p.TermsLastUpdated, Created: p.Created,
	}
}

// printDetails shows one project as a one-row table, like `projects list`:
// a header and the row on a terminal (long cells wrap, dates on two lines);
// piped, one tab-separated line.
func printDetails(f *cmdutil.Factory, p api.ProjectDetails) error {
	table := cmdutil.NewTable(f.IO, "ID", "NAME", "DESCRIPTION", "REFERENCE", "FALLBACK", "PUBLIC", "OPEN",
		"TERMS", "T ADDED", "T UPDATED", "CREATED") // T ADDED, T UPDATED: when terms last were
	// dates stay whole (date, time below); the text columns wrap instead
	for _, column := range []int{8, 9, 10} {
		table.SetMinWidth(column, len("2026-01-14"))
	}
	table.AddRow(strconv.Itoa(p.ID), p.Name, p.Description, p.ReferenceLanguage, p.FallbackLanguage,
		yesNo(p.Public), yesNo(p.Open), strconv.Itoa(p.Terms),
		cmdutil.TimeCell(f.IO, p.TermsLastAdded), cmdutil.TimeCell(f.IO, p.TermsLastUpdated), cmdutil.TimeCell(f.IO, p.Created))
	return table.Render()
}
