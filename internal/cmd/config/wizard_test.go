package config_test

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/prompt"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

// remoteAccount is what the fake API shows for one token.
type remoteAccount struct {
	projects map[int]string   // id → name
	langs    map[int][]string // id → language codes
}

// formID is the request's project id as the API echoes it back: an int.
func formID(form url.Values) int {
	id, _ := strconv.Atoi(form.Get("id"))
	return id
}

func fakeWizardAPI(t *testing.T, byToken map[string]remoteAccount) *testutil.FakeAPI {
	t.Helper()
	fake := testutil.NewFakeAPI(t)
	fake.StubFunc("projects/list", func(form url.Values) testutil.Stub {
		var list []map[string]any
		for id, name := range byToken[form.Get("api_token")].projects {
			list = append(list, map[string]any{"id": id, "name": name})
		}
		return testutil.OK(map[string]any{"projects": list})
	})
	fake.StubFunc("projects/view", func(form url.Values) testutil.Stub {
		return testutil.OK(map[string]any{"project": map[string]any{"id": formID(form), "reference_language": "en"}})
	})
	fake.StubFunc("languages/list", func(form url.Values) testutil.Stub {
		acc := byToken[form.Get("api_token")]
		var langs []map[string]any
		for id, codes := range acc.langs {
			if form.Get("id") == strconv.Itoa(id) {
				for _, c := range codes {
					langs = append(langs, map[string]any{"code": c})
				}
			}
		}
		return testutil.OK(map[string]any{"languages": langs})
	})
	return fake
}

func touch(t *testing.T, root string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func readRepoConfig(t *testing.T, env *testutil.Env) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(env.WorkDir, repoconfig.FileName))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

var personal = map[string]remoteAccount{
	"p-tok": {
		projects: map[int]string{4321: "Website", 7717: "Mobile app"},
		langs:    map[int][]string{4321: {"en", "fr", "pt-br"}, 7717: {"en", "de"}},
	},
}

func TestInit_fullWalkthrough(t *testing.T) {
	env := testutil.NewEnv(t, fakeWizardAPI(t, personal)).Login("personal", "p-tok")
	touch(t, env.WorkDir, "locales/en.json", "locales/fr.json", "locales/pt-BR.json")
	env.Answer(
		"Website (4321)",      // project
		testutil.Default,      // alias: "website"
		testutil.Default,      // source language: en (the project's reference language)
		"locales/{lang}.json", // file found in the repo
		testutil.Default,      // format: guessed key_value_json
		"web, shared",         // pull tags
		"any:",                // tags_match
		"web",                 // push tags (all)
		"y",                   // more push tag options
		"needs-review",        // new terms
		"removed",             // obsolete terms
		"",                    // overwritten translations: none
		"y",                   // another file
		"Type a path",         // not one of the found ones
		"bad.json",            // rejected: no {lang}
		"emails/{lang}.po",
		testutil.Default, // format: guessed po
		"", "",           // no pull tags, no push tags
		"n", // no more push tag options
		"n",
	)

	res := env.Run("config", "init")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	testutil.Golden(t, readRepoConfig(t, env))
	if env.Prompter.Rejected[0] != "put {lang} where the language code goes, e.g. locales/{lang}.json" {
		t.Errorf("rejected = %v", env.Prompter.Rejected)
	}
	if len(env.Prompter.Answers) != 0 {
		t.Errorf("unused answers: %v", env.Prompter.Answers)
	}
	for _, want := range []string{"Account: personal\n", "Wrote ", "(project website, 2 files)", "Next: `poeditor config check`"} {
		if !strings.Contains(res.Stderr, want) {
			t.Errorf("stderr lacks %q:\n%s", want, res.Stderr)
		}
	}
	if res.Stdout != "" {
		t.Errorf("stdout = %q", res.Stdout)
	}
}

