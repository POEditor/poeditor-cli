// Package transfer is what pull and push share: loading .poeditor.yml,
// deciding which project × file × language to transfer, and the JSON shape
// of the per-file results.
package transfer

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// LoadConfig finds, loads and validates .poeditor.yml. Warnings are printed
// on stderr; any error stops the command before anything is transferred.
func LoadConfig(f *cmdutil.Factory) (*repoconfig.Config, error) {
	dir, err := f.WorkingDir()
	if err != nil {
		return nil, err
	}
	path, err := repoconfig.Find(dir)
	if err != nil {
		return nil, err
	}
	cfg, err := repoconfig.Load(path)
	if err != nil {
		return nil, err
	}
	issues := cfg.Validate()
	for _, issue := range issues {
		fmt.Fprintln(f.IO.ErrOut, issue.String())
	}
	if repoconfig.HasErrors(issues) {
		return nil, fmt.Errorf("fix %s first (see `poeditor config show`)", path)
	}
	return cfg, nil
}

// ProjectRef names a project in JSON output.
type ProjectRef struct {
	ID    int    `json:"id"`
	Alias string `json:"alias,omitempty"`
}

// Ref returns the JSON reference of a project.
func Ref(p repoconfig.Project) ProjectRef { return ProjectRef{ID: p.ID, Alias: p.Alias} }

// Item is one file in one language of one project.
type Item struct {
	Project  repoconfig.Project
	File     repoconfig.File
	Language string
	Path     string // on disk
	Display  string // relative to the config dir, with "/", for messages and JSON
}

// Client returns the API client for a project (--account, then the
// project's account, then the default account).
func Client(f *cmdutil.Factory, p repoconfig.Project) (*api.Client, error) {
	account, err := f.ResolveFor(p.Account)
	if err != nil {
		return nil, fmt.Errorf("project %s: %w", p.Label(), err)
	}
	return f.APIClientWithToken(account.Token), nil
}

// ProjectLanguages lists the project's language codes, sorted.
func ProjectLanguages(ctx context.Context, client *api.Client, projectID int) ([]string, error) {
	var result struct {
		Languages []struct {
			Code string `json:"code"`
		} `json:"languages"`
	}
	if err := client.Post(ctx, "languages/list", url.Values{"id": {strconv.Itoa(projectID)}}, &result); err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(result.Languages))
	for _, l := range result.Languages {
		codes = append(codes, l.Code)
	}
	sort.Strings(codes)
	return codes, nil
}

// Items expands the files of a project into one Item per language: the
// file's `languages:` if set, else every project language, narrowed to
// only (the --language flags) when given.
func Items(cfg *repoconfig.Config, p repoconfig.Project, projectLangs, only []string) ([]Item, error) {
	for _, code := range only {
		if !slices.Contains(projectLangs, code) {
			return nil, fmt.Errorf("--language %s: project %s has no such language (it has: %s)", code, p.Label(), strings.Join(projectLangs, ", "))
		}
	}
	var items []Item
	for _, file := range p.Files {
		langs := file.Languages
		if len(langs) == 0 {
			langs = projectLangs
		}
		for _, lang := range langs {
			if len(only) > 0 && !slices.Contains(only, lang) {
				continue
			}
			path := file.LocalPath(cfg.Dir(), lang)
			display, err := filepath.Rel(cfg.Dir(), path)
			if err != nil {
				display = path
			}
			items = append(items, Item{Project: p, File: file, Language: lang, Path: path, Display: filepath.ToSlash(display)})
		}
	}
	return items, nil
}

// JSONList encodes a list for an API parameter (filters, tags).
func JSONList(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = strconv.Quote(v)
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

// ErrStopped wraps the first failure of a pull or push.
type ErrStopped struct {
	Display string
	Err     error
}

func (e *ErrStopped) Error() string { return e.Display + ": " + e.Err.Error() }
func (e *ErrStopped) Unwrap() error { return e.Err }

// IsStopped reports whether err is a per-file failure.
func IsStopped(err error) bool {
	var stopped *ErrStopped
	return errors.As(err, &stopped)
}
