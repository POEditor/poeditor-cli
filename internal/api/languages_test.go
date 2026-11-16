package api_test

import (
	"context"
	"net/url"
	"reflect"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/api"
)

func TestLanguagesList(t *testing.T) {
	var rec recorder
	client := rec.client(t, `{"languages":[{"name":"French","code":"fr","translations":10,"percentage":83.33,"updated":"2024-01-02T03:04:05+0000"}]}`)

	langs, err := client.LanguagesList(context.Background(), 7717)

	if err != nil {
		t.Fatal(err)
	}
	rec.assertForm(t, "languages/list", url.Values{"id": {"7717"}})
	if want := []api.Language{{LanguageRef: api.LanguageRef{Name: "French", Code: "fr"}, Translations: 10}}; !reflect.DeepEqual(langs, want) {
		t.Errorf("languages = %+v", langs)
	}
}

func TestLanguageCodes_sorted(t *testing.T) {
	var rec recorder
	client := rec.client(t, `{"languages":[{"code":"fr"},{"code":"de"},{"code":"en"}]}`)

	codes, err := client.LanguageCodes(context.Background(), 7717)

	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"de", "en", "fr"}; !reflect.DeepEqual(codes, want) {
		t.Errorf("codes = %v", codes)
	}
}
