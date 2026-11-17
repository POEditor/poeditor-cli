package contributors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdAdd returns `poeditor api contributors add` (API v2
// contributors/add).
func NewCmdAdd(f *cmdutil.Factory) *cobra.Command {
	var (
		project string
		n       api.NewContributor
	)

	cmd := &cobra.Command{
		Use:   "add <email>",
		Short: "Add a contributor to a project language, or an administrator",
		Long: `Give someone access to a project: a language to translate (--language), as a
proofreader too with --proofreader (the project needs proofreading on), or the
whole project as an administrator (--admin; only the owner can add one).

POEditor invites a person who has no account yet. Someone who already
translates the language becomes a proofreader with --proofreader.`,
		Example: `  poeditor api contributors add ana@example.com --name Ana --project web --language fr
  poeditor api contributors add ana@example.com --name Ana -p 7717 -l fr --proofreader
  poeditor api contributors add bo@example.com --name Bo -p web --admin`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n.Email = args[0]
			if err := checkNew(n); err != nil {
				return err
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			if err := p.Client.ContributorsAdd(cmd.Context(), p.ID, n); err != nil {
				if n.Language != "" {
					return p.LanguageError(cmd.Context(), "--language "+n.Language, err)
				}
				return err
			}
			fmt.Fprintf(f.IO.ErrOut, "Added %s to project %s as %s.\n", n.Email, p.Label, describe(n))
			return nil
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().StringVar(&n.Name, "name", "", "The person's `name` (required)")
	cmd.Flags().StringVarP(&n.Language, "language", "l", "", "The `language` to translate (its code, e.g. fr)")
	cmd.Flags().BoolVar(&n.Proofreader, "proofreader", false, "A proofreader of the language too")
	cmd.Flags().BoolVar(&n.Admin, "admin", false, "An administrator of the project, instead of one language")
	_ = cmd.MarkFlagRequired("name")
	cmd.MarkFlagsMutuallyExclusive("language", "admin")
	cmd.MarkFlagsMutuallyExclusive("proofreader", "admin")
	cmdutil.MarkAction(cmd)
	return cmd
}

// checkNew checks the flags before any API call.
func checkNew(n api.NewContributor) error {
	if !strings.Contains(n.Email, "@") {
		return fmt.Errorf("%q is not an email address", n.Email)
	}
	if strings.TrimSpace(n.Name) == "" {
		return errors.New("--name is empty")
	}
	if !n.Admin && n.Language == "" {
		return errors.New("give --language (a contributor of that language) or --admin (an administrator)")
	}
	return nil
}

// describe is the role given, for the message: "fr contributor",
// "fr proofreader", "administrator".
func describe(n api.NewContributor) string {
	switch {
	case n.Admin:
		return "administrator"
	case n.Proofreader:
		return n.Language + " proofreader"
	}
	return n.Language + " contributor"
}
