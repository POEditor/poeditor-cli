package config

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/prompt"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// wizard holds what the init/add steps share.
type wizard struct {
	cmd *cobra.Command
	f   *cmdutil.Factory
	ask prompt.Prompter
	dir string // where .poeditor.yml lives; file paths are relative to it
}

func newWizard(cmd *cobra.Command, f *cmdutil.Factory, dir string) (*wizard, error) {
	ask, err := f.Prompt()
	if err != nil {
		return nil, err
	}
	return &wizard{cmd: cmd, f: f, ask: ask, dir: dir}, nil
}

func (w *wizard) say(format string, args ...any) {
	fmt.Fprintf(w.f.IO.ErrOut, format+"\n", args...)
}

// chosenAccount is the account a new project uses, and what to write as its
// `account:` ("" when the user has a single account: it is simply the default).
type chosenAccount struct {
	resolved accounts.Resolved
	write    string
}

func (w *wizard) chooseAccount() (chosenAccount, error) {
	r, err := w.f.Resolver()
	if err != nil {
		return chosenAccount{}, err
	}
	names := r.Config.AccountNames()

	if w.f.Account != "" {
		res, err := r.Resolve(w.f.Account)
		if err != nil {
			return chosenAccount{}, err
		}
		write := ""
		if len(names) > 1 {
			write = res.Name
		}
		return chosenAccount{resolved: res, write: write}, nil
	}

	switch len(names) {
	case 0:
		res, err := r.Resolve("")
		if errors.Is(err, accounts.ErrNotLoggedIn) {
			return chosenAccount{}, errors.New("log in first: run `poeditor auth login`")
		}
		return chosenAccount{resolved: res}, err
	case 1:
		res, err := r.Resolve(names[0])
		if err == nil {
			w.say("Account: %s", names[0])
		}
		return chosenAccount{resolved: res}, err
	}

	def := slices.Index(names, r.Config.DefaultAccount)
	labels := make([]string, len(names))
	for i, name := range names {
		labels[i] = name
		if name == r.Config.DefaultAccount {
			labels[i] += " (default)"
		}
	}
	i, err := w.ask.Select("Which account is the project in?", labels, max(def, 0))
	if err != nil {
		return chosenAccount{}, err
	}
	res, err := r.Resolve(names[i])
	return chosenAccount{resolved: res, write: names[i]}, err
}

type remoteProject struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (w *wizard) chooseProject(client *api.Client, exclude map[int]bool) (remoteProject, error) {
	var result struct {
		Projects []remoteProject `json:"projects"`
	}
	if err := client.Post(w.cmd.Context(), "projects/list", nil, &result); err != nil {
		return remoteProject{}, err
	}
	var choices []remoteProject
	for _, p := range result.Projects {
		if !exclude[p.ID] {
			choices = append(choices, p)
		}
	}
	if len(choices) == 0 {
		if len(result.Projects) > 0 {
			return remoteProject{}, errors.New("every project of this account is already in " + repoconfig.FileName)
		}
		return remoteProject{}, errors.New("this account has no projects: create one on poeditor.com first")
	}
	sort.Slice(choices, func(i, j int) bool { return strings.ToLower(choices[i].Name) < strings.ToLower(choices[j].Name) })
	labels := make([]string, len(choices))
	for i, p := range choices {
		labels[i] = fmt.Sprintf("%s (%d)", p.Name, p.ID)
	}
	i, err := w.ask.Select("Which project?", labels, 0)
	if err != nil {
		return remoteProject{}, err
	}
	return choices[i], nil
}

