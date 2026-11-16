package root_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
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

// The api module's commands sit at the top level (`poeditor projects list`);
// `poeditor api` is the raw request command (like `gh api`), a single leaf
// with no subcommands: `poeditor api projects list` sends the raw request.
func TestAPIModuleIsTopLevel(t *testing.T) {
	var foundList, foundRaw bool
	for _, cmd := range testutil.Leaves() {
		switch cmd.CommandPath() {
		case "poeditor projects list":
			foundList = true
		case "poeditor api":
			foundRaw = true
			if cmd.HasSubCommands() {
				t.Error("`poeditor api` must have no subcommands")
			}
		}
	}
	if !foundList || !foundRaw {
		t.Fatalf("top level: projects list found = %v, api found = %v", foundList, foundRaw)
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
