package cmdutil_test

import (
	"errors"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

func TestNew_noToken(t *testing.T) {
	t.Setenv("POEDITOR_TOKEN", "")

	_, err := cmdutil.New().APIClient()

	if !errors.Is(err, cmdutil.ErrNoToken) {
		t.Fatalf("want ErrNoToken, got %v", err)
	}
}

func TestNew_tokenAndBaseURLFromEnv(t *testing.T) {
	tests := []struct {
		name, envURL, wantURL string
	}{
		{"production by default", "", api.DefaultBaseURL},
		{"override", "https://api.poeditor.loc/v2/", "https://api.poeditor.loc/v2/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("POEDITOR_TOKEN", "abc")
			t.Setenv("POEDITOR_API_URL", tt.envURL)

			client, err := cmdutil.New().APIClient()

			if err != nil {
				t.Fatal(err)
			}
			if client.Token != "abc" || client.BaseURL != tt.wantURL {
				t.Errorf("token=%q url=%q", client.Token, client.BaseURL)
			}
		})
	}
}
