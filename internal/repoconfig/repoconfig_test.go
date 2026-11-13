package repoconfig_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

func write(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, repoconfig.FileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFind(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := repoconfig.Find(deep); !errors.Is(err, repoconfig.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	want := write(t, root, "projects: []\n")
	got, err := repoconfig.Find(deep)
	if err != nil || got != want {
		t.Fatalf("Find = %q, %v; want %q", got, err, want)
	}
}

func TestLoad_full(t *testing.T) {
	path := write(t, t.TempDir(), `
projects:
  - id: 12345
    name: web
    account: client-a
    source_language: en
    files:
      - path: ios/{lang}.lproj/InfoPlist.strings
        format: apple_strings
        languages: [en, fr]
        language_map: {pt-br: pt-BR}
        pull:
          tags: [infoplist]
          tags_match: any
          filters: [translated]
          order: terms
          fallback_language: en
        push:
          updating: terms_translations
          overwrite: true
          sync_terms: false     # true would make the obsolete tags below useless (see TestValidate)
          read_from_source: true
          fuzzy_trigger: true
          tags:
            all: [infoplist]
            new: [needs-review]
            obsolete: [removed]
            overwritten_translations: [changed]
`)

	cfg, err := repoconfig.Load(path)

	if err != nil {
		t.Fatal(err)
	}
	if issues := cfg.Validate(); len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
	f := cfg.Projects[0].Files[0]
	if cfg.Projects[0].Account != "client-a" || f.Pull.TagsMatch != "any" || !f.Push.ReadFromSource ||
		f.Push.Tags.OverwrittenTranslations[0] != "changed" || f.LanguageMap["pt-br"] != "pt-BR" {
		t.Errorf("fields not decoded: %+v", cfg.Projects[0])
	}
}

func TestLoad_unknownKeyIsAnError(t *testing.T) {
	path := write(t, t.TempDir(), "projects:\n  - id: 1\n    filez: []\n")

	_, err := repoconfig.Load(path)

	if err == nil || !strings.Contains(err.Error(), "line 3: field filez not found") || !strings.HasPrefix(err.Error(), path+": ") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_emptyFile(t *testing.T) {
	cfg, err := repoconfig.Load(write(t, t.TempDir(), ""))

	if err != nil {
		t.Fatal(err)
	}
	if issues := cfg.Validate(); len(issues) != 1 || !strings.Contains(issues[0].Message, "no projects") {
		t.Errorf("issues = %v", issues)
	}
}

func TestValidate(t *testing.T) {
	const ok = "    source_language: en\n"
	tests := []struct {
		name string
		yaml string
		want []string // "severity: where: message" substrings, in order; empty = no issues
	}{
		{"minimal valid", `
projects:
  - id: 1
` + ok + `    files:
      - {path: "l/{lang}.json", format: key_value_json}
`, nil},
		{"missing id, files", `
projects:
  - name: web
` + ok, []string{"error: project web: `id` is required", "error: project web: no files"}},
		{"numeric name and duplicate name", `
projects:
  - {id: 1, name: "123", source_language: en, files: [{path: "a/{lang}.json", format: json}]}
  - {id: 2, name: "123", source_language: en, files: [{path: "b/{lang}.json", format: json}]}
`, []string{"must not be all digits", "must not be all digits", "duplicate project name"}},
		{"bad account name", `
projects:
  - {id: 1, account: "Client A", source_language: en, files: [{path: "a/{lang}.json", format: json}]}
`, []string{`invalid account name "Client A"`}},
		{"no source language", `
projects:
  - {id: 1, files: [{path: "a/{lang}.json", format: json}]}
`, []string{"warning: project 1: `source_language` is not set"}},
		{"path problems", `
projects:
  - id: 1
` + ok + `    files:
      - {format: json}
      - {path: "/abs/{lang}.json", format: json}
      - {path: 'win\{lang}.json', format: json}
      - {path: "x/{locale}.json", format: json}
      - {path: "no-placeholder.json", format: json}
      - {path: "single.pot", format: pot, languages: [en]}
`, []string{
			"file 1: `path` is required",
			"file 2 (/abs/{lang}.json): `path` must be relative",
			`file 3 (win\{lang}.json): use / in`,
			"file 4 (x/{locale}.json): unknown placeholder {locale}",
			"file 4 (x/{locale}.json): `path` needs {lang}",
			"file 5 (no-placeholder.json): `path` needs {lang}",
		}},
		{"format problems", `
projects:
  - id: 1
` + ok + `    files:
      - {path: "a/{lang}.json"}
      - {path: "b/{lang}.x", format: brand_new}
      - {path: "c/{lang}.mo", format: mo, push: {updating: terms}}
`, []string{
			"error: project 1, file 1 (a/{lang}.json): `format` is required",
			`warning: project 1, file 2 (b/{lang}.x): unknown format "brand_new"`,
			"format mo can be pulled but not pushed",
		}},
		{"language map collision", `
projects:
  - id: 1
` + ok + `    files:
      - {path: "a/{lang}.json", format: json, language_map: {pt-br: pt, pt-pt: pt}}
`, []string{"maps both pt-br and pt-pt to \"pt\""}},
		{"pull problems", `
projects:
  - id: 1
` + ok + `    files:
      - path: "a/{lang}.json"
        format: json
        pull: {tags_match: any, filters: [translated, done], order: date}
      - path: "b/{lang}.json"
        format: json
        pull: {tags: [x], tags_match: some}
`, []string{
			"pull.tags_match has no effect without pull.tags",
			`unknown filter "done"`,
			`pull.order "date"`,
			`pull.tags_match "some": use all, any, exclude`,
		}},
		{"push updating", `
projects:
  - id: 1
` + ok + `    files:
      - {path: "a/{lang}.json", format: json, push: {updating: everything}}
`, []string{`push.updating "everything": use terms, terms_translations, translations`}},
		{"obsolete tags with sync_terms", `
projects:
  - id: 1
` + ok + `    files:
      - {path: "a/{lang}.json", format: json, push: {sync_terms: true, tags: {obsolete: [gone]}}}
      - {path: "b/{lang}.json", format: json, push: {sync_terms: true, tags: {all: [b]}}}
`, []string{"warning: project 1, file 1 (a/{lang}.json): push.tags.obsolete has no effect with push.sync_terms: the API deletes obsolete terms instead of tagging them"}},
		{"same path twice", `
projects:
  - {id: 1, source_language: en, files: [{path: "a/{lang}.json", format: json}]}
  - {id: 2, source_language: en, files: [{path: "a/{lang}.json", format: json}]}
`, []string{"error: project 2, file 1 (a/{lang}.json): same path as project 1, file 1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := repoconfig.Load(write(t, t.TempDir(), tt.yaml))
			if err != nil {
				t.Fatal(err)
			}

			issues := cfg.Validate()

			if len(issues) != len(tt.want) {
				t.Fatalf("got %d issues, want %d:\n%v", len(issues), len(tt.want), issues)
			}
			for i, want := range tt.want {
				if !strings.Contains(issues[i].String(), want) {
					t.Errorf("issue %d = %q, want it to contain %q", i, issues[i].String(), want)
				}
			}
		})
	}
}

func TestHasErrors(t *testing.T) {
	warning := repoconfig.Issue{Severity: repoconfig.Warning}
	err := repoconfig.Issue{Severity: repoconfig.Error}

	if repoconfig.HasErrors(nil) || repoconfig.HasErrors([]repoconfig.Issue{warning}) {
		t.Error("warnings alone are not errors")
	}
	if !repoconfig.HasErrors([]repoconfig.Issue{warning, err}) {
		t.Error("an error issue was missed")
	}
}

func TestSelect(t *testing.T) {
	cfg, err := repoconfig.Load(write(t, t.TempDir(), `
projects:
  - {id: 1, name: web, files: []}
  - {id: 2, files: []}
`))
	if err != nil {
		t.Fatal(err)
	}

	for selector, wantID := range map[string]int{"web": 1, "1": 1, "2": 2} {
		got, err := cfg.Select(selector)
		if err != nil || len(got) != 1 || got[0].ID != wantID {
			t.Errorf("Select(%q) = %+v, %v", selector, got, err)
		}
	}
	if all, _ := cfg.Select(""); len(all) != 2 {
		t.Errorf("Select(\"\") returned %d projects", len(all))
	}
	if _, err := cfg.Select("nope"); err == nil || !strings.Contains(err.Error(), `no project "nope"`) || !strings.Contains(err.Error(), "(projects: web, 2)") {
		t.Errorf("Select(nope) error = %v", err)
	}
}

func TestLocalPath(t *testing.T) {
	f := repoconfig.File{Path: "res/values-{lang}/strings.xml", LanguageMap: map[string]string{"pt-br": "pt-rBR"}}
	dir := filepath.Join("repo", "app")

	if got, want := f.LocalPath(dir, "fr"), filepath.Join("repo", "app", "res", "values-fr", "strings.xml"); got != want {
		t.Errorf("fr: %q, want %q", got, want)
	}
	if got, want := f.LocalPath(dir, "pt-br"), filepath.Join("repo", "app", "res", "values-pt-rBR", "strings.xml"); got != want {
		t.Errorf("pt-br: %q, want %q", got, want)
	}
}