// projectLanguages returns the project's language codes and its default
// (reference) language, if set.
func (w *wizard) projectLanguages(client *api.Client, id int) ([]string, string, error) {
	params := url.Values{"id": {strconv.Itoa(id)}}
	var view struct {
		Project struct {
			ReferenceLanguage string `json:"reference_language"`
		} `json:"project"`
	}
	if err := client.Post(w.cmd.Context(), "projects/view", params, &view); err != nil {
		return nil, "", err
	}
	var langs struct {
		Languages []struct {
			Code string `json:"code"`
		} `json:"languages"`
	}
	if err := client.Post(w.cmd.Context(), "languages/list", params, &langs); err != nil {
		return nil, "", err
	}
	codes := make([]string, 0, len(langs.Languages))
	for _, l := range langs.Languages {
		codes = append(codes, l.Code)
	}
	sort.Strings(codes)
	return codes, view.Project.ReferenceLanguage, nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(name string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(s) > 40 {
		s = strings.TrimRight(s[:40], "-")
	}
	if s != "" && allDigitsRe.MatchString(s) {
		s = "project-" + s
	}
	return s
}

var allDigitsRe = regexp.MustCompile(`^[0-9]+$`)

func (w *wizard) askName(projectName string, taken map[string]bool) (string, error) {
	return w.ask.Input("Short name for --project (empty for none)", slug(projectName), func(s string) error {
		switch {
		case s == "":
			return nil
		case allDigitsRe.MatchString(s):
			return errors.New("must not be all digits: it would be mistaken for a project id")
		case taken[s]:
			return fmt.Errorf("%q is already used by another project", s)
		case strings.ContainsAny(s, " \t"):
			return errors.New("no spaces")
		}
		return nil
	})
}

func (w *wizard) askSourceLanguage(codes []string, reference string) (string, error) {
	if len(codes) == 0 {
		return "", errors.New("the project has no languages: add one on poeditor.com first")
	}
	def := slices.Index(codes, reference)
	if def < 0 {
		def = max(slices.Index(codes, "en"), 0)
	}
	i, err := w.ask.Select("Source language (the one `push` uploads terms from)", codes, def)
	if err != nil {
		return "", err
	}
	return codes[i], nil
}

const typePath = "Type a path…"

// askFile asks for one file: path (from what is in the repo, or typed),
// format, pull tags and push tags.
func (w *wizard) askFile(codes []string, takenPaths map[string]bool) (repoconfig.File, error) {
	var file repoconfig.File

	var candidates []repoconfig.Candidate
	for _, c := range repoconfig.Detect(w.dir, codes) {
		if !takenPaths[c.Pattern] {
			candidates = append(candidates, c)
		}
	}

	pathChoice := -1
	if len(candidates) > 0 {
		labels := make([]string, 0, len(candidates)+1)
		for _, c := range candidates {
			labels = append(labels, fmt.Sprintf("%s  (found: %s)", c.Pattern, strings.Join(c.Found, ", ")))
		}
		labels = append(labels, typePath)
		i, err := w.ask.Select("Which file? (found in this directory)", labels, 0)
		if err != nil {
			return file, err
		}
		if i < len(candidates) {
			pathChoice = i
		}
	}

	if pathChoice >= 0 {
		c := candidates[pathChoice]
		file.Path, file.LanguageMap = c.Pattern, c.LanguageMap
	} else {
		p, err := w.ask.Input("File path, with {lang} where the language code goes", "", func(s string) error {
			if err := repoconfig.ValidatePath(s, false); err != nil {
				return err
			}
			if takenPaths[s] {
				return errors.New("this path is already in " + repoconfig.FileName)
			}
			return nil
		})
		if err != nil {
			return file, err
		}
		file.Path = p
	}

	format, err := w.askFormat(repoconfig.GuessFormat(file.Path))
	if err != nil {
		return file, err
	}
	file.Format = format

	pullTags, err := w.askTags("Only pull terms with these tags? (comma-separated; empty for all terms)")
	if err != nil {
		return file, err
	}
	if len(pullTags) > 0 {
		modes := []string{
			"all: terms that have every one of these tags",
			"any: terms that have at least one of them",
			"exclude: terms that have none of them",
		}
		i, err := w.ask.Select("Which terms should pull export?", modes, 0)
		if err != nil {
			return file, err
		}
		file.Pull = &repoconfig.Pull{Tags: pullTags, TagsMatch: repoconfig.TagsMatchModes[i]}
	}

	tags, err := w.askPushTags()
	if err != nil {
		return file, err
	}
	if tags != nil {
		file.Push = &repoconfig.Push{Tags: tags}
	}
	return file, nil
}

// askPushTags asks for the tags push puts on every term, then, only if the
// user wants them, the tags for new, obsolete and overwritten terms.
// It returns nil when every group is empty.
func (w *wizard) askPushTags() (*repoconfig.PushTags, error) {
	const list = " (comma-separated; empty for none)"
	tags := &repoconfig.PushTags{}
	var err error
	if tags.All, err = w.askTags("Tag every pushed term with?" + list); err != nil {
		return nil, err
	}
	more, err := w.ask.Confirm("Also tag new, obsolete or overwritten terms on push?", false)
	if err != nil {
		return nil, err
	}
	if more {
		if tags.New, err = w.askTags("Tag terms that push creates (new terms)?" + list); err != nil {
			return nil, err
		}
		if tags.Obsolete, err = w.askTags("Tag terms that are in POEditor but no longer in the file (obsolete)?" + list); err != nil {
			return nil, err
		}
		if tags.OverwrittenTranslations, err = w.askTags("Tag terms whose translations push overwrites?" + list); err != nil {
			return nil, err
		}
	}
	if len(tags.All)+len(tags.New)+len(tags.Obsolete)+len(tags.OverwrittenTranslations) == 0 {
		return nil, nil
	}
	return tags, nil
}

func (w *wizard) askFormat(guess string) (string, error) {
	names := make([]string, 0, len(repoconfig.Formats))
	for name := range repoconfig.Formats {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(repoconfig.Formats[names[i]].Name) < strings.ToLower(repoconfig.Formats[names[j]].Name)
	})
	labels := make([]string, len(names))
	for i, name := range names {
		labels[i] = fmt.Sprintf("%s (%s)", repoconfig.Formats[name].Name, name)
	}
	i, err := w.ask.Select("File format", labels, max(slices.Index(names, guess), 0))
	if err != nil {
		return "", err
	}
	return names[i], nil
}

