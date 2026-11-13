package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// NewCmdList returns `poeditor auth list`.
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your accounts, the default one and where each token comes from",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, err := f.Resolver()
			if err != nil {
				return err
			}
			cfg := r.Config

			if asJSON {
				out := []accountJSON{}
				for _, name := range cfg.AccountNames() {
					_, source := r.Token(name)
					out = append(out, accountJSON{Name: name, Default: name == cfg.DefaultAccount, Token: tokenSource(source)})
				}
				return cmdutil.PrintJSON(f.IO, out)
			}

			if len(cfg.Accounts) == 0 {
				if !f.IO.IsStdoutTTY() {
					return nil
				}
				if f.Getenv("POEDITOR_TOKEN") != "" {
					fmt.Fprintln(f.IO.Out, "No accounts; commands use POEDITOR_TOKEN from the environment.")
				} else {
					fmt.Fprintln(f.IO.Out, "No accounts. Run `poeditor auth login`.")
				}
				return nil
			}

			table := cmdutil.NewTable(f.IO, "ACCOUNT", "DEFAULT", "TOKEN")
			for _, name := range cfg.AccountNames() {
				def := ""
				if name == cfg.DefaultAccount {
					def = "yes"
				}
				_, source := r.Token(name)
				table.AddRow(name, def, tokenSource(source))
			}
			return table.Render()
		},
	}

	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

type accountJSON struct {
	Name    string `json:"name"`
	Default bool   `json:"default"`
	Token   string `json:"token"` // where the token comes from
}

func tokenSource(source string) string {
	if source == "" {
		return "missing"
	}
	return source
}
