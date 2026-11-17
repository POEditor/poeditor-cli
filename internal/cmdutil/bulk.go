package cmdutil

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// CheckBulkInput checks that a bulk command's items come from one place:
// --from-file, or the arguments plus rowFlags (the flags that describe one
// item). nArgs is how many arguments one item takes; usage names them in
// the error ("a term", "a term and the comment").
func CheckBulkInput(cmd *cobra.Command, args []string, fromFile string, nArgs int, usage string, rowFlags ...string) error {
	if fromFile == "" {
		if len(args) != nArgs {
			return fmt.Errorf("give %s, or --from-file (see --help)", usage)
		}
		return nil
	}
	if len(args) > 0 {
		return errors.New("--from-file: give the items in the file or as arguments, not both")
	}
	for _, name := range rowFlags {
		if cmd.Flags().Changed(name) {
			return fmt.Errorf("--%s: with --from-file, give it in the file, for each item", name)
		}
	}
	return nil
}
