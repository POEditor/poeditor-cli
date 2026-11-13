package auth

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// NewCmdList returns `poeditor auth list`.
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your accounts, the default one and where each token comes from",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, err := f.Resolver()
			if err != nil {
				return err
			}
			cfg := r.Config

			if len(cfg.Accounts) == 0 {
				if f.Getenv("POEDITOR_TOKEN") != "" {
					fmt.Fprintln(f.IO.Out, "No accounts; commands use POEDITOR_TOKEN from the environment.")
				} else {
					fmt.Fprintln(f.IO.Out, "No accounts. Run `poeditor auth login`.")
				}
				return nil
			}

			w := tabwriter.NewWriter(f.IO.Out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ACCOUNT\tDEFAULT\tTOKEN")
			for _, name := range cfg.AccountNames() {
				def := ""
				if name == cfg.DefaultAccount {
					def = "yes"
				}
				_, source := r.Token(name)
				if source == "" {
					source = "missing"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", name, def, source)
			}
			return w.Flush()
		},
	}
}
