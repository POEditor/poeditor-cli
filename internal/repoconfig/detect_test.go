package repoconfig_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

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

func TestDetect(t *testing.T) {
	root := t.TempDir()
	touch(t, root,
		"locales/en.json", "locales/fr.json", "locales/pt-BR.json",
		"app/src/main/res/values/strings.xml",
		"app/src/main/res/values-fr/strings.xml", "app/src/main/res/values-pt-rBR/strings.xml",
		"ios/en.lproj/Localizable.strings", "ios/fr.lproj/Localizable.strings",
		"docs/frontend.md",                 // "fr" inside a word: no match
		"node_modules/lib/locales/de.json", // dependencies are skipped
		".git/objects/de.json",             // hidden dirs are skipped
	)

	got := repoconfig.Detect(root, []string{"de", "en", "fr", "pt-br"})

	want := []repoconfig.Candidate{
		{Pattern: "locales/{lang}.json", Found: []string{"en", "fr", "pt-br"},
			LanguageMap: map[string]string{"pt-br": "pt-BR"}, Format: "key_value_json"},
		{Pattern: "app/src/main/res/values-{lang}/strings.xml", Found: []string{"fr", "pt-br"},
			LanguageMap: map[string]string{"pt-br": "pt-rBR"}, Format: "android_strings"},
		{Pattern: "ios/{lang}.lproj/Localizable.strings", Found: []string{"en", "fr"}, Format: "apple_strings"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect =\n%+v\nwant\n%+v", got, want)
	}
}

func TestDetect_nothing(t *testing.T) {
	root := t.TempDir()
	touch(t, root, "README.md", "src/main.go")

	if got := repoconfig.Detect(root, []string{"en", "fr"}); len(got) != 0 {
		t.Errorf("want no candidates, got %+v", got)
	}
}

func TestGuessFormat(t *testing.T) {
	tests := map[string]string{
		"locales/{lang}.json":                  "key_value_json",
		"res/values-{lang}/strings.xml":        "android_strings",
		"other/{lang}.xml":                     "",
		"{lang}.lproj/Localizable.strings":     "apple_strings",
		"{lang}.lproj/Localizable.stringsdict": "stringsdict",
		"i18n/{lang}.yaml":                     "yml",
		"po/{lang}.po":                         "po",
		"src/i18n/{lang}.ts":                   "", // TypeScript far more often than Qt
		"lib/l10n/app_{lang}.arb":              "arb",
	}
	for path, want := range tests {
		if got := repoconfig.GuessFormat(path); got != want {
			t.Errorf("GuessFormat(%q) = %q, want %q", path, got, want)
		}
	}
}
