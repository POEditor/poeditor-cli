package cmdutil_test

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

func TestAddUnique(t *testing.T) {
	tests := []struct {
		name    string
		add     []*cobra.Command
		wantErr string
	}{
		{"distinct names", []*cobra.Command{{Use: "terms"}, {Use: "parser"}}, ""},
		{"clash with existing sibling", []*cobra.Command{{Use: "projects list"}}, `command name "projects" under "poeditor" is already used by "projects"`},
		{"clash within the batch", []*cobra.Command{{Use: "terms"}, {Use: "terms"}}, `command name "terms" under "poeditor" is already used by "terms"`},
		{"alias clashes with a name", []*cobra.Command{{Use: "proj", Aliases: []string{"projects"}}}, `command name "projects" under "poeditor" is already used by "projects"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent := &cobra.Command{Use: "poeditor"}
			parent.AddCommand(&cobra.Command{Use: "projects"})

			err := cmdutil.AddUnique(parent, tt.add...)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got := len(parent.Commands()); got != 1+len(tt.add) {
					t.Errorf("%d commands attached, want %d", got, 1+len(tt.add))
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
