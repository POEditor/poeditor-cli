package languages

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// NewCmdAvailable returns `poeditor api languages available` (API v2
// languages/available).
func NewCmdAvailable(f *cmdutil.Factory) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "available",
		Short: "List the languages POEditor offers, with their codes",
		Long: `List every language you can add to a project: its code (what --language and
.poeditor.yml take) and its name. Your account's custom languages are
included, so the list depends on --account.

Piped, each language is one tab-separated line; --json gives the list as JSON.`,
		Example: `  poeditor api languages available
  poeditor api languages available | grep -i portuguese
  poeditor api languages available --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := f.APIClient()
			if err != nil {
				return err
			}
			langs, err := client.LanguagesAvailable(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				return cmdutil.PrintJSON(f.IO, availableJSON(langs))
			}
			return printAvailable(f, langs)
		},
	}

	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

func printAvailable(f *cmdutil.Factory, langs []api.LanguageRef) error {
	if len(langs) == 0 {
		if f.IO.IsStdoutTTY() {
			fmt.Fprintln(f.IO.Out, "No languages available.")
		}
		return nil
	}
	table := cmdutil.NewTable(f.IO, "CODE", "NAME")
	for _, l := range langs {
		table.AddRow(l.Code, l.Name)
	}
	return table.Render()
}

// availableLanguageJSON is one language in --json, as our own type so an API change
// never changes our output by accident.
type availableLanguageJSON struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func availableJSON(langs []api.LanguageRef) []availableLanguageJSON {
	out := make([]availableLanguageJSON, len(langs))
	for i, l := range langs {
		out[i] = availableLanguageJSON{Code: l.Code, Name: l.Name}
	}
	return out
}
