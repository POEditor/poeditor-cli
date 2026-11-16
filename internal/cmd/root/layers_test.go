package root_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const internalPath = modulePath + "internal/"

// layers says which internal packages each foundation package may import.
// Packages not listed (cmdutil, transfer, testutil, cmd/...) may import any
// of these; nothing may import upwards. Adding a package means adding a line
// here, which is the moment to decide where it sits.
var layers = map[string][]string{
	"api":        {},
	"build":      {},
	"config":     {},
	"iostreams":  {},
	"keyring":    {},
	"prompt":     {},
	"accounts":   {"config", "keyring"},
	"repoconfig": {"accounts"},
}

// upper packages may use the foundation, and are used only from internal/cmd
// (testutil excepted, as it builds the real root command).
var upper = map[string][]string{
	"cmdutil":  {"accounts", "api", "build", "config", "iostreams", "keyring", "prompt"},
	"transfer": {"api", "cmdutil", "repoconfig"},
}

func TestPackageLayers(t *testing.T) {
	root := filepath.Join(findRepoRoot(t), "internal")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || d.Name() == "testdata" {
			return err
		}
		pkg, _ := filepath.Rel(root, path)
		pkg = filepath.ToSlash(pkg)
		imports, isPackage := internalImports(t, path)
		if !isPackage {
			return nil
		}

		allowed, isFoundation := layers[pkg]
		if !isFoundation {
			allowed, isFoundation = upper[pkg], upper[pkg] != nil
		}
		if !isFoundation && !strings.HasPrefix(pkg, "cmd/") && pkg != "testutil" {
			t.Errorf("internal/%s is not in the layer map of layers_test.go: add it to layers or upper", pkg)
			return nil
		}
		for _, imp := range imports {
			if isFoundation && !contains(allowed, imp) {
				t.Errorf("internal/%s imports internal/%s, which its layer does not allow (layers_test.go)", pkg, imp)
			}
			if strings.HasPrefix(imp, "cmd/") && !strings.HasPrefix(pkg, "cmd/") && pkg != "testutil" {
				t.Errorf("internal/%s imports internal/%s: only commands import commands", pkg, imp)
			}
			if imp == "testutil" {
				t.Errorf("internal/%s imports internal/testutil outside a test", pkg)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// internalImports lists the internal packages imported by the non-test Go
// files of dir, relative to internal/; isPackage is false when dir has none.
func internalImports(t *testing.T, dir string) (imports []string, isPackage bool) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		isPackage = true
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			if rel, ok := strings.CutPrefix(path, internalPath); ok && !seen[rel] {
				seen[rel] = true
				imports = append(imports, rel)
			}
		}
	}
	return imports, isPackage
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
