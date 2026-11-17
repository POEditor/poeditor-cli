package terms_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/accounts"
	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestAddComment(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		stdin      string
		data       string
		result     map[string]int
		wantStderr string
	}{
		{"one comment", []string{"save", "Max 10 characters", "-p", "web", "--context", "button"}, "",
			`[{"term":"save","context":"button","comment":"Max 10 characters"}]`,
			map[string]int{"parsed": 1, "with_added_comment": 1}, "Commented on 1 of 1 term in project web.\n"},
		{"pinned", []string{"save", "Keep it short", "--id", "7717", "--pinned"}, "",
			`[{"term":"save","context":"","comment":"Keep it short","pinned":1}]`,
			map[string]int{"parsed": 1, "with_added_comment": 1}, "Commented on 1 of 1 term in project web.\n"},
		{"from stdin, one not found", []string{"-p", "web", "--from-file", "-"},
			`[{"term":"a","comment":"x","pinned":1},{"term":"b","context":"c","comment":"y"}]`,
			`[{"term":"a","context":"","comment":"x","pinned":1},{"term":"b","context":"c","comment":"y"}]`,
			map[string]int{"parsed": 2, "with_added_comment": 1},
			"Commented on 1 of 2 terms in project web.\n1 term was not found: a term is matched by its text and its context (--context).\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("terms/add_comment", testutil.OK(map[string]any{"terms": tt.result}))
			env := writeEnv(t, fake)
			env.Stdin = tt.stdin

			res := env.Run(append([]string{"api", "terms", "add-comment"}, tt.args...)...)

			if res.Err != nil {
				t.Fatal(res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "terms/add_comment" || req.Form.Get("id") != "7717" || req.Form.Get("api_token") != testutil.TestToken {
				t.Errorf("request = %s %v", req.Endpoint, req.Form)
			}
			assertData(t, req.Form.Get("data"), tt.data)
			if res.Stdout != "" || res.Stderr != tt.wantStderr {
				t.Errorf("stdout = %q, stderr = %q, want %q", res.Stdout, res.Stderr, tt.wantStderr)
			}
		})
	}
}

func TestAddComment_json(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("terms/add_comment", testutil.OK(map[string]any{"terms": map[string]int{"parsed": 1, "with_added_comment": 1}}))

	res := writeEnv(t, fake).Run("api", "terms", "add-comment", "save", "x", "-p", "web", "--json")

	if res.Err != nil {
		t.Fatal(res.Err)
	}
	testutil.Golden(t, res.Stdout)
}

func TestAddComment_badInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		file string
		want string
	}{
		{"no comment", []string{"save", "-p", "web"}, "", "give a term and the comment, or --from-file (see --help)"},
		{"an empty comment", []string{"save", " ", "-p", "web"}, "", `the comment: no "comment"`},
		{"three arguments", []string{"a", "b", "c", "-p", "web"}, "", "accepts at most 2 arg(s), received 3"},
		{"--pinned with a file", []string{"-p", "web", "--from-file", "t.json", "--pinned"}, `[{"term":"x","comment":"y"}]`, "--pinned: with --from-file"},
		{"an entry without comment", []string{"-p", "web", "--from-file", "t.json"}, `[{"term":"x"}]`, `entry 1: no "comment"`},
		{"a language (comments have none)", []string{"-p", "web", "--from-file", "t.json"}, `[{"term":"x","comment":"y","language":"fr"}]`, `unknown field "language"`},
		{"--project is required", []string{"save", "x"}, "", `required flag "--project" not set`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			env := writeEnv(t, fake)
			if tt.file != "" {
				writeFile(t, env, "t.json", tt.file)
			}

			res := env.Run(append([]string{"api", "terms", "add-comment"}, tt.args...)...)

			if res.Err == nil || !strings.Contains(res.Err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", res.Err, tt.want)
			}
			if n := len(fake.Requests()); n != 0 {
				t.Errorf("%d API requests, want none", n)
			}
		})
	}
}

func TestAddComment_apiFailure(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("terms/add_comment", testutil.Fail("4012", "Invalid project ID"))

	res := testutil.Run(t, fake, "api", "terms", "add-comment", "save", "x", "-p", "7717")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4012" {
		t.Fatalf("want API error 4012, got %v", res.Err)
	}
}

func TestAddComment_notLoggedIn(t *testing.T) {
	res := testutil.Run(t, nil, "api", "terms", "add-comment", "save", "x", "-p", "7717")

	if !errors.Is(res.Err, accounts.ErrNotLoggedIn) {
		t.Errorf("error = %v, want ErrNotLoggedIn", res.Err)
	}
}
