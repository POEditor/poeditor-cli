package projects_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

const syncFile = `[{"term":"save","context":"button","tags":["web"]},{"term":"%d files","plural":"%d files"}]`

func TestSync_needsYes(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	e := testutil.NewEnv(t, fake) // not logged in
	writeFile(t, e, "terms.json", syncFile)

	res := e.Run("api", "projects", "sync", "terms.json", "--project", "web")

	want := "syncing project web with the 2 terms of terms.json deletes every other term, with its translations; nothing was sent. To sync, run:\n  poeditor api projects sync terms.json --project web --yes"
	if res.Err == nil || res.Err.Error() != want {
		t.Errorf("error = %v\nwant    %s", res.Err, want)
	}
	if n := len(fake.Requests()); n != 0 {
		t.Errorf("%d requests sent without --yes", n)
	}
}

func TestSync(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		json bool
	}{
		{"file", []string{"terms.json", "-p", "web", "--yes"}, false},
		{"stdin, json", []string{"-", "--id", "7717", "--yes", "--json"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("projects/sync", testutil.OK(map[string]any{"terms": map[string]int{"parsed": 2, "added": 1, "updated": 1, "deleted": 7}}))
			e := env(t, fake)
			writeFile(t, e, "terms.json", syncFile)
			e.Stdin = syncFile

			res := e.Run(append([]string{"api", "projects", "sync"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "projects/sync" || req.Form.Get("id") != "7717" || req.Form.Has("yes") {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			assertJSON(t, req.Form.Get("data"), `[{"term":"save","context":"button","tags":["web"]},{"term":"%d files","context":"","plural":"%d files"}]`)
			if res.Stderr != "Synced project web: 2 terms parsed, 1 added, 1 updated, 7 deleted.\n" {
				t.Errorf("stderr = %q", res.Stderr)
			}
			if tt.json {
				testutil.Golden(t, res.Stdout)
			} else if res.Stdout != "" {
				t.Errorf("stdout = %q", res.Stdout)
			}
		})
	}
}

// A row without a term would make POEditor delete every term: refused before
// anything, with --yes or without.
func TestSync_badFile(t *testing.T) {
	tests := []struct {
		name, file, want string
	}{
		{"a row without term", `[{"term":"a"},{"context":"b"}]`, `terms.json: entry 2: no "term"`},
		{"only blank terms", `[{"term":" "}]`, `entry 1: no "term"`},
		{"empty", `[]`, "the list is empty"},
		{"an unknown field", `[{"term":"a","tag":"x"}]`, `unknown field "tag"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			e := env(t, fake)
			writeFile(t, e, "terms.json", tt.file)

			res := e.Run("api", "projects", "sync", "terms.json", "-p", "web", "--yes")

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d requests", n)
			}
		})
	}
}

func TestSync_errors(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/sync", testutil.Fail("4033", "Currently processing an uploaded file. Please try again later."))
	e := env(t, fake)
	writeFile(t, e, "terms.json", syncFile)

	res := e.Run("api", "projects", "sync", "terms.json", "-p", "web", "--yes")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4033" || res.Stderr != "" {
		t.Errorf("error = %v, stderr = %q", res.Err, res.Stderr)
	}
	ne := testutil.NewEnv(t, nil)
	writeFile(t, ne, "terms.json", syncFile)
	if res := ne.Run("api", "projects", "sync", "terms.json", "-p", "7717", "--yes"); !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("not logged in: %v", res.Err)
	}
	empty := testutil.NewFakeAPI(t)
	for _, args := range [][]string{{"api", "projects", "sync", "terms.json", "--yes"}, {"api", "projects", "sync", "-p", "web"}} {
		if res := env(t, empty).Run(args...); res.Err == nil {
			t.Errorf("%v: no error", args)
		}
	}
}
