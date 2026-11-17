package projects

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// detailsJSON is a project in --json (view, add, update): the API's fields,
// as our own type so an API change never changes our output by accident.
type detailsJSON struct {
	ID                int      `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Public            bool     `json:"public"`
	Open              bool     `json:"open"`
	ReferenceLanguage string   `json:"reference_language"`
	FallbackLanguage  string   `json:"fallback_language"`
	Terms             int      `json:"terms"`
	TermsLastAdded    api.Time `json:"terms_last_added"`
	TermsLastUpdated  api.Time `json:"terms_last_updated"`
	Created           api.Time `json:"created"`
}

func toDetailsJSON(p api.ProjectDetails) detailsJSON {
	return detailsJSON{
		ID: p.ID, Name: p.Name, Description: p.Description, Public: p.Public != 0, Open: p.Open != 0,
		ReferenceLanguage: p.ReferenceLanguage, FallbackLanguage: p.FallbackLanguage, Terms: p.Terms,
		TermsLastAdded: p.TermsLastAdded, TermsLastUpdated: p.TermsLastUpdated, Created: p.Created,
	}
}

// printDetails shows one project, a field per line: "Label: value" on a
// terminal; piped, "key<TAB>value" with the API's keys and date format.
func printDetails(f *cmdutil.Factory, p api.ProjectDetails) error {
	tty := f.IO.IsStdoutTTY()
	date := func(t api.Time) string {
		if tty {
			return orNone(cmdutil.TimeLine(t))
		}
		if t.IsZero() {
			return ""
		}
		return t.Format(api.TimeLayout)
	}
	text := func(s string) string {
		if tty {
			return orNone(s)
		}
		return s
	}
	fields := []struct{ label, key, value string }{
		{"ID", "id", strconv.Itoa(p.ID)},
		{"Name", "name", p.Name},
		{"Description", "description", text(p.Description)},
		{"Reference language", "reference_language", text(p.ReferenceLanguage)},
		{"Fallback language", "fallback_language", text(p.FallbackLanguage)},
		{"Public", "public", yesNo(p.Public)},
		{"Open", "open", yesNo(p.Open)},
		{"Terms", "terms", strconv.Itoa(p.Terms)},
		{"Terms last added", "terms_last_added", date(p.TermsLastAdded)},
		{"Terms last updated", "terms_last_updated", date(p.TermsLastUpdated)},
		{"Created", "created", date(p.Created)},
	}
	if !tty {
		// the table's piped form: key<TAB>value, tabs and newlines in values escaped
		table := cmdutil.NewTable(f.IO, "KEY", "VALUE")
		for _, fl := range fields {
			table.AddRow(fl.key, fl.value)
		}
		return table.Render()
	}
	for _, fl := range fields {
		// a value of several lines (a description) stays in its column
		value := strings.ReplaceAll(fl.value, "\n", "\n"+strings.Repeat(" ", 20))
		if _, err := fmt.Fprintf(f.IO.Out, "%-20s%s\n", fl.label+":", value); err != nil {
			return err
		}
	}
	return nil
}

func orNone(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
