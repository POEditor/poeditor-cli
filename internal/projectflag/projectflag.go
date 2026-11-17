// Package projectflag is the --project flag of project-scoped commands
// (terms, languages, translations…): which project to act on, and with which
// account. It is required — there is no default project, so a command never
// acts on the wrong one by accident.
package projectflag

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// Add registers the required --project (-p) flag, and --id, a synonym with
// the API documentation's name, so a parameter copied from the docs works
// too. --id has no row of its own in --help: the --project line names it.
// Resolve checks that one of them was given: cobra's own required check
// cannot know they are the same.
func Add(cmd *cobra.Command, value *string) {
	cmd.Flags().StringVarP(value, "project", "p", "", "The project: its alias in .poeditor.yml, or its id (required; --id works too)")
	cmd.Flags().StringVar(value, "id", "", "Same as --project (the API's name for it)")
	_ = cmd.Flags().MarkHidden("id")
	cmd.MarkFlagsMutuallyExclusive("project", "id")
}

// ErrMissing is returned by Resolve when neither --project nor --id was given.
var ErrMissing = errors.New(`required flag "--project" not set (the project's alias in .poeditor.yml, or its id)`)

// Project is the project a command acts on.
type Project struct {
	ID      int
	Label   string // the alias, else the id: how messages name it
	Account string // the account its requests use
	Client  *api.Client
	Ref     repoconfig.ProjectRef // how --json names it: {id, alias?}
}

// Resolve turns --project into a project and its client.
//
//   - An alias or id listed in .poeditor.yml (looked up from the working
//     directory) is that project, with the account rule of pull and push:
//     --account, else the project's `account:`, else the default account.
//   - Any other id works without the file, with --account or the default
//     account. A file that can't be read is then only a warning on stderr;
//     an alias needs it, so for an alias it is the error.
func Resolve(f *cmdutil.Factory, value string) (Project, error) {
	if value == "" {
		return Project{}, ErrMissing
	}
	id, isID := parseID(value)
	cfg, err := loadConfig(f)
	switch {
	case err != nil && isID:
		// an id does not need the file: a broken one (maybe in a parent
		// directory) must not block it
		fmt.Fprintf(f.IO.ErrOut, "warning: %v\n(ignored: --project %s is an id)\n", err, value)
		cfg = nil
	case err != nil:
		return Project{}, err
	}

	p := repoconfig.Project{ID: id}
	switch {
	case cfg != nil:
		found, err := cfg.Select(value)
		if err == nil {
			p = found[0]
		} else if !isID {
			return Project{}, fmt.Errorf("--project: %w", err)
		}
	case !isID:
		return Project{}, fmt.Errorf("--project %q: not a project id, and there is no %s here to look up an alias (ids: `poeditor api projects list`)", value, repoconfig.FileName)
	}

	account, err := f.ResolveFor(p.Account)
	if err != nil {
		return Project{}, err
	}
	return Project{ID: p.ID, Label: p.Label(), Account: account.Name, Client: f.APIClientWithToken(account.Token), Ref: p.Ref()}, nil
}

// loadConfig returns .poeditor.yml, or nil when there is none.
func loadConfig(f *cmdutil.Factory) (*repoconfig.Config, error) {
	dir, err := f.WorkingDir()
	if err != nil {
		return nil, err
	}
	path, err := repoconfig.Find(dir)
	if errors.Is(err, repoconfig.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return repoconfig.Load(path)
}

func parseID(value string) (int, bool) {
	id, err := strconv.Atoi(value)
	return id, err == nil && id > 0
}

// LanguageError explains an API refusal of a language code — subject names
// it as the user gave it ("--language xx", or "xx" for an argument). Any
// other error is returned as it is. Only then are the project's languages
// fetched, so a correct call costs one request.
func (p Project) LanguageError(ctx context.Context, subject string, err error) error {
	var apiErr *api.Error
	if !errors.As(err, &apiErr) {
		return err
	}
	switch apiErr.Code {
	case api.CodeUnknownLanguage:
		return fmt.Errorf("%s: not a language code POEditor knows (e.g. en, fr, pt-br; all of them: `poeditor api languages available`)", subject)
	case api.CodeLanguageNotInProject:
		codes, listErr := p.Client.LanguageCodes(ctx, p.ID)
		if listErr != nil || len(codes) == 0 {
			return fmt.Errorf("%s: project %s has no such language", subject, p.Label)
		}
		return fmt.Errorf("%s: project %s has no such language (it has: %s)", subject, p.Label, strings.Join(codes, ", "))
	}
	return err
}
