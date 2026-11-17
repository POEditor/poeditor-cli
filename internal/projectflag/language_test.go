package projectflag_test

import (
	"context"
	"errors"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
	"github.com/POEditor/poeditor-cli/internal/testutil"
)

func TestLanguageError(t *testing.T) {
	other := errors.New("network down")
	tests := []struct {
		name      string
		err       error
		languages any // languages/list result; nil = it fails
		want      string
		wantCalls int
	}{
		{"not in the project: lists the project's languages", &api.Error{Code: api.CodeLanguageNotInProject},
			map[string]any{"languages": []map[string]any{{"code": "fr"}, {"code": "en"}}},
			"xx: project web has no such language (it has: en, fr)", 1},
		{"not in the project, the list fails too", &api.Error{Code: api.CodeLanguageNotInProject}, nil,
			"xx: project web has no such language", 1},
		{"unknown code", &api.Error{Code: api.CodeUnknownLanguage}, nil,
			"xx: not a language code POEditor knows (e.g. en, fr, pt-br; all of them: `poeditor api languages available`)", 0},
		{"another API error is kept", &api.Error{Code: "4012", Message: "Invalid project ID"}, nil,
			"API error 4012: Invalid project ID", 0},
		{"a non-API error is kept", other, nil, "network down", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeAPI(t)
			if tt.languages != nil {
				fake.Stub("languages/list", testutil.OK(tt.languages))
			} else {
				fake.Stub("languages/list", testutil.Fail("4011", "nope"))
			}
			p := projectflag.Project{ID: 7717, Label: "web", Client: api.NewClient(fake.URL, testutil.TestToken, "test")}

			err := p.LanguageError(context.Background(), "xx", tt.err)

			if err == nil || err.Error() != tt.want {
				t.Errorf("error = %v, want %q", err, tt.want)
			}
			if n := len(fake.Requests()); n != tt.wantCalls {
				t.Errorf("%d requests, want %d", n, tt.wantCalls)
			}
		})
	}
}
