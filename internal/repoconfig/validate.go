package repoconfig

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/POEditor/poeditor-cli/internal/accounts"
)

// Severity of a validation issue.
type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

// Issue is one problem found in the config.
type Issue struct {
	Severity Severity `json:"severity"`
	Where    string   `json:"where"` // e.g. "project web, file 2 (locales/{lang}.json)"
	Message  string   `json:"message"`
}

func (i Issue) String() string { return fmt.Sprintf("%s: %s: %s", i.Severity, i.Where, i.Message) }

// HasErrors reports whether any issue is an Error.
func HasErrors(issues []Issue) bool {
	return slices.ContainsFunc(issues, func(i Issue) bool { return i.Severity == Error })
}

var (
	allDigits   = regexp.MustCompile(`^[0-9]+$`)
	placeholder = regexp.MustCompile(`\{[^}]*\}`)
)

// Validate checks the values Load cannot: everything the API would reject or,
// worse, silently ignore.
func (c *Config) Validate() []Issue {
	v := &validator{}
	if len(c.Projects) == 0 {
		v.add(Error, c.path, "no projects: add at least one under `projects:`")
		return v.issues
	}

	names := map[string]bool{}
	seenPaths := map[string]string{}
	for pi, p := range c.Projects {
		where := fmt.Sprintf("project %d", pi+1)
		if p.Name != "" {
			where = fmt.Sprintf("project %s", p.Name)
		}

		if p.ID <= 0 {
			v.add(Error, where, "`id` is required: the project ID shown in POEditor")
		}
		if p.Name != "" {
			if allDigits.MatchString(p.Name) {
				v.add(Error, where, "`name` must not be all digits (it would be mistaken for a project id)")
			}
			if names[p.Name] {
				v.add(Error, where, "duplicate project name")
			}
			names[p.Name] = true
		}
		if p.Account != "" {
			if err := accounts.ValidateName(p.Account); err != nil {
				v.add(Error, where, err.Error())
			}
		}
		if p.SourceLanguage == "" {
			v.add(Warning, where, "`source_language` is not set: `push` will not know which language to upload terms from")
		}
		if len(p.Files) == 0 {
			v.add(Error, where, "no files: add at least one under `files:`")
		}

		for fi, f := range p.Files {
			fwhere := fmt.Sprintf("%s, file %d", where, fi+1)
			if f.Path != "" {
				fwhere = fmt.Sprintf("%s, file %d (%s)", where, fi+1, f.Path)
			}
			v.file(fwhere, f)
			if f.Path != "" {
				if other, dup := seenPaths[f.Path]; dup {
					v.add(Error, fwhere, "same path as "+other)
				}
				seenPaths[f.Path] = fwhere
			}
		}
	}
	return v.issues
}

type validator struct{ issues []Issue }

func (v *validator) add(sev Severity, where, msg string) {
	v.issues = append(v.issues, Issue{Severity: sev, Where: where, Message: msg})
}

func (v *validator) file(where string, f File) {
	switch {
	case f.Path == "":
		v.add(Error, where, "`path` is required, e.g. locales/{lang}.json")
	case path.IsAbs(f.Path) || strings.HasPrefix(f.Path, "\\") || (len(f.Path) > 1 && f.Path[1] == ':'):
		v.add(Error, where, "`path` must be relative to the directory of "+FileName)
	case strings.Contains(f.Path, "\\"):
		v.add(Error, where, "use / in `path`, also on Windows")
	}
	for _, ph := range placeholder.FindAllString(f.Path, -1) {
		if ph != "{lang}" {
			v.add(Error, where, fmt.Sprintf("unknown placeholder %s: only {lang} is supported", ph))
		}
	}
	if f.Path != "" && !strings.Contains(f.Path, "{lang}") && len(f.Languages) != 1 {
		v.add(Error, where, "`path` needs {lang} unless `languages` lists exactly one language")
	}

	format, known := Formats[f.Format]
	switch {
	case f.Format == "":
		v.add(Error, where, "`format` is required, e.g. key_value_json")
	case !known:
		v.add(Warning, where, fmt.Sprintf("unknown format %q; known formats: %s", f.Format, strings.Join(formatNames(), ", ")))
	}

	mapped := map[string]string{}
	for code, local := range f.LanguageMap {
		if code == "" || local == "" {
			v.add(Error, where, "`language_map` entries need both a POEditor code and a local code")
			continue
		}
		if other, dup := mapped[local]; dup {
			a, b := sortedPair(other, code)
			v.add(Error, where, fmt.Sprintf("`language_map` maps both %s and %s to %q: they would overwrite each other's file", a, b, local))
		}
		mapped[local] = code
	}

	if p := f.Pull; p != nil {
		if p.TagsMatch != "" {
			if !slices.Contains(TagsMatchModes, p.TagsMatch) {
				v.add(Error, where, fmt.Sprintf("pull.tags_match %q: use %s", p.TagsMatch, strings.Join(TagsMatchModes, ", ")))
			} else if len(p.Tags) == 0 {
				v.add(Error, where, "pull.tags_match has no effect without pull.tags")
			}
		}
		for _, filter := range p.Filters {
			if !slices.Contains(PullFilters, filter) {
				v.add(Error, where, fmt.Sprintf("pull.filters: unknown filter %q (the API would ignore it); use %s", filter, strings.Join(PullFilters, ", ")))
			}
		}
		if p.Order != "" && p.Order != "terms" {
			v.add(Error, where, fmt.Sprintf("pull.order %q: the only supported order is terms", p.Order))
		}
	}

	if p := f.Push; p != nil {
		if known && !format.Import {
			v.add(Error, where, fmt.Sprintf("format %s can be pulled but not pushed: remove the `push` block", f.Format))
		}
		if p.Updating != "" && !slices.Contains(UpdatingModes, p.Updating) {
			v.add(Error, where, fmt.Sprintf("push.updating %q: use %s", p.Updating, strings.Join(UpdatingModes, ", ")))
		}
		if p.SyncTerms && p.Tags != nil && len(p.Tags.Obsolete) > 0 {
			v.add(Warning, where, "push.tags.obsolete has no effect with push.sync_terms: the API deletes obsolete terms instead of tagging them")
		}
	}
}

func formatNames() []string {
	names := make([]string, 0, len(Formats))
	for name := range Formats {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedPair(a, b string) (string, string) {
	if a > b {
		return b, a
	}
	return a, b
}
