package translations_test

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

// translations/automatic as the API documents it: pairs in the provider's
// codes, a failed one possibly without target.
const (
	allTranslated = `{"successful":[{"source":"en","target":"fr","chars":1204},{"source":"en","target":"pt-BR","chars":1180}],"failed":[]}`
	someFailed    = `{"successful":[{"source":"en","target":"fr","chars":1204}],"failed":[{"source":"en","target":null,"message":"Invalid pair."},{"source":"en","target":"ar","message":"Not enough automatic translation characters."}]}`
	// The API fails a pair with nothing left to translate; the CLI does not
	// (a CI job that runs on every push must not fail when nothing is new).
	nothingNew      = `{"successful":[],"failed":[{"source":"EN","target":"FR","message":"No untranslated terms found."}]}`
	nothingNewMixed = `{"successful":[],"failed":[{"source":"EN","target":"FR","message":"No untranslated terms found."},{"source":"EN","target":"DE","message":"Target language not found in the project."}]}`
)

func TestAutomatic_request(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want url.Values // besides id and api_token
	}{
		{"provider codes default to the project's", []string{"--provider", "deepl", "--source-language", "en", "--target", "fr", "--target", "pt-br=pt-BR"}, url.Values{
			"provider": {"deepl"}, "source_language": {"en"}, "provider_source_language": {"en"},
			"target_languages": {`[{"project_language":"fr","provider_language":"fr"},{"project_language":"pt-br","provider_language":"pt-BR"}]`},
		}},
		{"tags, match mode, options", []string{"--provider", "google", "--source-language", "en-us=en", "--target", "fr", "--tag", "new", "--tag", "web", "--tags-match", "all", "--option", "format=html", "--option", "formality=less"}, url.Values{
			"provider": {"google"}, "source_language": {"en-us"}, "provider_source_language": {"en"},
			"target_languages": {`[{"project_language":"fr","provider_language":"fr"}]`},
			"options":          {`{"format":"html","formality":"less","tags":["new","web"],"tags_match":"all"}`},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("translations/automatic", testutil.OK(json.RawMessage(allTranslated)))

			res := env(t, fake).Run(append([]string{"api", "translations", "automatic", "-p", "web"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			got := fake.OnlyRequest(t).Form
			if got.Get("id") != "7717" || got.Get("api_token") != testutil.TestToken {
				t.Errorf("form = %v", got)
			}
			for k, v := range tt.want {
				if k == "options" {
					assertData(t, got.Get(k), v[0])
				} else if got.Get(k) != v[0] {
					t.Errorf("%s = %q, want %q", k, got.Get(k), v[0])
				}
			}
			if _, sent := tt.want["options"]; !sent && got.Has("options") {
				t.Errorf("options sent: %s", got.Get("options"))
			}
		})
	}
}

func TestAutomatic_output(t *testing.T) {
	tests := []struct {
		name    string
		result  string
		args    []string
		tty     bool
		wantErr string
	}{
		{"table", allTranslated, nil, true, ""},
		{"table_piped", allTranslated, nil, false, ""},
		{"table_failed", someFailed, nil, true, "2 of 3 language pairs failed"},
		{"json", allTranslated, []string{"--json"}, true, ""},
		{"json_failed", someFailed, []string{"--json"}, true, "2 of 3 language pairs failed"},
		{"nothing_new", nothingNew, nil, true, ""},
		{"nothing_new_and_failed", nothingNewMixed, nil, true, "1 of 2 language pairs failed"},
		{"json_nothing_new", nothingNew, []string{"--json"}, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("translations/automatic", testutil.OK(json.RawMessage(tt.result)))
			e := env(t, fake)
			e.TTY = tt.tty

			res := e.Run(append([]string{"api", "translations", "automatic", "--id", "7717", "--provider", "google", "--source-language", "en", "--target", "fr"}, tt.args...)...)

			if tt.wantErr == "" && res.Err != nil {
				t.Fatal(res.Err)
			}
			if tt.wantErr != "" && (res.Err == nil || res.Err.Error() != tt.wantErr) {
				t.Errorf("error = %v, want %q", res.Err, tt.wantErr)
			}
			testutil.Golden(t, res.Stdout)
		})
	}
}

func TestAutomatic_badInput(t *testing.T) {
	base := []string{"api", "translations", "automatic", "-p", "web"}
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"unknown provider", []string{"--provider", "babel", "--source-language", "en", "--target", "fr"}, `--provider "babel": use google, microsoft, deepl`},
		{"no provider", []string{"--source-language", "en", "--target", "fr"}, `required flag(s) "provider" not set`},
		{"no target", []string{"--provider", "google", "--source-language", "en"}, `required flag(s) "target" not set`},
		{"empty provider code", []string{"--provider", "google", "--source-language", "en", "--target", "pt-br="}, `--target "pt-br=": use a language code, or code=provider-code`},
		{"no source code", []string{"--provider", "google", "--source-language", "=en", "--target", "fr"}, `--source-language "=en"`},
		{"bad match mode", []string{"--provider", "google", "--source-language", "en", "--target", "fr", "--tag", "x", "--tags-match", "some"}, `--tags-match "some": use all, any, exclude`},
		{"match mode without tags", []string{"--provider", "google", "--source-language", "en", "--target", "fr", "--tags-match", "all"}, "--tags-match needs --tag"},
		{"option without =", []string{"--provider", "google", "--source-language", "en", "--target", "fr", "--option", "html"}, `--option "html": use key=value`},
		{"tags as an option", []string{"--provider", "google", "--source-language", "en", "--target", "fr", "--option", "tags=x"}, "--option tags: use --tag and --tags-match"},
		{"an argument", []string{"fr", "--provider", "google", "--source-language", "en", "--target", "fr"}, `unknown command "fr"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)

			res := env(t, fake).Run(append(base, tt.args...)...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d API requests, want none", n)
			}
		})
	}
}

func TestAutomatic_apiFailure(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("translations/automatic", testutil.Fail("4040", "Unknown tag: nwe"))

	res := testutil.Run(t, fake, "api", "translations", "automatic", "-p", "7717", "--provider", "google", "--source-language", "en", "--target", "fr", "--tag", "nwe")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4040" || apiErr.Message != "Unknown tag: nwe" {
		t.Fatalf("got %v", res.Err)
	}
	if res.Stdout != "" {
		t.Errorf("stdout = %q", res.Stdout)
	}
}

func TestAutomatic_notLoggedIn(t *testing.T) {
	res := testutil.Run(t, nil, "api", "translations", "automatic", "-p", "7717", "--provider", "google", "--source-language", "en", "--target", "fr")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("error = %v, want ErrNotLoggedIn", res.Err)
	}
}
