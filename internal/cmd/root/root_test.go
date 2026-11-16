package root_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/cmd/api/raw"
	"github.com/POEditor/poeditor-cli/internal/cmd/root"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

const modulePath = "github.com/POEditor/poeditor-cli/"

// Every runnable command must live in a package whose tests use testutil.Main,
// which is what enforces a test case per command and per flag.
func TestEveryCommandPackageHasCoverageMain(t *testing.T) {
	repoRoot := findRepoRoot(t)
	checked := map[string]bool{}

	for _, cmd := range testutil.Leaves() {
		pkg := testutil.CommandPackage(cmd)
		if checked[pkg] {
			continue
		}
		checked[pkg] = true

		if !strings.HasPrefix(pkg, modulePath) {
			t.Errorf("%q is implemented outside the module (%s)", cmd.CommandPath(), pkg)
			continue
		}
		dir := filepath.Join(repoRoot, filepath.FromSlash(strings.TrimPrefix(pkg, modulePath)))
		if !dirHasTestMain(t, dir) {
			t.Errorf("%q: package %s has no test file calling testutil.Main(m)", cmd.CommandPath(), pkg)
		}
	}
	if len(checked) == 0 {
		t.Fatal("no commands found in the tree")
	}
}

func TestVersionFlag(t *testing.T) {
	res := testutil.Run(t, nil, "--version")

	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	testutil.Golden(t, res.Stdout)
}

func TestUnknownCommand(t *testing.T) {
	res := testutil.Run(t, nil, "nope")

	if res.Err == nil || !strings.Contains(res.Err.Error(), `unknown command "nope"`) {
		t.Fatalf("want unknown command error, got %v", res.Err)
	}
}

// Every command declares whether it prints data or performs an action, and
// every data command offers --json (cmdutil.AddJSONFlag / cmdutil.MarkAction).
func TestEveryCommandDeclaresItsOutput(t *testing.T) {
	for _, cmd := range testutil.Leaves() {
		switch kind := cmd.Annotations[cmdutil.OutputAnnotation]; kind {
		case cmdutil.OutputData:
			if cmd.Flags().Lookup("json") == nil {
				t.Errorf("%q prints data but has no --json flag (use cmdutil.AddJSONFlag)", cmd.CommandPath())
			}
		case cmdutil.OutputAction:
		default:
			t.Errorf("%q does not declare its output: call cmdutil.AddJSONFlag (data) or cmdutil.MarkAction (action)", cmd.CommandPath())
		}
	}
}

// Only the workflow commands (pull, push) and the auth and config modules
// sit at the top level. Every API v2 endpoint is exactly one command,
// `poeditor api <group> <action>`, and nothing else lives under `api`.
func TestCommandTree(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	var top []string
	for _, cmd := range root.NewCmdRoot(&cmdutil.Factory{IO: ios}).Commands() {
		if !cmd.Hidden && cmd.Name() != "help" {
			top = append(top, cmd.Name())
		}
	}
	if want := []string{"api", "auth", "config", "pull", "push"}; !reflect.DeepEqual(top, want) {
		t.Errorf("top level = %v, want %v", top, want)
	}

	endpoints := map[string]bool{}
	for _, endpoint := range raw.Endpoints {
		group, action, _ := strings.Cut(endpoint, "/")
		endpoints["poeditor api "+group+" "+raw.CommandName(action)] = true
	}
	found := map[string]bool{}
	for _, cmd := range testutil.Leaves() {
		path := cmd.CommandPath()
		if !strings.HasPrefix(path, "poeditor api ") {
			continue
		}
		if !endpoints[path] {
			t.Errorf("%q is not an API v2 endpoint", path)
		}
		found[path] = true
	}
	for path := range endpoints {
		if !found[path] {
			t.Errorf("no command for %q", path)
		}
	}
	for _, typed := range []string{"poeditor api projects list", "poeditor api terms list"} {
		for _, cmd := range testutil.Leaves() {
			if cmd.CommandPath() == typed && strings.HasSuffix(testutil.CommandPackage(cmd), "/raw") {
				t.Errorf("%q is a raw command; it has a dedicated one", typed)
			}
		}
	}
}

// A group run with a word that is not one of its commands fails, at every
// level, instead of printing its help and exiting 0.
func TestUnknownSubcommand(t *testing.T) {
	for _, args := range [][]string{{"api", "nope"}, {"api", "terms", "nope"}, {"auth", "nope"}} {
		res := testutil.Run(t, nil, args...)

		if res.Err == nil || !strings.Contains(res.Err.Error(), `unknown command "nope"`) {
			t.Errorf("%v: err = %v", args, res.Err)
		}
	}
	res := testutil.Run(t, nil, "api", "terms")
	if res.Err != nil || !strings.Contains(res.Stdout, "Available Commands") {
		t.Errorf("a group alone shows its help: err = %v, stdout = %.80q", res.Err, res.Stdout)
	}
}

func dirHasTestMain(t *testing.T, dir string) bool {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(src), "testutil.Main(m)") {
			return true
		}
	}
	return false
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test directory")
		}
		dir = parent
	}
}
