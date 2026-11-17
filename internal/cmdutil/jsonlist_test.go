package cmdutil_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

type item struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func needsName(it item) error {
	if it.Name == "" {
		return errors.New(`no "name"`)
	}
	return nil
}

func TestReadJSONList(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []item
		wantErr string
	}{
		{"a list", `[{"name":"a","count":1},{"name":"b"}]`, []item{{"a", 1}, {"b", 0}}, ""},
		{"not JSON", `name,count`, nil, "--from-file in.json: invalid character"},
		{"not a list", `{"name":"a"}`, nil, "cannot unmarshal object"},
		{"an unknown field", `[{"name":"a","cuont":1}]`, nil, `unknown field "cuont"`},
		{"a wrong type", `[{"name":"a","count":"1"}]`, nil, "cannot unmarshal string"},
		{"empty", `[]`, nil, "--from-file in.json: the list is empty"},
		{"null", `null`, nil, "--from-file in.json: the list is empty"},
		{"two values", `[{"name":"a"}] [{"name":"b"}]`, nil, "--from-file in.json: more than one JSON value"},
		{"an item fails the check", `[{"name":"a"},{"count":2}]`, nil, `--from-file in.json: entry 2: no "name"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "in.json"), []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{IO: ios, WorkDir: dir}

			got, err := cmdutil.ReadJSONList(f, "in.json", needsName)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}

func TestReadJSONList_missingFileAndStdin(t *testing.T) {
	ios, stdin, _, _ := iostreams.Test()
	stdin.WriteString(`[{"name":"x"}]`)
	f := &cmdutil.Factory{IO: ios, WorkDir: t.TempDir()}

	if _, err := cmdutil.ReadJSONList[item](f, "nope.json", nil); err == nil || !strings.HasPrefix(err.Error(), "--from-file: open") {
		t.Errorf("missing file: error = %v", err)
	}
	got, err := cmdutil.ReadJSONList[item](f, "-", nil)
	if err != nil || len(got) != 1 || got[0].Name != "x" {
		t.Errorf("stdin: %+v, %v", got, err)
	}
}
