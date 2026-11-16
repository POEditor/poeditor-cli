package cmdutil

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Module is a family of commands living under internal/cmd/<module>/.
//
// The api module is attached at the top level (`poeditor pull`, `poeditor
// api projects list`); every other module gets its own group command
// (`poeditor parser <tool>`).
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

// NewGroup returns a command that only holds subcommands (`poeditor api`,
// `poeditor api terms`). Alone or with --help it prints its help; with a
// word that is not one of its subcommands it fails with "unknown command"
// and cobra's suggestions. (A plain cobra group would print its help and
// exit 0 for a typo, or complain about the first flag it does not know.)
func NewGroup(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:                        use,
		Short:                      short,
		SuggestionsMinimumDistance: 2, // cobra defaults it on the root only
		// flags here belong to the subcommand the user meant: don't parse
		// them, so the unknown word is what gets reported
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			for _, arg := range args {
				if arg == "-h" || arg == "--help" {
					return cmd.Help()
				}
				if name == "" && !strings.HasPrefix(arg, "-") {
					name = arg
				}
			}
			if name == "" {
				return cmd.Help()
			}
			msg := fmt.Sprintf("unknown command %q for %q", name, cmd.CommandPath())
			if suggestions := cmd.SuggestionsFor(name); len(suggestions) > 0 {
				msg += "\n\nDid you mean this?\n\t" + strings.Join(suggestions, "\n\t")
			}
			return errors.New(msg)
		},
	}
}