func TestInit_chooseAmongAccounts(t *testing.T) {
	remote := map[string]remoteAccount{
		"p-tok": personal["p-tok"],
		"c-tok": {projects: map[int]string{9001: "Client app"}, langs: map[int][]string{9001: {"en", "ro"}}},
	}
	env := testutil.NewEnv(t, fakeWizardAPI(t, remote)).Login("personal", "p-tok").Login("client-a", "c-tok")
	env.Answer("client-a", "Client app", testutil.Default, testutil.Default,
		"app/{lang}.json", testutil.Default, "", "", "n", "n")

	res := env.Run("config", "init")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if got := env.Prompter.Offered["Which account is the project in?"]; !reflect.DeepEqual(got, []string{"client-a", "personal (default)"}) {
		t.Errorf("account options = %v", got)
	}
	if body := readRepoConfig(t, env); !strings.Contains(body, "    account: client-a\n") || !strings.Contains(body, "id: 9001") {
		t.Errorf("config:\n%s", body)
	}
}

func TestInit_nameValidation(t *testing.T) {
	env := testutil.NewEnv(t, fakeWizardAPI(t, personal)).Login("personal", "p-tok")
	env.Answer("Mobile app", "1234", "my app", "mobile", testutil.Default,
		"m/{lang}.json", testutil.Default, "", "", "n", "n")

	res := env.Run("config", "init")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	want := []string{"must not be all digits: it would be mistaken for a project id", "no spaces"}
	if !reflect.DeepEqual(env.Prompter.Rejected, want) {
		t.Errorf("rejected = %v, want %v", env.Prompter.Rejected, want)
	}
	if body := readRepoConfig(t, env); !strings.Contains(body, "alias: mobile") || strings.Contains(body, "account:") {
		t.Errorf("config:\n%s", body)
	}
}

func TestInit_refusals(t *testing.T) {
	t.Run("file exists", func(t *testing.T) {
		env := testutil.NewEnv(t, nil).Login("personal", "p").WriteRepoConfig(twoProjects).Answer()

		res := env.Run("config", "init")

		if res.Err == nil || !strings.Contains(res.Err.Error(), "already exists: use `poeditor config add`") {
			t.Fatalf("error = %v", res.Err)
		}
		if len(env.Prompter.Asked) != 0 {
			t.Errorf("asked %v", env.Prompter.Asked)
		}
	})

	t.Run("not interactive", func(t *testing.T) {
		res := testutil.NewEnv(t, nil).Login("personal", "p").Run("config", "init")

		if !errors.Is(res.Err, cmdutil.ErrNotInteractive) {
			t.Fatalf("error = %v", res.Err)
		}
	})

	t.Run("not logged in", func(t *testing.T) {
		res := testutil.NewEnv(t, nil).Answer().Run("config", "init")

		if res.Err == nil || res.Err.Error() != "log in first: run `poeditor auth login`" {
			t.Fatalf("error = %v", res.Err)
		}
	})

	t.Run("nothing written when cancelled", func(t *testing.T) {
		env := testutil.NewEnv(t, fakeWizardAPI(t, personal)).Login("personal", "p-tok")
		env.Answer("Website", testutil.Default, testutil.Default, testutil.Cancel)

		res := env.Run("config", "init")

		if !errors.Is(res.Err, prompt.ErrCancelled) {
			t.Fatalf("error = %v", res.Err)
		}
		if _, err := os.Stat(filepath.Join(env.WorkDir, repoconfig.FileName)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("a config was written: %v", err)
		}
	})
}

