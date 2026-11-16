// Package repoconfig is the per-repo .poeditor.yml: which POEditor projects
// the repo uses and how they map to local files. See docs/design-decisions.md.
package repoconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// FileName is the repo config file name.
const FileName = ".poeditor.yml"

// ErrNotFound is returned by Find when no .poeditor.yml exists up the tree.
var ErrNotFound = errors.New("no " + FileName + " found here or in any parent directory. Run `poeditor config init` to create one")

// Config is a parsed .poeditor.yml.
type Config struct {
	Projects []Project `yaml:"projects" json:"projects"`

	path string
}

// Project is one POEditor project used by the repo.
type Project struct {
	ID             int    `yaml:"id" json:"id"`
	Alias          string `yaml:"alias,omitempty" json:"alias,omitempty"` // local nickname for --project
	Account        string `yaml:"account,omitempty" json:"account,omitempty"`
	SourceLanguage string `yaml:"source_language,omitempty" json:"source_language,omitempty"`
	Files          []File `yaml:"files" json:"files"`
}

// File is one local file pattern of a project.
type File struct {
	Path        string            `yaml:"path" json:"path"`
	Format      string            `yaml:"format" json:"format"`
	Languages   []string          `yaml:"languages,omitempty" json:"languages,omitempty"`
	LanguageMap map[string]string `yaml:"language_map,omitempty" json:"language_map,omitempty"`
	Pull        *Pull             `yaml:"pull,omitempty" json:"pull,omitempty"`
	Push        *Push             `yaml:"push,omitempty" json:"push,omitempty"`
}

// Pull holds the projects/export options.
type Pull struct {
	Tags             []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	TagsMatch        string   `yaml:"tags_match,omitempty" json:"tags_match,omitempty"`
	Filters          []string `yaml:"filters,omitempty" json:"filters,omitempty"`
	Order            string   `yaml:"order,omitempty" json:"order,omitempty"`
	FallbackLanguage string   `yaml:"fallback_language,omitempty" json:"fallback_language,omitempty"`
}

// Push holds the projects/upload options.
type Push struct {
	Updating       string    `yaml:"updating,omitempty" json:"updating,omitempty"`
	Overwrite      bool      `yaml:"overwrite,omitempty" json:"overwrite,omitempty"`
	SyncTerms      bool      `yaml:"sync_terms,omitempty" json:"sync_terms,omitempty"`
	ReadFromSource bool      `yaml:"read_from_source,omitempty" json:"read_from_source,omitempty"`
	FuzzyTrigger   bool      `yaml:"fuzzy_trigger,omitempty" json:"fuzzy_trigger,omitempty"`
	Tags           *PushTags `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// PushTags are the projects/upload tag targets.
type PushTags struct {
	All                     []string `yaml:"all,omitempty" json:"all,omitempty"`
	New                     []string `yaml:"new,omitempty" json:"new,omitempty"`
	Obsolete                []string `yaml:"obsolete,omitempty" json:"obsolete,omitempty"`
	OverwrittenTranslations []string `yaml:"overwritten_translations,omitempty" json:"overwritten_translations,omitempty"`
}

// Find returns the path of the nearest .poeditor.yml at or above dir.
func Find(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, FileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

// Load parses path. Unknown keys are an error, so typos never pass silently.
// Load does not validate values: call Validate.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	cfg := &Config{path: path}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, decodeError(path, err)
	}
	return cfg, nil
}

// renamedKeys are project keys that were renamed: old → new.
var renamedKeys = map[string]string{"name": "alias"}

var unknownProjectKey = regexp.MustCompile(`^(line \d+): field (\S+) not found in type repoconfig\.Project$`)

// decodeError says what is wrong in the file, and how to fix a key that was
// renamed since the file was written.
func decodeError(path string, err error) error {
	var typeErr *yaml.TypeError
	if !errors.As(err, &typeErr) {
		return fmt.Errorf("%s: %s", path, strings.TrimPrefix(err.Error(), "yaml: "))
	}
	problems := make([]string, len(typeErr.Errors))
	for i, problem := range typeErr.Errors {
		if m := unknownProjectKey.FindStringSubmatch(problem); m != nil && renamedKeys[m[2]] != "" {
			problem = fmt.Sprintf("%s: field %q was renamed to %q (rename it there)", m[1], m[2], renamedKeys[m[2]])
		}
		problems[i] = problem
	}
	if len(problems) == 1 {
		return fmt.Errorf("%s: %s", path, problems[0])
	}
	return fmt.Errorf("%s:\n  %s", path, strings.Join(problems, "\n  "))
}

// Path is the file the config was loaded from.
func (c *Config) Path() string { return c.path }

// Dir is the directory file paths are relative to.
func (c *Config) Dir() string { return filepath.Dir(c.path) }

// ProjectRef is how JSON output names a project: its id, the alias from
// .poeditor.yml (if any) and, when the command fetched it, its POEditor name.
type ProjectRef struct {
	ID    int    `json:"id"`
	Alias string `json:"alias,omitempty"`
	Name  string `json:"name,omitempty"`
}

// Ref returns the project's JSON reference (without the POEditor name).
func (p Project) Ref() ProjectRef { return ProjectRef{ID: p.ID, Alias: p.Alias} }

// Label is how a project is named in messages: its alias, else its id.
func (p Project) Label() string {
	if p.Alias != "" {
		return p.Alias
	}
	return strconv.Itoa(p.ID)
}

// Select returns the projects matching --project (an alias or an id), or all
// of them when selector is empty.
func (c *Config) Select(selector string) ([]Project, error) {
	if selector == "" {
		return c.Projects, nil
	}
	for _, p := range c.Projects {
		if p.Alias == selector || strconv.Itoa(p.ID) == selector {
			return []Project{p}, nil
		}
	}
	labels := make([]string, len(c.Projects))
	for i, p := range c.Projects {
		labels[i] = p.Label()
	}
	return nil, fmt.Errorf("no project %q in %s (projects: %s)", selector, c.path, strings.Join(labels, ", "))
}

// LocalPath is where language lang of this file lives on disk, under dir.
func (f File) LocalPath(dir, lang string) string {
	code := lang
	if mapped, ok := f.LanguageMap[lang]; ok {
		code = mapped
	}
	return filepath.Join(dir, filepath.FromSlash(strings.ReplaceAll(f.Path, "{lang}", code)))
}
