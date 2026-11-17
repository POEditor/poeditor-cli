package root_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/POEditor/poeditor-cli/internal/testutil"
)

// The user docs (README.md and docs/*.md) must keep up with the commands:
// every command has a section in its docs file, headed by its full name in
// backticks, and every flag of the command is mentioned in that section; a
// hidden synonym (--id) at least once in the file.

// docsFile is the docs file that documents cmd.
func docsFile(cmd *cobra.Command) string {
	words := strings.Fields(cmd.CommandPath()) // poeditor api terms add, poeditor pull…
	switch words[1] {
	case "api":
		return words[2] + ".md"
	case "pull", "push":
		return "pull-push.md"
	}
	return words[1] + ".md" // auth, config
}

func TestDocs_everyCommandAndFlag(t *testing.T) {
	root := findRepoRoot(t)
	for _, cmd := range testutil.Leaves() {
		path := cmd.CommandPath()
		file := docsFile(cmd)
		data, err := os.ReadFile(filepath.Join(root, "docs", file))
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		section, ok := docSection(string(data), "`"+path+"`")
		if !ok {
			t.Errorf("docs/%s has no section headed `%s`", file, path)
			continue
		}
		cmd.LocalFlags().VisitAll(func(fl *pflag.Flag) {
			switch {
			case fl.Name == "help":
			case fl.Hidden:
				// a synonym without its own --help row (--id): once per file is enough
				if !strings.Contains(string(data), "`--"+fl.Name+"`") {
					t.Errorf("docs/%s: flag --%s (of `%s`) is not mentioned anywhere", file, fl.Name, path)
				}
			case !strings.Contains(section, "--"+fl.Name):
				t.Errorf("docs/%s, section `%s`: flag --%s is not mentioned", file, path, fl.Name)
			}
		})
	}
}

// The README links to every docs file, and every docs file is linked.
func TestDocs_readmeLinksEveryFile(t *testing.T) {
	root := findRepoRoot(t)
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "docs", "*.md"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no docs/*.md: %v", err)
	}
	for _, f := range files {
		if link := "(docs/" + filepath.Base(f) + ")"; !strings.Contains(string(readme), link) {
			t.Errorf("README.md does not link docs/%s", filepath.Base(f))
		}
	}
}

// Relative links between the docs (and from the README) point at files and
// headings that exist.
func TestDocs_linksResolve(t *testing.T) {
	root := findRepoRoot(t)
	sources, _ := filepath.Glob(filepath.Join(root, "docs", "*.md"))
	sources = append(sources, filepath.Join(root, "README.md"))
	link := regexp.MustCompile(`\]\(([^)#:]*\.md|[^)#:]*\.svg)?(#[^)]+)?\)`)
	for _, src := range sources {
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(root, src)
		for _, m := range link.FindAllStringSubmatch(string(data), -1) {
			target, anchor := m[1], strings.TrimPrefix(m[2], "#")
			targetPath := src
			if target != "" {
				targetPath = filepath.Join(filepath.Dir(src), target)
			}
			content, err := os.ReadFile(targetPath)
			if err != nil {
				t.Errorf("%s: link %s: %v", rel, m[0], err)
				continue
			}
			if anchor != "" && !hasAnchor(string(content), anchor) {
				t.Errorf("%s: link %s: no heading with that anchor", rel, m[0])
			}
		}
	}
}

// docSection returns the text from the heading that contains title to the
// next heading of the same or a higher level.
func docSection(doc, title string) (string, bool) {
	lines := strings.Split(doc, "\n")
	for i, line := range lines {
		level := headingLevel(line)
		if level == 0 || !strings.Contains(line, title) {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if l := headingLevel(lines[j]); l > 0 && l <= level {
				end = j
				break
			}
		}
		return strings.Join(lines[i:end], "\n"), true
	}
	return "", false
}

func headingLevel(line string) int {
	n := len(line) - len(strings.TrimLeft(line, "#"))
	if n == 0 || n > 6 || !strings.HasPrefix(line[n:], " ") {
		return 0
	}
	return n
}

// hasAnchor reports a heading whose GitHub anchor is anchor: lower-cased,
// punctuation dropped, spaces as dashes.
func hasAnchor(doc, anchor string) bool {
	for _, line := range strings.Split(doc, "\n") {
		if level := headingLevel(line); level > 0 && slug(line[level+1:]) == anchor {
			return true
		}
	}
	return false
}

func slug(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(heading)) {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"`poeditor api terms add`": "poeditor-api-terms-add",
		"`.poeditor.yml`":          "poeditoryml",
		"Picking a project":        "picking-a-project",
		"`pull:` options":          "pull-options",
	} {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}
