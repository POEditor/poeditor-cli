package raw_test

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmd/api/raw"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestMain(m *testing.M) { testutil.Main(m) }

// The fixtures' shapes need not match their endpoints: a raw command prints
// whatever comes back. Raw results keep the key order the API sends, which the table must follow.
const (
	projectsResult = `{"projects":[` +
		`{"id":7717,"name":"Mobile app","public":0,"created":"2026-01-14T09:12:44+0000"},` +
		`{"id":4321,"name":"Web\tsite","public":1,"created":"2025-06-02T17:03:10+0000","tags":["web","v2"],"owner":null}]}`
	projectResult = `{"project":{"id":7717,"name":"Web","reference_language":"en","terms":12,"settings":{"a":1}}}`
	statsResult   = `{"terms":{"parsed":3,"added":2}}`
	codesResult   = `{"codes":["en","fr"]}`
	urlResult     = `{"url":"https://api.poeditor.com/v2/download/file/abc"}`
	emptyResult   = `{"projects":[]}`
	// a date-time splits over two lines on a terminal, without a UTC offset
	// (another offset is kept); text that only starts with one does not.
	// A list of plain values (tags) is joined with ", " on a terminal.
	termsResult = `{"terms":[` +
		`{"term":"release_note","context":"","created":"2026-01-14T09:12:44+0000","updated":""},` +
		`{"term":"2026-01-14T09:12:44+0000 was the release","context":"home","created":"2025-06-02T17:03:10+0000","updated":"2026-03-01T08:00:00+0000"},` +
		`{"term":"offset","context":"","created":"2026-02-01T10:00:00+0200","updated":"2026-02-02T10:00:00Z"},` +
		`{"term":"tagged","context":"","tags":["web","v2, beta"]},` +
		`{"term":"nested","context":"","tags":[{"a":1},["b"]]}]}`
)

