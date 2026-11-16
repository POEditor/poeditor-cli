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

// Translations often hold HTML; --json shows it as written.
func TestPrintJSON_keepsHTML(t *testing.T) {
	ios, _, out, _ := iostreams.Test()

	if err := cmdutil.PrintJSON(ios, map[string]string{"text": "<b>Save</b> & close"}); err != nil {
		t.Fatal(err)
	}
	if want := "{\n  \"text\": \"<b>Save</b> & close\"\n}\n"; out.String() != want {
		t.Errorf("got %q, want %q", out.String(), want)
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

func TestOutputMarks(t *testing.T) {
	var asJSON bool
	cases := map[string]func(*cobra.Command){
		cmdutil.OutputData:   func(c *cobra.Command) { cmdutil.AddJSONFlag(c, &asJSON) },
		cmdutil.OutputAction: cmdutil.MarkAction,
	}
	for want, mark := range cases {
		cmd := &cobra.Command{Use: "x"}
		mark(cmd)
		if got := cmd.Annotations[cmdutil.OutputAnnotation]; got != want {
			t.Errorf("annotation = %q, want %q", got, want)
		}
		if hasJSON := cmd.Flags().Lookup("json") != nil; hasJSON != (want == cmdutil.OutputData) {
			t.Errorf("%s: --json flag present = %v", want, hasJSON)
		}
	}
}

func TestTable_fitsTheTerminal(t *testing.T) {
	long := "The quick brown fox jumps over the lazy dog and keeps running"
	tests := []struct {
		name  string
		width int
		rows  [][]string
		want  string
	}{
		{
			name:  "no width known: nothing cut",
			width: 0,
			rows:  [][]string{{"1", long}},
			want:  "ID  TEXT\n1   " + long + "\n",
		},
		{
			name:  "fits: nothing wrapped, no blank lines",
			width: 80,
			rows:  [][]string{{"1", "short"}, {"2", "also short"}},
			want:  "ID  TEXT\n1   short\n2   also short\n",
		},
		{
			name:  "one long column takes what the short ones leave and wraps",
			width: 30,
			rows:  [][]string{{"1", long}, {"2", "next"}},
			want: "ID  TEXT\n" +
				"1   The quick brown fox jumps\n" +
				"    over the lazy dog and\n" +
				"    keeps running\n" +
				"\n" +
				"2   next\n",
		},
		{
			name:  "a short column keeps its width, the long one wraps",
			width: 30,
			rows:  [][]string{{"1", long, "short text"}},
			want: "ID  TEXT            TEXT2\n" +
				"1   The quick       short text\n" +
				"    brown fox\n" +
				"    jumps over the\n" +
				"    lazy dog and\n" +
				"    keeps running\n",
		},
		{
			name:  "a word wider than the column is split",
			width: 16,
			rows:  [][]string{{"1", "see https://poeditor.com/docs"}},
			want: "ID  TEXT\n" +
				"1   see\n" +
				"    https://poed\n" +
				"    itor.com/doc\n" +
				"    s\n",
		},
		{
			name:  "line breaks are kept",
			width: 80,
			rows:  [][]string{{"1", "first\r\nsecond\nthird"}},
			want:  "ID  TEXT\n1   first\n    second\n    third\n",
		},
		{
			name:  "CJK counts two columns per character",
			width: 20,
			rows:  [][]string{{"1", "翻訳されたテキストはとても長いです"}, {"22", "short"}},
			want: "ID  TEXT\n" +
				"1   翻訳されたテキス\n" +
				"    トはとても長いで\n" +
				"    す\n" +
				"\n" +
				"22  short\n",
		},
		{
			name:  "too narrow: columns keep a minimum width, headers are cut",
			width: 6,
			rows:  [][]string{{"12345", "The fox"}},
			want:  "ID   TE…\n123  The\n45   fox\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, out, _ := iostreams.Test()
			ios.SetStdoutTTY(true)
			ios.SetTerminalWidth(tt.width)
			header := []string{"ID", "TEXT", "TEXT2"}[:len(tt.rows[0])]
			table := cmdutil.NewTable(ios, header...)
			for _, row := range tt.rows {
				table.AddRow(row...)
			}

			if err := table.Render(); err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Errorf("got\n%s\nwant\n%s", out.String(), tt.want)
			}
		})
	}
}

func TestTable_minWidth(t *testing.T) {
	status := "FUZZY: 0  UPDATED: 2026-03-01" // 29 columns, must stay on one line
	tests := []struct {
		name  string
		width int
		want  string
	}{
		{
			name:  "the column keeps its minimum, the others wrap",
			width: 42, // 4 for gaps, 29 reserved, 9 left for TERM and NOTE
			want: "TERM   TRANSLATION                    NOTE\n" +
				"a      FUZZY: 0  UPDATED: 2026-03-01  a\n" +
				"long   Salvează                       long\n" +
				"term                                  note\n",
		},
		{
			name:  "too narrow for the minimum: shared as usual",
			width: 30,
			want: "TERM       TRANSLAT…  NOTE\n" +
				"a long     FUZZY: 0   a long\n" +
				"term       UPDATED:   note\n" +
				"           2026-03-0\n" +
				"           1\n" +
				"           Salvează\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, out, _ := iostreams.Test()
			ios.SetStdoutTTY(true)
			ios.SetTerminalWidth(tt.width)
			table := cmdutil.NewTable(ios, "TERM", "TRANSLATION", "NOTE")
			table.SetMinWidth(1, len(status))
			table.AddRow("a long term", status+"\nSalvează", "a long note")

			if err := table.Render(); err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Errorf("got\n%s\nwant\n%s", out.String(), tt.want)
			}
		})
	}
}

func TestTable_pipedIsNeverCut(t *testing.T) {
	ios, _, out, _ := iostreams.Test()
	ios.SetTerminalWidth(10)
	table := cmdutil.NewTable(ios, "ID", "TEXT")
	table.AddRow("1", "a long text that is not cut")

	if err := table.Render(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "1\ta long text that is not cut\n" {
		t.Errorf("got %q", out.String())
	}
}
