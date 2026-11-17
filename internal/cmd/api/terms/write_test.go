package terms_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/testutil"
)

// Helpers of the tests of the commands that write terms.

func writeFile(t *testing.T, env *testutil.Env, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(env.WorkDir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// assertData checks the "data" parameter as JSON (spacing aside).
func assertData(t *testing.T, got, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		t.Fatalf("data is not JSON: %v: %s", err, got)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("bad want: %v", err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("data = %s\nwant   %s", got, want)
	}
}

// writeEnv is a logged-in Env with .poeditor.yml naming project 7717 "web".
func writeEnv(t *testing.T, fake *testutil.FakeAPI) *testutil.Env {
	env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
	env.WriteRepoConfig(`projects:
  - id: 7717
    alias: web
    files:
      - path: locales/{lang}.json
        format: key_value_json
`)
	return env
}
