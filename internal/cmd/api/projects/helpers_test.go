package projects_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/testutil"
)

// projects/view, add and update as the API sends them
// (ApiV2_Projects::generateProjectArray).
const (
	webProject = `{"project":{"id":7717,"name":"Web","description":"The site\twith a tab\nand a line","public":1,"open":0,` +
		`"reference_language":"en","fallback_language":"","terms":12,"terms_last_added":"2026-03-01T08:00:00+0000",` +
		`"terms_last_updated":"2026-03-02T09:30:00+0000","created":"2026-01-14T09:12:44+0000"}}`
	newProject = `{"project":{"id":9001,"name":"Mobile app","description":"","public":0,"open":0,` +
		`"reference_language":"","fallback_language":"","terms":0,"terms_last_added":"","terms_last_updated":"","created":"2026-11-17T10:00:00+0000"}}`
)

func writeFile(t *testing.T, env *testutil.Env, name, content string) {
	t.Helper()
	path := filepath.Join(env.WorkDir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// env is a logged-in Env with .poeditor.yml naming project 7717 "web".
func env(t *testing.T, fake *testutil.FakeAPI) *testutil.Env {
	e := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
	e.WriteRepoConfig(`projects:
  - id: 7717
    alias: web
    files:
      - path: locales/{lang}.json
        format: key_value_json
`)
	return e
}

// assertJSON checks a JSON parameter, spacing aside.
func assertJSON(t *testing.T, got, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		t.Fatalf("not JSON: %v: %s", err, got)
	}
	_ = json.Unmarshal([]byte(want), &w)
	if !reflect.DeepEqual(g, w) {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