func TestAPI_output(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		result string // "" = the endpoint returns no result
		tty    bool
	}{
		{"list", []string{"api", "contributors", "list"}, projectsResult, true},
		{"list_piped", []string{"api", "contributors", "list"}, projectsResult, false},
		{"terms", []string{"api", "contributors", "list", "--id", "7717"}, termsResult, true},
		{"terms_piped", []string{"api", "contributors", "list", "--id", "7717"}, termsResult, false},
		{"list_empty", []string{"api", "contributors", "list"}, emptyResult, true},
		{"list_empty_piped", []string{"api", "contributors", "list"}, emptyResult, false},
		{"object", []string{"api", "projects", "view", "-f", "id=7717"}, projectResult, true},
		{"object_piped", []string{"api", "projects", "view", "-f", "id=7717"}, projectResult, false},
		{"stats", []string{"api", "terms", "add", "-f", "id=7717"}, statsResult, true},
		{"values", []string{"api", "contributors", "add"}, codesResult, true},
		{"scalar", []string{"api", "projects", "export", "-f", "id=7717"}, urlResult, true},
		{"json", []string{"api", "projects", "view", "-f", "id=7717", "--json"}, projectResult, true},
		{"envelope", []string{"api", "projects", "view", "-f", "id=7717", "--envelope"}, projectResult, true},
		{"no_result", []string{"api", "projects", "delete", "-f", "id=7717", "--yes"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			endpoint := tt.args[1] + "/" + tt.args[2]
			var result any
			if tt.result != "" {
				result = json.RawMessage(tt.result)
			}
			fake.Stub(endpoint, testutil.OK(result))
			env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
			env.TTY = tt.tty

			res := env.Run(tt.args...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != endpoint || req.Form.Get("api_token") != testutil.TestToken {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			if res.Stderr != "" {
				t.Errorf("stderr = %q", res.Stderr)
			}
			testutil.Golden(t, res.Stdout)
		})
	}
}

func TestAPI_output_narrowTerminal(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("contributors/list", testutil.OK(json.RawMessage(projectsResult)))
	env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
	env.Width = 60

	res := env.Run("api", "contributors", "list")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	for _, line := range testutil.Lines(res.Stdout) {
		if w := len([]rune(line)); w > 60 {
			t.Errorf("line is %d columns wide, terminal has 60: %q", w, line)
		}
	}
	testutil.Golden(t, res.Stdout)
}

func TestAPI_fields(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("terms/add", testutil.OK(map[string]any{"terms": map[string]int{"parsed": 1, "added": 1}}))
	env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
	terms := `[{"term":"Hello","context":""}]`
	if err := os.WriteFile(filepath.Join(env.WorkDir, "terms.json"), []byte(terms), 0o644); err != nil {
		t.Fatal(err)
	}
	env.Stdin = "from stdin"

	res := env.Run("api", "terms", "add", "-f", "id=7717", "-f", "tags=a", "--field", "tags=b=c", "-F", "data=terms.json", "--field-file", "note=-")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	req := fake.OnlyRequest(t)
	want := url.Values{"api_token": {testutil.TestToken}, "id": {"7717"}, "tags": {"a", "b=c"}, "data": {terms}, "note": {"from stdin"}}
	if req.Endpoint != "terms/add" || !reflect.DeepEqual(req.Form, want) {
		t.Errorf("request = %s %v\nwant %v", req.Endpoint, req.Form, want)
	}
}

// Every public endpoint is exactly one command under `poeditor api`; each raw
// one sends its own endpoint, with every flag of the raw commands.
func TestAPI_everyEndpoint(t *testing.T) {
	commands := map[string]*cobra.Command{}
	for _, cmd := range testutil.Leaves() {
		commands[cmd.CommandPath()] = cmd
	}
	ran := 0
	for _, endpoint := range raw.Endpoints {
		group, action, _ := strings.Cut(endpoint, "/")
		args := []string{"api", group, raw.CommandName(action)}
		cmd := commands["poeditor "+strings.Join(args, " ")]
		if cmd == nil {
			t.Errorf("%s: no command `poeditor %s`", endpoint, strings.Join(args, " "))
			continue
		}
		if !strings.HasSuffix(testutil.CommandPackage(cmd), "/internal/cmd/api/raw") {
			continue // a dedicated command, tested in its own package
		}
		t.Run(endpoint, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub(endpoint, testutil.OK(json.RawMessage(`{"ok":1}`)))
			env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
			for name, content := range map[string]string{"data.json": "[]", "en.json": "{}"} {
				if err := os.WriteFile(filepath.Join(env.WorkDir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			runArgs := append(args, "--id", "7717", "-f", "a=1", "-F", "data=data.json", "--upload", "en.json", "--json", "--envelope")
			if cmd.Flags().Lookup("yes") != nil {
				runArgs = append(runArgs, "--yes")
			}

			res := env.Run(runArgs...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != endpoint || req.Form.Get("id") != "7717" || req.Form.Get("a") != "1" || req.Form.Get("data") != "[]" || req.FileName != "en.json" || req.Form.Has("yes") {
				t.Errorf("request = %s %v, file %q", req.Endpoint, req.Form, req.FileName)
			}
		})
		ran++
	}
	if ran == 0 {
		t.Fatal("no raw command found")
	}
}

// Raw commands that delete data send nothing without --yes, and say how to
// send it; with --yes the request goes out (yes itself is not a parameter).
func TestAPI_deletesNeedYes(t *testing.T) {
	// the deleting endpoints that still have a raw command (a dedicated one tests its own --yes)
	deleting := map[string][]string{
		"projects/delete":     {"api", "projects", "delete"},
		"projects/sync":       {"api", "projects", "sync"},
		"terms/delete":        {"api", "terms", "delete"},
		"translations/delete": {"api", "translations", "delete"},
		"contributors/remove": {"api", "contributors", "remove"},
	}
	for endpoint, args := range deleting {
		t.Run(endpoint, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub(endpoint, testutil.OK(nil))
			params := []string{"--id", "7717", "--data", `[{"term":"it's"}]`}

			res := testutil.Run(t, fake, append(args, params...)...)

			want := endpoint + " deletes data and cannot be undone; nothing was sent. To send it, run:\n" +
				"  poeditor " + strings.Join(args, " ") + ` --id 7717 --data '[{"term":"it'\''s"}]' --yes`
			if res.Err == nil || res.Err.Error() != want {
				t.Errorf("err = %v\nwant  %s", res.Err, want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Fatalf("%d requests sent without --yes", n)
			}

			res = testutil.Run(t, fake, append(args, append(params, "--yes")...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Form.Get("id") != "7717" || req.Form.Has("yes") {
				t.Errorf("form = %v", req.Form)
			}
		})
	}
	// the others have no --yes: it would be an API parameter
	res := testutil.Run(t, nil, "api", "projects", "view", "--help")
	if strings.Contains(res.Stdout, "--yes") {
		t.Error("projects view offers --yes")
	}
}

func TestAPI_parameterFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want url.Values // without api_token
	}{
		{"value as next argument", []string{"--id", "7717"}, url.Values{"id": {"7717"}}},
		{"value after =", []string{"--id=7717"}, url.Values{"id": {"7717"}}},
		{"dashes become underscores", []string{"--id", "1", "--reference-language", "en"}, url.Values{"id": {"1"}, "reference_language": {"en"}}},
		{"alone sends 1", []string{"--overwrite", "--id", "1"}, url.Values{"overwrite": {"1"}, "id": {"1"}}},
		{"alone at the end sends 1", []string{"--id", "1", "--overwrite"}, url.Values{"id": {"1"}, "overwrite": {"1"}}},
		{"negative value with =", []string{"--offset=-1"}, url.Values{"offset": {"-1"}}},
		{"empty value with =", []string{"--description="}, url.Values{"description": {""}}},
		{"repeated", []string{"--tags", "a", "--tags", "b"}, url.Values{"tags": {"a", "b"}}},
		{"mixed with --field", []string{"--id", "1", "-f", "name=x", "--field", "id=2"}, url.Values{"id": {"1", "2"}, "name": {"x"}}},
		{"our flags keep their meaning", []string{"--id", "1", "--json"}, url.Values{"id": {"1"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("projects/update", testutil.OK(nil))

			res := testutil.Run(t, fake, append([]string{"api", "projects", "update"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			got := fake.OnlyRequest(t).Form
			got.Del("api_token")
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("params = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPI_argumentPlacement(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"global flag before the endpoint", []string{"api", "--account", "client-a", "projects", "view", "--id", "7"}},
		{"global flag after the parameters", []string{"api", "projects", "view", "--id", "7", "--account=client-a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("projects/view", testutil.OK(nil))
			env := testutil.NewEnv(t, fake).Login("personal", "tok-personal").Login("client-a", "tok-client")

			res := env.Run(tt.args...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "projects/view" || req.Form.Get("id") != "7" || req.Form.Get("api_token") != "tok-client" {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
		})
	}
}

func TestAPI_help(t *testing.T) {
	for _, args := range [][]string{{"api", "projects", "view", "--help"}, {"api", "projects", "view", "-h"}} {
		fake := testutil.NewFakeAPI(t)

		res := testutil.Run(t, fake, args...)

		if res.Err != nil || !strings.Contains(res.Stdout, "API parameters are flags named after them") {
			t.Errorf("%v: err = %v, stdout = %.80q", args, res.Err, res.Stdout)
		}
		if n := len(fake.Requests()); n != 0 {
			t.Errorf("%v: %d API calls, want none", args, n)
		}
	}
}

func TestAPI_upload(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/upload", testutil.OK(map[string]any{"terms": map[string]int{"parsed": 1}}))
	env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
	if err := os.MkdirAll(filepath.Join(env.WorkDir, "locales"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.WorkDir, "locales", "en.json"), []byte(`{"a":"b"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	res := env.Run("api", "projects", "upload", "-f", "id=7717", "-f", "updating=terms", "--upload", "locales/en.json")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	req := fake.OnlyRequest(t)
	if req.FileName != "en.json" || string(req.FileData) != `{"a":"b"}` {
		t.Errorf("file = %q %q", req.FileName, req.FileData)
	}
	if req.Form.Get("updating") != "terms" || req.Form.Get("api_token") != testutil.TestToken {
		t.Errorf("form = %v", req.Form)
	}
}

func TestAPI_account(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("contributors/list", testutil.OK(map[string]any{"languages": []any{}}))
	env := testutil.NewEnv(t, fake).Login("personal", "tok-personal").Login("client-a", "tok-client")

	res := env.Run("api", "contributors", "list", "--id", "7", "--account", "client-a")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if got := fake.OnlyRequest(t).Form.Get("api_token"); got != "tok-client" {
		t.Errorf("token = %q", got)
	}
}

func TestAPI_apiFailure(t *testing.T) {
	for _, tt := range []struct {
		name       string
		args       []string
		wantStdout bool
	}{
		{"result", []string{"api", "projects", "view", "-f", "id=1"}, false},
		{"envelope", []string{"api", "projects", "view", "-f", "id=1", "--envelope"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("projects/view", testutil.Fail("4012", "Invalid project"))

			res := testutil.Run(t, fake, tt.args...)

			var apiErr *api.Error
			if !errors.As(res.Err, &apiErr) || apiErr.Code != "4012" {
				t.Fatalf("err = %v", res.Err)
			}
			if tt.wantStdout {
				testutil.Golden(t, res.Stdout)
			} else if res.Stdout != "" {
				t.Errorf("stdout = %q", res.Stdout)
			}
		})
	}
}

func TestAPI_notLoggedIn(t *testing.T) {
	res := testutil.Run(t, nil, "api", "contributors", "list", "--id", "7")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Fatalf("err = %v", res.Err)
	}
}

func TestAPI_badArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"an argument", []string{"api", "projects", "view", "7717"}, `unexpected argument "7717": API parameters are given as --name value`},
		{"an argument after --", []string{"api", "projects", "view", "--", "--id"}, `unexpected argument "--id"`},
		{"not an endpoint", []string{"api", "projects", "frobnicate", "--id", "1"}, `unknown command "frobnicate" for "poeditor api projects"`},
		{"a typo gets a suggestion", []string{"api", "projects", "veiw", "--id", "1"}, "Did you mean this?\n\tview"},
		{"the API docs' spelling", []string{"api", "projects/view", "--id", "1"}, `unknown command "projects/view" for "poeditor api"`},
		{"token as a parameter flag", []string{"api", "contributors", "list", "--api-token", "x"}, "--api-token: the token comes from the account"},
		{"parameter flag without a name", []string{"api", "contributors", "list", "--=x"}, "--=x: missing parameter name"},
		{"unknown shorthand", []string{"api", "projects", "view", "-i", "7717"}, `unknown shorthand flag "-i": API parameters are given as --name value`},
		{"our flag with a bad value", []string{"api", "contributors", "list", "--json=maybe"}, "invalid argument"},
		{"field without =", []string{"api", "projects", "view", "-f", "id"}, `--field "id": use key=value`},
		{"field with empty key", []string{"api", "projects", "view", "-f", "=1"}, `--field "=1": use key=value`},
		{"token as a field", []string{"api", "contributors", "list", "-f", "api_token=x"}, "the token comes from the account"},
		{"token from a file", []string{"api", "contributors", "list", "-F", "api_token=t.txt"}, "the token comes from the account"},
		{"missing field file", []string{"api", "terms", "add", "-F", "data=nope.json"}, "--field-file data: open"},
		{"missing upload", []string{"api", "projects", "upload", "--upload", "nope.json"}, "--upload: open"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)

			res := testutil.Run(t, fake, tt.args...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d API calls, want none", n)
			}
		})
	}
}

// PHP's json_encode escapes every non-ASCII character (\u00e9); nested cells
// and --json must show the characters, not the escapes.
func TestAPI_unicodeEscapesAreDecoded(t *testing.T) {
	result := `{"terms":[{"term":"caf\u00e9","translation":{"content":"\u0421\u043e\u0445\u0440\u0430\u043d\u0438\u0442\u044c \u4fdd\u5b58 \ud83d\ude00 \"q\" <b>"}}]}`
	for _, args := range [][]string{
		{"api", "contributors", "list", "--id", "7717"},
		{"api", "contributors", "list", "--id", "7717", "--json"},
	} {
		t.Run(strings.Join(args[3:], " "), func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("contributors/list", testutil.OK(json.RawMessage(result)))
			env := testutil.NewEnv(t, fake).Login("default", testutil.TestToken)
			env.Width = 200

			res := env.Run(args...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			if strings.Contains(res.Stdout, `\u`) {
				t.Errorf("escapes left in the output:\n%s", res.Stdout)
			}
			for _, want := range []string{"café", "Сохранить 保存 😀", `\"q\" <b>`} {
				if !strings.Contains(res.Stdout, want) {
					t.Errorf("output lacks %q:\n%s", want, res.Stdout)
				}
			}
		})
	}
}
