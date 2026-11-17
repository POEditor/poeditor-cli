package cmdutil_test

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

func TestCheckBulkInput(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		fromFile string
		flags    []string // set on the command line
		want     string   // "" = accepted
	}{
		{"arguments", []string{"save"}, "", []string{"context"}, ""},
		{"a file", nil, "t.json", nil, ""},
		{"too few arguments", nil, "", nil, "give a term, or --from-file (see --help)"},
		{"too many arguments", []string{"a", "b"}, "", nil, "give a term, or --from-file (see --help)"},
		{"arguments and a file", []string{"save"}, "t.json", nil, "--from-file: give the items in the file or as arguments, not both"},
		{"an item flag and a file", nil, "t.json", []string{"context"}, "--context: with --from-file, give it in the file, for each item"},
		{"another flag and a file", nil, "t.json", []string{"other"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("context", "", "")
			cmd.Flags().String("other", "", "")
			for _, name := range tt.flags {
				_ = cmd.Flags().Set(name, "x")
			}

			err := cmdutil.CheckBulkInput(cmd, tt.args, tt.fromFile, 1, "a term", "context")

			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