func TestAdd_project(t *testing.T) {
	env := testutil.NewEnv(t, fakeWizardAPI(t, personal)).Login("personal", "p-tok")
	env.WriteRepoConfig("# keep me\nprojects:\n  - {id: 4321, alias: website, source_language: en, files: [{path: \"l/{lang}.json\", format: json}]}\n")
	// no translation files in the directory: the wizard goes straight to typing a path
	env.Answer("Another project", "Mobile app", testutil.Default, testutil.Default,
		"l/{lang}.json", "mobile/{lang}.json", testutil.Default, "", "", "n", "n")

	res := env.Run("config", "add")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if got := env.Prompter.Offered["Which project?"]; !reflect.DeepEqual(got, []string{"Mobile app (7717)"}) {
		t.Errorf("projects offered = %v (a project already in the file must not be offered)", got)
	}
	if !reflect.DeepEqual(env.Prompter.Rejected, []string{"this path is already in .poeditor.yml"}) {
		t.Errorf("rejected = %v", env.Prompter.Rejected)
	}
	body := readRepoConfig(t, env)
	if !strings.HasPrefix(body, "# keep me\n") || !strings.Contains(body, "id: 7717") {
		t.Errorf("config:\n%s", body)
	}
	if !strings.Contains(res.Stderr, "Added project mobile-app (1 file)") {
		t.Errorf("stderr = %q", res.Stderr)
	}
}

func TestAdd_fileToExistingProject(t *testing.T) {
	env := testutil.NewEnv(t, fakeWizardAPI(t, personal)).Login("personal", "p-tok")
	env.WriteRepoConfig(`projects:
  - {id: 4321, alias: website, source_language: en, files: [{path: "l/{lang}.json", format: json}]}
  - {id: 7717, alias: mobile, source_language: en, files: [{path: "m/{lang}.json", format: json}]}
`)
	touch(t, env.WorkDir, "android/values-de/strings.xml", "android/values-en/strings.xml")
	env.Answer("A file to an existing project", "mobile", "android/values-{lang}/strings.xml", testutil.Default, "", "", "n")

	res := env.Run("config", "add")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	cfg, err := repoconfig.Load(filepath.Join(env.WorkDir, repoconfig.FileName))
	if err != nil {
		t.Fatal(err)
	}
	files := cfg.Projects[1].Files
	if len(files) != 2 || files[1].Path != "android/values-{lang}/strings.xml" || files[1].Format != "android_strings" {
		t.Errorf("mobile files = %+v", files)
	}
	if len(cfg.Projects[0].Files) != 1 {
		t.Errorf("website files changed: %+v", cfg.Projects[0].Files)
	}
}

func TestAdd_refusesBrokenConfig(t *testing.T) {
	env := testutil.NewEnv(t, nil).Login("personal", "p")
	env.WriteRepoConfig("projects:\n  - {id: 1, files: [{path: \"a.json\", format: json}]}\n").Answer()

	res := env.Run("config", "add")

	if res.Err == nil || !strings.HasPrefix(res.Err.Error(), "fix ") {
		t.Fatalf("error = %v", res.Err)
	}
	if len(env.Prompter.Asked) != 0 {
		t.Errorf("asked %v", env.Prompter.Asked)
	}
}

func TestInit_morePushTagsOnlyWhenAsked(t *testing.T) {
	env := testutil.NewEnv(t, fakeWizardAPI(t, personal)).Login("personal", "p-tok")
	env.Answer("Mobile app", testutil.Default, testutil.Default, "m/{lang}.json", testutil.Default,
		"", "", "y", "", "", "changed", "n")

	res := env.Run("config", "init")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	body := readRepoConfig(t, env)
	if !strings.Contains(body, "push:\n          tags:\n            overwritten_translations:\n              - changed\n") {
		t.Errorf("only the overwritten group should be written:\n%s", body)
	}
	for _, unwanted := range []string{"all:", "new:", "obsolete:"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("%s written although left empty:\n%s", unwanted, body)
		}
	}
	want := []string{
		"Tag every pushed term with? (comma-separated; empty for none)",
		"Also tag new, obsolete or overwritten terms on push?",
		"Tag terms that push creates (new terms)? (comma-separated; empty for none)",
		"Tag terms that are in POEditor but no longer in the file (obsolete)? (comma-separated; empty for none)",
		"Tag terms whose translations push overwrites? (comma-separated; empty for none)",
	}
	asked := strings.Join(env.Prompter.Asked, "\n")
	for _, w := range want {
		if !strings.Contains(asked, w) {
			t.Errorf("never asked %q; asked:\n%s", w, asked)
		}
	}
}
