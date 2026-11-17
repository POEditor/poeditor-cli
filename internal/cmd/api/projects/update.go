package projects

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
)

// NewCmdUpdate returns `poeditor api projects update` (API v2
// projects/update).
func NewCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	var (
		project       string
		name          string
		description   string
		reference     string
		fallback      string
		proofreading  string
		lockProofread string
		asJSON        bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Change a project's name, description, languages or proofreading",
		Long: `Change a project's settings. Only the flags given change; --reference-language ""
or --fallback-language "" clears that language. A language must be one of the
project's (checked first: POEditor would ignore a wrong code silently).

--proofreading on|off turns proofreading on or off; --lock-proofread on|off
keeps contributors from changing proofread translations.

--json prints the project as it is after the change.`,
		Example: `  poeditor api projects update --project web --name "Web (2026)"
  poeditor api projects update -p 7717 --reference-language en --fallback-language ""
  poeditor api projects update -p web --proofreading on --lock-proofread on`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			u, err := projectUpdate(cmd, name, description, reference, fallback, proofreading, lockProofread)
			if err != nil {
				return err
			}
			p, err := projectflag.Resolve(f, project)
			if err != nil {
				return err
			}
			if err := checkLanguages(cmd.Context(), p, u); err != nil {
				return err
			}
			details, err := p.Client.ProjectsUpdate(cmd.Context(), p.ID, u)
			if err != nil {
				return err
			}
			fmt.Fprintf(f.IO.ErrOut, "Updated project %s.\n", p.Label)
			if asJSON {
				return cmdutil.PrintJSON(f.IO, toDetailsJSON(details))
			}
			return nil
		},
	}

	projectflag.Add(cmd, &project)
	cmd.Flags().StringVar(&name, "name", "", "The project's new `name`")
	cmd.Flags().StringVar(&description, "description", "", "The project's `description` (\"\" clears it)")
	cmd.Flags().StringVar(&reference, "reference-language", "", "The reference `language`: one of the project's (\"\" clears it)")
	cmd.Flags().StringVar(&fallback, "fallback-language", "", "The fallback `language` of exports (\"\" clears it)")
	cmd.Flags().StringVar(&proofreading, "proofreading", "", "Turn proofreading `on|off`")
	cmd.Flags().StringVar(&lockProofread, "lock-proofread", "", "Keep contributors from changing proofread translations: `on|off`")
	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

// projectUpdate maps the flags given onto the API's update.
func projectUpdate(cmd *cobra.Command, name, description, reference, fallback, proofreading, lockProofread string) (api.ProjectUpdate, error) {
	var u api.ProjectUpdate
	str := func(flag string, v *string) *string {
		if cmd.Flags().Changed(flag) {
			s := strings.TrimSpace(*v)
			return &s
		}
		return nil
	}
	onOff := func(flag, v string) (*bool, error) {
		if !cmd.Flags().Changed(flag) {
			return nil, nil
		}
		switch v {
		case "on":
			on := true
			return &on, nil
		case "off":
			off := false
			return &off, nil
		}
		return nil, fmt.Errorf("--%s %q: use on or off", flag, v)
	}
	u.Name = str("name", &name)
	u.Description = str("description", &description)
	u.ReferenceLanguage = str("reference-language", &reference)
	u.FallbackLanguage = str("fallback-language", &fallback)
	var err error
	if u.Proofreading, err = onOff("proofreading", proofreading); err != nil {
		return u, err
	}
	if u.LockProofread, err = onOff("lock-proofread", lockProofread); err != nil {
		return u, err
	}
	if u.Name != nil && *u.Name == "" {
		return u, errors.New("--name is empty")
	}
	if u.IsZero() {
		return u, errors.New("nothing to change: give --name, --description, --reference-language, --fallback-language, --proofreading or --lock-proofread")
	}
	return u, nil
}

// checkLanguages refuses a language the project does not have: the server
// would ignore it and report success.
func checkLanguages(ctx context.Context, p projectflag.Project, u api.ProjectUpdate) error {
	given := []struct {
		flag string
		code *string
	}{{"--reference-language", u.ReferenceLanguage}, {"--fallback-language", u.FallbackLanguage}}
	given = slices.DeleteFunc(given, func(g struct {
		flag string
		code *string
	}) bool {
		return g.code == nil || *g.code == ""
	})
	if len(given) == 0 {
		return nil
	}
	codes, err := p.Client.LanguageCodes(ctx, p.ID)
	if err != nil {
		return err
	}
	for _, g := range given {
		if !slices.Contains(codes, *g.code) {
			if len(codes) == 0 {
				return fmt.Errorf("%s %s: project %s has no languages yet (add them with `poeditor api languages add`)", g.flag, *g.code, p.Label)
			}
			return fmt.Errorf("%s %s: project %s has no such language (it has: %s)", g.flag, *g.code, p.Label, strings.Join(codes, ", "))
		}
	}
	return nil
}
