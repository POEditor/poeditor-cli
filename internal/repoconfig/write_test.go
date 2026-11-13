package repoconfig_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

func TestCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), repoconfig.FileName)
	cfg := repoconfig.New(path, []repoconfig.Project{{ID: 1, Name: "web", SourceLanguage: "en",
		Files: []repoconfig.File{{Path: "l/{lang}.json", Format: "key_value_json"}}}})

	if err := cfg.Create(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	want := repoconfig.Header + `
projects:
  - id: 1
    name: web
    source_language: en
    files:
      - path: l/{lang}.json
        format: key_value_json
`
	if string(data) != want {
		t.Errorf("file:\n%s\nwant:\n%s", data, want)
	}

	if err := cfg.Create(); !errors.Is(err, fs.ErrExist) {
		t.Errorf("second Create should refuse to overwrite, got %v", err)
	}
}

const handWritten = `# our team's config
projects:
  # the website
  - id: 1
    name: web
    source_language: en
    files: [{path: "a/{lang}.json", format: json}]  # inline on purpose
`

func TestAppendProject_keepsComments(t *testing.T) {
	path := write(t, t.TempDir(), handWritten)

	err := repoconfig.AppendProject(path, repoconfig.Project{ID: 2, SourceLanguage: "en",
		Files: []repoconfig.File{{Path: "b/{lang}.po", Format: "po"}}})

	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	for _, keep := range []string{"# our team's config", "# the website", "# inline on purpose"} {
		if !strings.Contains(string(data), keep) {
			t.Errorf("comment %q lost:\n%s", keep, data)
		}
	}
	cfg, err := repoconfig.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Projects) != 2 || cfg.Projects[1].ID != 2 || cfg.Projects[1].Files[0].Format != "po" {
		t.Errorf("projects = %+v", cfg.Projects)
	}
}

func TestAppendFile(t *testing.T) {
	path := write(t, t.TempDir(), handWritten)

	if err := repoconfig.AppendFile(path, 0, repoconfig.File{Path: "c/{lang}.json", Format: "json"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := repoconfig.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if files := cfg.Projects[0].Files; len(files) != 2 || files[1].Path != "c/{lang}.json" {
		t.Errorf("files = %+v", files)
	}
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), "# the website") {
		t.Errorf("comment lost:\n%s", data)
	}

	if err := repoconfig.AppendFile(path, 5, repoconfig.File{}); err == nil || !strings.Contains(err.Error(), "no project number 6") {
		t.Errorf("bad index: %v", err)
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		path   string
		single bool
		want   string
	}{
		{"locales/{lang}.json", false, ""},
		{"messages.pot", true, ""},
		{"", false, "enter a path"},
		{"/abs/{lang}.json", false, "must be relative"},
		{`C:\x\{lang}.json`, false, "must be relative"},
		{`dir\{lang}.json`, false, "use /"},
		{"x/{locale}.json", false, "unknown placeholder {locale}"},
		{"messages.json", false, "put {lang} where"},
	}
	for _, tt := range tests {
		err := repoconfig.ValidatePath(tt.path, tt.single)
		if tt.want == "" && err != nil || tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
			t.Errorf("ValidatePath(%q) = %v, want %q", tt.path, err, tt.want)
		}
	}
}
