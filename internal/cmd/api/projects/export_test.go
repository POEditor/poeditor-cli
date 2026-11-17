package projects_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

const frPO = "msgid \"save\"\nmsgstr \"Enregistrer\"\n"

func exportFake(t *testing.T) *testutil.FakeAPI {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/export", testutil.OK(map[string]string{"url": fake.File("fr.po", []byte(frPO))}))
	return fake
}

func TestExport_params(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want map[string]string // besides id, language, type; "" = not sent
	}{
		{"required only", nil, map[string]string{"filters": "", "tags": "", "tags_match": "", "order": "", "fallback_language": ""}},
		{"every option", []string{"--filter", "translated", "--filter", "not_fuzzy", "--tag", "web", "--tag", "v2", "--tags-match", "all", "--order", "terms", "--fallback-language", "en"},
			map[string]string{"filters": `["translated","not_fuzzy"]`, "tags": `["web","v2"]`, "tags_match": "all", "order": "terms", "fallback_language": "en"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := exportFake(t)

			res := env(t, fake).Run(append([]string{"api", "projects", "export", "-p", "web", "--language", "fr", "--format", "po"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.Requests()[0]
			if req.Endpoint != "projects/export" || req.Form.Get("id") != "7717" || req.Form.Get("language") != "fr" || req.Form.Get("type") != "po" {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			for k, want := range tt.want {
				if got := req.Form.Get(k); got != want || (want == "" && req.Form.Has(k)) {
					t.Errorf("%s = %q, want %q", k, got, want)
				}
			}
		})
	}
}

// Without --output the file goes to stdout, as downloaded.
func TestExport_stdout(t *testing.T) {
	fake := exportFake(t)

	res := env(t, fake).Run("api", "projects", "export", "--id", "7717", "-l", "fr", "--format", "po")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if res.Stdout != frPO || res.Stderr != "" {
		t.Errorf("stdout = %q, stderr = %q", res.Stdout, res.Stderr)
	}
	if got := fake.Downloads(); !reflect.DeepEqual(got, []string{"fr.po"}) {
		t.Errorf("downloads = %v", got)
	}
}

// --output writes the file (creating its directory), then leaves it alone
// when it would not change.
func TestExport_output(t *testing.T) {
	fake := exportFake(t)
	e := env(t, fake)

	res := e.Run("api", "projects", "export", "-p", "web", "-l", "fr", "--format", "po", "--output", "locales/fr.po")
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if res.Stdout != "" || res.Stderr != "created locales/fr.po\n" {
		t.Errorf("stdout = %q, stderr = %q", res.Stdout, res.Stderr)
	}
	if data, _ := os.ReadFile(filepath.Join(e.WorkDir, "locales", "fr.po")); string(data) != frPO {
		t.Errorf("file = %q", data)
	}

	res = e.Run("api", "projects", "export", "-p", "web", "-l", "fr", "--format", "po", "-o", "locales/fr.po", "--json")
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if res.Stderr != "unchanged locales/fr.po\n" {
		t.Errorf("stderr = %q", res.Stderr)
	}
	testutil.Golden(t, strings.ReplaceAll(res.Stdout, fake.FileURL("fr.po"), "<url>"))
}

// --json without --output describes the export and downloads nothing.
func TestExport_json(t *testing.T) {
	fake := exportFake(t)

	res := env(t, fake).Run("api", "projects", "export", "-p", "web", "-l", "fr", "--format", "po", "--json")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if n := len(fake.Downloads()); n != 0 {
		t.Errorf("%d downloads", n)
	}
	testutil.Golden(t, strings.ReplaceAll(res.Stdout, fake.FileURL("fr.po"), "<url>"))
}

func TestExport_badInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no language", []string{"--format", "po"}, `required flag(s) "language" not set`},
		{"no format", []string{"-l", "fr"}, `required flag(s) "format" not set`},
		{"an unknown filter", []string{"-l", "fr", "--format", "po", "--filter", "done"}, `--filter "done": use translated, untranslated`},
		{"a match mode without tags", []string{"-l", "fr", "--format", "po", "--tags-match", "all"}, "--tags-match needs --tag"},
		{"a bad match mode", []string{"-l", "fr", "--format", "po", "--tag", "x", "--tags-match", "none"}, `--tags-match "none": use all, any, exclude`},
		{"a bad order", []string{"-l", "fr", "--format", "po", "--order", "date"}, `--order "date": the only order is terms`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)

			res := env(t, fake).Run(append([]string{"api", "projects", "export", "-p", "web"}, tt.args...)...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d requests", n)
			}
		})
	}
}

func TestExport_errors(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/export", testutil.Fail(api.CodeLanguageNotInProject, "Project not containing specified language"))
	fake.Stub("languages/list", testutil.OK(map[string]any{"languages": []map[string]any{{"code": "en"}}}))

	res := env(t, fake).Run("api", "projects", "export", "-p", "web", "-l", "fr", "--format", "po")

	if want := "--language fr: project web has no such language (it has: en)"; res.Err == nil || res.Err.Error() != want || res.Stdout != "" {
		t.Errorf("error = %v, stdout = %q", res.Err, res.Stdout)
	}
	if res := testutil.Run(t, nil, "api", "projects", "export", "-p", "7717", "-l", "fr", "--format", "po"); !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("not logged in: %v", res.Err)
	}
}

// repoconfig may not import api, so it keeps its own copy of the modes.
func TestTagsMatchModes_sameList(t *testing.T) {
	if !reflect.DeepEqual(repoconfig.TagsMatchModes, api.TagsMatchModes) {
		t.Errorf("repoconfig %v, api %v", repoconfig.TagsMatchModes, api.TagsMatchModes)
	}
}
