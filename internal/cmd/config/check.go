package config

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// NewCmdCheck returns `poeditor config check`.
func NewCmdCheck(f *cmdutil.Factory) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check " + repoconfig.FileName + " against POEditor",
		Long: `Check ` + repoconfig.FileName + ` against POEditor.

First the file itself is validated (as ` + "`config show`" + ` does). Then, for each
project, it checks with the API that the project's account has a token, that
the token can see the project, and that every language the file names
(source_language, languages, language_map, pull.fallback_language) exists in
the project. Tags are not checked: API v2 has no tag list.

Exits with an error if anything is wrong, so it can run in CI.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, issues, err := load(f)
			if err != nil {
				return err
			}

			report := checkReport{File: cfg.Path(), Issues: issues, Projects: []projectReport{}}
			if !repoconfig.HasErrors(issues) {
				for _, p := range cfg.Projects {
					report.Projects = append(report.Projects, checkProject(cmd, f, p))
				}
			}
			report.OK = !repoconfig.HasErrors(report.allIssues())

			if asJSON {
				if report.Issues == nil {
					report.Issues = []repoconfig.Issue{}
				}
				if err := cmdutil.PrintJSON(f.IO, report); err != nil {
					return err
				}
			} else {
				printIssues(f, issues)
				for _, p := range report.Projects {
					status := "ok"
					if repoconfig.HasErrors(p.Issues) {
						status = "FAILED"
					}
					line := fmt.Sprintf("%s  %s", status, p.label)
					if p.Project.Name != "" {
						line += fmt.Sprintf(": %q (%d), account %s, %d languages", p.Project.Name, p.Project.ID, p.Account, len(p.Languages))
					}
					fmt.Fprintln(f.IO.Out, line)
					for _, issue := range p.Issues {
						fmt.Fprintf(f.IO.Out, "    %s: %s\n", issue.Severity, issue.Message)
					}
				}
			}

			if !report.OK {
				return problemsError(cfg, report.allIssues())
			}
			return nil
		},
	}

	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

type checkReport struct {
	OK       bool               `json:"ok"`
	File     string             `json:"file"`
	Issues   []repoconfig.Issue `json:"issues"` // problems in the file itself
	Projects []projectReport    `json:"projects"`
}

func (r checkReport) allIssues() []repoconfig.Issue {
	all := append([]repoconfig.Issue(nil), r.Issues...)
	for _, p := range r.Projects {
		all = append(all, p.Issues...)
	}
	return all
}

type projectReport struct {
	Project   projectRef         `json:"project"`
	Account   string             `json:"account"`
	Languages []string           `json:"languages"` // the project's language codes
	Issues    []repoconfig.Issue `json:"issues"`

	label string
}

// projectRef is how JSON output names a project: its id, the alias from
// .poeditor.yml (if any) and, when the command fetched it, its POEditor name.
type projectRef struct {
	ID    int    `json:"id"`
	Alias string `json:"alias,omitempty"`
	Name  string `json:"name,omitempty"`
}

func checkProject(cmd *cobra.Command, f *cmdutil.Factory, p repoconfig.Project) projectReport {
	rep := projectReport{Project: projectRef{ID: p.ID, Alias: p.Alias}, Languages: []string{}, Issues: []repoconfig.Issue{}, label: p.Label()}
	fail := func(sev repoconfig.Severity, msg string) {
		rep.Issues = append(rep.Issues, repoconfig.Issue{Severity: sev, Where: "project " + p.Label(), Message: msg})
	}

	account, err := f.ResolveFor(p.Account)
	if err != nil {
		fail(repoconfig.Error, err.Error())
		return rep
	}
	rep.Account = account.Name
	if rep.Account == "" {
		rep.Account = "POEDITOR_TOKEN"
	}
	client := f.APIClientWithToken(account.Token)
	params := url.Values{"id": {strconv.Itoa(p.ID)}}

	var view struct {
		Project struct {
			Name string `json:"name"`
		} `json:"project"`
	}
	if err := client.Post(cmd.Context(), "projects/view", params, &view); err != nil {
		fail(repoconfig.Error, fmt.Sprintf("cannot open project %d as %s: %v", p.ID, rep.Account, err))
		return rep
	}
	rep.Project.Name = view.Project.Name

	var langs struct {
		Languages []struct {
			Code string `json:"code"`
		} `json:"languages"`
	}
	if err := client.Post(cmd.Context(), "languages/list", params, &langs); err != nil {
		fail(repoconfig.Error, fmt.Sprintf("cannot list the languages of project %d: %v", p.ID, err))
		return rep
	}
	have := map[string]bool{}
	for _, l := range langs.Languages {
		have[l.Code] = true
		rep.Languages = append(rep.Languages, l.Code)
	}
	sort.Strings(rep.Languages)

	missing := func(code, field string, sev repoconfig.Severity) {
		if code != "" && !have[code] {
			fail(sev, fmt.Sprintf("%s: language %q is not in the project (it has: %s)", field, code, strings.Join(rep.Languages, ", ")))
		}
	}
	missing(p.SourceLanguage, "source_language", repoconfig.Error)
	for i, file := range p.Files {
		prefix := fmt.Sprintf("file %d (%s) ", i+1, file.Path)
		for _, code := range file.Languages {
			missing(code, prefix+"languages", repoconfig.Error)
		}
		codes := make([]string, 0, len(file.LanguageMap))
		for code := range file.LanguageMap {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		for _, code := range codes {
			missing(code, prefix+"language_map", repoconfig.Warning)
		}
		if file.Pull != nil {
			missing(file.Pull.FallbackLanguage, prefix+"pull.fallback_language", repoconfig.Error)
		}
	}
	return rep
}
