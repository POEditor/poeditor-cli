package transfer_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/repoconfig"
	"github.com/POEditor/poeditor-cli/internal/transfer"
)

func TestItems(t *testing.T) {
	dir := t.TempDir()
	cfg := repoconfig.New(filepath.Join(dir, repoconfig.FileName), nil)
	p := repoconfig.Project{ID: 1, Alias: "web", Files: []repoconfig.File{
		{Path: "l/{lang}.json", Format: "json", LanguageMap: map[string]string{"pt-br": "pt-BR"}},
		{Path: "e/{lang}.po", Format: "po", Languages: []string{"en"}},
	}}
	langs := []string{"en", "fr", "pt-br"}

	display := func(items []transfer.Item) []string {
		var out []string
		for _, it := range items {
			out = append(out, it.Language+" "+it.Display)
		}
		return out
	}

	all, err := transfer.Items(cfg, p, langs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"en l/en.json", "fr l/fr.json", "pt-br l/pt-BR.json", "en e/en.po"}; !reflect.DeepEqual(display(all), want) {
		t.Errorf("all = %v, want %v", display(all), want)
	}
	if all[2].Path != filepath.Join(dir, "l", "pt-BR.json") {
		t.Errorf("path = %q", all[2].Path)
	}

	only, err := transfer.Items(cfg, p, langs, []string{"fr"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"fr l/fr.json"}; !reflect.DeepEqual(display(only), want) {
		t.Errorf("--language fr = %v", display(only))
	}

	if _, err := transfer.Items(cfg, p, langs, []string{"ja"}); err == nil ||
		err.Error() != "--language ja: project web has no such language (it has: en, fr, pt-br)" {
		t.Errorf("unknown language: %v", err)
	}
}

func TestJSONList(t *testing.T) {
	if got := transfer.JSONList([]string{"a", `b"c`}); got != `["a","b\"c"]` {
		t.Errorf("got %s", got)
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deep", "fr.json")

	steps := []struct {
		data   string
		dryRun bool
		want   string
		onDisk string // "" = file must not exist
	}{
		{"v1", true, transfer.Created, ""},
		{"v1", false, transfer.Created, "v1"},
		{"v1", false, transfer.Unchanged, "v1"},
		{"v2", true, transfer.Updated, "v1"},
		{"v2", false, transfer.Updated, "v2"},
	}
	for i, s := range steps {
		got, err := transfer.WriteFile(path, []byte(s.data), s.dryRun)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if got != s.want {
			t.Errorf("step %d: status %q, want %q", i, got, s.want)
		}
		data, err := os.ReadFile(path)
		if s.onDisk == "" {
			if err == nil {
				t.Errorf("step %d: dry run created the file", i)
			}
			continue
		}
		if string(data) != s.onDisk {
			t.Errorf("step %d: on disk %q, want %q", i, data, s.onDisk)
		}
	}

	leftovers, _ := filepath.Glob(filepath.Join(dir, "deep", ".poeditor-*"))
	if len(leftovers) != 0 {
		t.Errorf("temp files left behind: %v", leftovers)
	}
}

func TestWriteFile_keepsMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no Unix modes on Windows")
	}
	path := filepath.Join(t.TempDir(), "x.json")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := transfer.WriteFile(path, []byte("new"), false); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 600", info.Mode().Perm())
	}
}

func TestErrStopped(t *testing.T) {
	inner := os.ErrPermission
	err := error(&transfer.ErrStopped{Display: "l/fr.json", Err: inner})
	if err.Error() != "l/fr.json: "+inner.Error() || !transfer.IsStopped(err) || !strings.Contains(err.Error(), "permission") {
		t.Errorf("got %v", err)
	}
}
