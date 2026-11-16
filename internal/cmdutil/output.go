package cmdutil

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

// Every runnable command declares what it prints, in Annotations[OutputAnnotation]:
// OutputData (it lists or shows data: must have --json, see AddJSONFlag) or
// OutputAction (it changes something and only reports on stderr). The root
// tests fail a command that declares neither, or a data command without --json.
const (
	OutputAnnotation = "poeditor/output"
	OutputData       = "data"
	OutputAction     = "action"
)

// AddJSONFlag adds --json to a data command and marks it as one.
func AddJSONFlag(cmd *cobra.Command, target *bool) {
	cmd.Flags().BoolVar(target, "json", false, "Output JSON")
	annotate(cmd, OutputData)
}

// MarkAction marks a command that changes state and prints no data.
func MarkAction(cmd *cobra.Command) { annotate(cmd, OutputAction) }

func annotate(cmd *cobra.Command, kind string) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[OutputAnnotation] = kind
}

// PrintJSON writes v as indented JSON.
func PrintJSON(ios *iostreams.IOStreams, v any) error {
	enc := json.NewEncoder(ios.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
