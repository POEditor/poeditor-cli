package cmdutil

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Module is a family of commands living under internal/cmd/<module>/.
//
// The api module is attached at the top level (`poeditor projects list`); every
// other module gets its own group command (`poeditor parser <tool>`).
type Module struct {
	Name     string
	Short    string
	Commands func(f *Factory) []*cobra.Command
}

// AddUnique attaches cmds to parent, refusing any whose name or alias is
// already taken by a sibling. cobra itself accepts duplicates silently and
// then dispatches to whichever comes first.
func AddUnique(parent *cobra.Command, cmds ...*cobra.Command) error {
	taken := map[string]string{}
	for _, existing := range parent.Commands() {
		for _, name := range append([]string{existing.Name()}, existing.Aliases...) {
			taken[name] = existing.Name()
		}
	}
	for _, cmd := range cmds {
		for _, name := range append([]string{cmd.Name()}, cmd.Aliases...) {
			if owner, ok := taken[name]; ok {
				return fmt.Errorf("command name %q under %q is already used by %q", name, parent.Name(), owner)
			}
			taken[name] = cmd.Name()
		}
		parent.AddCommand(cmd)
	}
	return nil
}
