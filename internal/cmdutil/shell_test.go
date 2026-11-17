package cmdutil_test

import (
	"testing"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

func TestShellJoin(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"fr", "--project", "web"}, "fr --project web"},
		{[]string{"--name", "My app"}, "--name 'My app'"},
		{[]string{"it's", ""}, `'it'\''s' ''`},
		{[]string{"$HOME", "a*"}, "'$HOME' 'a*'"},
	}
	for _, tt := range tests {
		if got := cmdutil.ShellJoin(tt.args); got != tt.want {
			t.Errorf("ShellJoin(%q) = %s, want %s", tt.args, got, tt.want)
		}
	}
}