func (w *wizard) askTags(title string) ([]string, error) {
	raw, err := w.ask.Input(title, "", nil)
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, t := range strings.Split(raw, ",") {
		if t = strings.TrimSpace(t); t != "" && !slices.Contains(tags, t) {
			tags = append(tags, t)
		}
	}
	return tags, nil
}

// askProject runs the steps for one new project.
func (w *wizard) askProject(existing *repoconfig.Config) (repoconfig.Project, error) {
	excludeIDs, takenNames, takenPaths := map[int]bool{}, map[string]bool{}, map[string]bool{}
	if existing != nil {
		for _, p := range existing.Projects {
			excludeIDs[p.ID] = true
			if p.Name != "" {
				takenNames[p.Name] = true
			}
			for _, f := range p.Files {
				takenPaths[f.Path] = true
			}
		}
	}

	account, err := w.chooseAccount()
	if err != nil {
		return repoconfig.Project{}, err
	}
	client := w.f.APIClientWithToken(account.resolved.Token)

	remote, err := w.chooseProject(client, excludeIDs)
	if err != nil {
		return repoconfig.Project{}, err
	}
	codes, reference, err := w.projectLanguages(client, remote.ID)
	if err != nil {
		return repoconfig.Project{}, err
	}

	name, err := w.askName(remote.Name, takenNames)
	if err != nil {
		return repoconfig.Project{}, err
	}
	source, err := w.askSourceLanguage(codes, reference)
	if err != nil {
		return repoconfig.Project{}, err
	}

	project := repoconfig.Project{ID: remote.ID, Name: name, Account: account.write, SourceLanguage: source}
	for {
		file, err := w.askFile(codes, takenPaths)
		if err != nil {
			return project, err
		}
		project.Files = append(project.Files, file)
		takenPaths[file.Path] = true

		more, err := w.ask.Confirm("Add another file to this project?", false)
		if err != nil {
			return project, err
		}
		if !more {
			return project, nil
		}
	}
}

func fileCount(n int) string {
	if n == 1 {
		return "1 file"
	}
	return strconv.Itoa(n) + " files"
}
