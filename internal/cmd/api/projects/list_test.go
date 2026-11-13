package projects_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestMain(m *testing.M) { testutil.Main(m) }

var twoProjects = map[string]any{
	"projects": []map[string]any{
		{"id": 7717, "name": "Mobile app", "public": 0, "open": 0, "created": "2026-01-14T09:12:44+0000"},
		{"id": 4321, "name": "Website", "public": 1, "open": 1, "created": "2025-06-02T17:03:10+0000"},
	},
}

var noProjects = map[string]any{"projects": []any{}}

func TestList_output(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		result any
	}{
		{"table", []string{"projects", "list"}, twoProjects},
		{"table_empty", []string{"projects", "list"}, noProjects},
		{"json", []string{"projects", "list", "--json"}, twoProjects},
		{"json_empty", []string{"projects", "list", "--json"}, noProjects},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			fake.Stub("projects/list", testutil.OK(tt.result))

			res := testutil.Run(t, fake, tt.args...)

			if res.Err != nil {
				t.Fatalf("unexpected error: %v", res.Err)
			}
			req := fake.OnlyRequest(t)
			if req.Endpoint != "projects/list" {
				t.Errorf("endpoint = %q, want projects/list", req.Endpoint)
			}
			if got := req.Form.Get("api_token"); got != testutil.TestToken {
				t.Errorf("api_token = %q, want %q", got, testutil.TestToken)
			}
			if res.Stderr != "" {
				t.Errorf("unexpected stderr: %q", res.Stderr)
			}
			testutil.Golden(t, res.Stdout)
		})
	}
}

func TestList_apiError(t *testing.T) {
	fake := testutil.NewFakeAPI(t)
	fake.Stub("projects/list", testutil.Fail("4011", "Invalid API Token"))

	res := testutil.Run(t, fake, "projects", "list")

	var apiErr *api.Error
	if !errors.As(res.Err, &apiErr) || apiErr.Code != "4011" {
		t.Fatalf("want API error 4011, got %v", res.Err)
	}
	if res.Err.Error() != "API error 4011: Invalid API Token" {
		t.Errorf("message = %q", res.Err.Error())
	}
	if res.Stdout != "" {
		t.Errorf("nothing should be printed on error, got %q", res.Stdout)
	}
}

func TestList_noToken(t *testing.T) {
	res := testutil.Run(t, nil, "projects", "list")

	if !errors.Is(res.Err, cmdutil.ErrNoToken) {
		t.Fatalf("want ErrNoToken, got %v", res.Err)
	}
}

func TestList_rejectsArguments(t *testing.T) {
	fake := testutil.NewFakeAPI(t)

	res := testutil.Run(t, fake, "projects", "list", "extra")

	if res.Err == nil || !strings.Contains(res.Err.Error(), "unknown command") {
		t.Fatalf("want an argument error, got %v", res.Err)
	}
	if n := len(fake.Requests()); n != 0 {
		t.Errorf("no API call expected, got %d", n)
	}
}
