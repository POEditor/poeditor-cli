package projects_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

var projectLanguages = map[string]any{"languages": []map[string]any{{"code": "en"}, {"code": "fr"}}}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		form      map[string]string // "" = not sent
		languages bool              // languages/list called first
	}{
		{"name only", []string{"--name", " Web (2026) "},
			map[string]string{"name": "Web (2026)", "description": "", "reference_language": "", "fallback_language": "", "proofreading": "", "lock_proofread": ""}, false},
		{"clear the description and the fallback: no language check", []string{"--description", "", "--fallback-language", ""},
			map[string]string{"description": "", "fallback_language": "", "name": ""}, false},
		{"languages are checked", []string{"--reference-language", "en", "--fallback-language", "fr"},
			map[string]string{"reference_language": "en", "fallback_language": "fr"}, true},
		{"switches", []string{"--proofreading", "on", "--lock-proofread", "off"},
			map[string]string{"proofreading": "1", "lock_proofread": "0"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("projects/update", testutil.OK(json.RawMessage(webProject)))
			fake.Stub("languages/list", testutil.OK(projectLanguages))

			res := env(t, fake).Run(append([]string{"api", "projects", "update", "--project", "web"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			reqs := fake.Requests()
			if want := map[bool]int{false: 1, true: 2}[tt.languages]; len(reqs) != want {
				t.Fatalf("%d requests, want %d", len(reqs), want)
			}
			req := reqs[len(reqs)-1]
			if req.Endpoint != "projects/update" || req.Form.Get("id") != "7717" {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			for k, want := range tt.form {
				if got := req.Form.Get(k); got != want {
					t.Errorf("%s = %q, want %q", k, got, want)
				}
			}
			if res.Stderr != "Updated project web.\n" || res.Stdout != "" {
				t.Errorf("stdout = %q, stderr = %q", res.Stdout, res.Stderr)
			}
		})
	}
}

// Only the flags given are sent: an unset flag is no parameter at all.
func TestUpdate_onlyGivenFlags(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/update", testutil.OK(json.RawMessage(webProject)))

	res := env(t, fake).Run("api", "projects", "update", "--id", "7717", "--description", "", "--json")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	form := fake.OnlyRequest(t).Form
	form.Del("api_token")
	if len(form) != 2 || !form.Has("description") || form.Get("description") != "" {
		t.Errorf("form = %v, want id and an empty description", form)
	}
	testutil.Golden(t, res.Stdout)
}

func TestUpdate_badInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		want      string
		wantCalls int // languages/list only
	}{
		{"nothing to change", nil, "nothing to change: give --name", 0},
		{"an empty name", []string{"--name", " "}, "--name is empty", 0},
		{"proofreading yes", []string{"--proofreading", "yes"}, `--proofreading "yes": use on or off`, 0},
		{"lock 1", []string{"--lock-proofread", "1"}, `--lock-proofread "1": use on or off`, 0},
		{"a language the project lacks", []string{"--reference-language", "de"}, "--reference-language de: project web has no such language (it has: en, fr)", 1},
		{"reference checked before fallback", []string{"--fallback-language", "it", "--reference-language", "de"}, "--reference-language de:", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("languages/list", testutil.OK(projectLanguages))

			res := env(t, fake).Run(append([]string{"api", "projects", "update", "-p", "web"}, tt.args...)...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != tt.wantCalls {
				t.Errorf("%d requests, want %d (never projects/update)", n, tt.wantCalls)
			}
		})
	}
}

func TestUpdate_noLanguagesYet(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("languages/list", testutil.OK(map[string]any{"languages": []any{}}))

	res := env(t, fake).Run("api", "projects", "update", "-p", "web", "--reference-language", "en")

	if want := "--reference-language en: project web has no languages yet (add them with `poeditor api languages add`)"; res.Err == nil || res.Err.Error() != want {
		t.Errorf("error = %v", res.Err)
	}
}

func TestUpdate_apiErrors(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/update", testutil.Fail("4040", "The description can't contain links"))

	res := testutil.Run(t, fake, "api", "projects", "update", "-p", "7717", "--description", "http://x")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || res.Stderr != "" {
		t.Errorf("error = %v, stderr = %q", res.Err, res.Stderr)
	}
	if res := testutil.Run(t, nil, "api", "projects", "update", "-p", "7717", "--name", "x"); !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("not logged in: %v", res.Err)
	}
}
