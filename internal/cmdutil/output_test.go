package cmdutil_test

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

func TestTable(t *testing.T) {
	tests := []struct {
		name string
		tty  bool
		want string
	}{
		{"terminal: header and aligned columns", true, "ID  NAME\n1   Web site\n22  App one\n"},
		{"pipe: no header, tab separated", false, "1\tWeb site\n22\tApp one\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, out, _ := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			table := cmdutil.NewTable(ios, "ID", "NAME")
			table.AddRow("1", "Web site")
			table.AddRow("22", "App\tone")

			if err := table.Render(); err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Errorf("got %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestTable_fieldsCannotBreakRows(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	table := cmdutil.NewTable(ios, "A", "B")
	table.AddRow("x", "line1\nline2\r\nline3")

	_ = table.Render()

	if want := "x\tline1 line2 line3\n"; out.String() != want {
		t.Errorf("got %q, want %q", out.String(), want)
	}
}

func TestPrintJSON(t *testing.T) {
	ios, _, out, _ := iostreams.Test()

	if err := cmdutil.PrintJSON(ios, []map[string]int{{"id": 1}}); err != nil {
		t.Fatal(err)
	}
	if want := "[\n  {\n    \"id\": 1\n  }\n]\n"; out.String() != want {
		t.Errorf("got %q", out.String())
	}
}

func TestOutputAnnotations(t *testing.T) {
	var asJSON bool
	data := &cobra.Command{Use: "list"}
	cmdutil.AddJSONFlag(data, &asJSON)
	action := &cobra.Command{Use: "logout"}
	cmdutil.MarkAction(action)

	if data.Annotations[cmdutil.OutputAnnotation] != cmdutil.OutputData || data.Flags().Lookup("json") == nil {
		t.Errorf("data command not set up: %v", data.Annotations)
	}
	if action.Annotations[cmdutil.OutputAnnotation] != cmdutil.OutputAction {
		t.Errorf("action command not marked: %v", action.Annotations)
	}
}
