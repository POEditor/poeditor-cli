package repoconfig

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Header is the comment at the top of a .poeditor.yml written by the CLI.
const Header = `# POEditor CLI: which POEditor projects this repo uses and where their files live.
# Run "poeditor config show" to see what it maps and "poeditor config check" to
# check it against POEditor. The API token is never stored here.
`

// New returns a config that will be saved to path.
func New(path string, projects []Project) *Config {
	return &Config{Projects: projects, path: path}
}

// Create writes a new file with the header. It fails if the file exists.
func (c *Config) Create() error {
	body, err := encode(c)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(c.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(Header + "\n" + string(body)); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// AppendProject adds a project to the file at path, keeping everything else
// in it (comments, order, formatting of untouched entries) as it is.
func AppendProject(path string, p Project) error {
	return edit(path, func(root *yaml.Node) error {
		projects, err := projectsNode(root)
		if err != nil {
			return err
		}
		node, err := toNode(p)
		if err != nil {
			return err
		}
		projects.Content = append(projects.Content, node)
		return nil
	})
}

// AppendFile adds a file to the project at index (0-based) in the file at path.
func AppendFile(path string, index int, f File) error {
	return edit(path, func(root *yaml.Node) error {
		projects, err := projectsNode(root)
		if err != nil {
			return err
		}
		if index < 0 || index >= len(projects.Content) {
			return fmt.Errorf("no project number %d", index+1)
		}
		files := mapValue(projects.Content[index], "files")
		if files == nil || files.Kind != yaml.SequenceNode {
			return errors.New("the project has no `files:` list")
		}
		node, err := toNode(f)
		if err != nil {
			return err
		}
		files.Style = 0 // block style, even if the list was written inline
		files.Content = append(files.Content, node)
		return nil
	})
}

func edit(path string, change func(root *yaml.Node) error) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return fmt.Errorf("%s is empty", path)
	}
	if err := change(doc.Content[0]); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	out, err := encode(&doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

func projectsNode(root *yaml.Node) (*yaml.Node, error) {
	projects := mapValue(root, "projects")
	if projects == nil || projects.Kind != yaml.SequenceNode {
		return nil, errors.New("no `projects:` list")
	}
	projects.Style = 0
	return projects, nil
}

func mapValue(m *yaml.Node, key string) *yaml.Node {
	if m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func toNode(v any) (*yaml.Node, error) {
	var n yaml.Node
	if err := n.Encode(v); err != nil {
		return nil, err
	}
	return &n, nil
}

func encode(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), enc.Close()
}

// ValidatePath checks a file path the way Validate does, for prompts.
// singleLanguage is true when the file lists exactly one language.
func ValidatePath(p string, singleLanguage bool) error {
	switch {
	case strings.TrimSpace(p) == "":
		return errors.New("enter a path, e.g. locales/{lang}.json")
	case path.IsAbs(p) || strings.HasPrefix(p, "\\") || (len(p) > 1 && p[1] == ':'):
		return errors.New("the path must be relative to this directory")
	case strings.Contains(p, "\\"):
		return errors.New("use / in the path, also on Windows")
	}
	for _, ph := range placeholder.FindAllString(p, -1) {
		if ph != "{lang}" {
			return fmt.Errorf("unknown placeholder %s: only {lang} is supported", ph)
		}
	}
	if !strings.Contains(p, "{lang}") && !singleLanguage {
		return errors.New("put {lang} where the language code goes, e.g. locales/{lang}.json")
	}
	return nil
}
