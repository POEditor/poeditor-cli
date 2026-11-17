package terms

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/projectflag"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// Code shared by the commands that write terms (add, update, delete,
// add-comment): each takes one term from its arguments and flags, or many
// from --from-file, and reports the API's counts.

// checkInput checks that the terms come from one place: --from-file, or
// the arguments plus rowFlags (the flags that describe one term). nArgs is
// how many arguments one term takes.
func checkInput(cmd *cobra.Command, args []string, fromFile string, nArgs int, usage string, rowFlags ...string) error {
	if fromFile == "" {
		if len(args) != nArgs {
			return fmt.Errorf("give %s, or --from-file (see --help)", usage)
		}
		return nil
	}
	if len(args) > 0 {
		return errors.New("--from-file: give the terms in the file or as arguments, not both")
	}
	for _, name := range rowFlags {
		if cmd.Flags().Changed(name) {
			return fmt.Errorf("--%s: with --from-file, give it in the file, for each term", name)
		}
	}
	return nil
}

// writeResult is what --json prints: our own shape, like push's.
type writeResult struct {
	Project repoconfig.ProjectRef `json:"project"`
	Terms   any                   `json:"terms"`
}

// report prints the summary line on stderr, and a note when some terms
// were left out; with --json, the counts on stdout.
func report(f *cmdutil.Factory, p projectflag.Project, asJSON bool, summary, note string, counts any) error {
	fmt.Fprintln(f.IO.ErrOut, summary)
	if note != "" {
		fmt.Fprintln(f.IO.ErrOut, note)
	}
	if asJSON {
		return cmdutil.PrintJSON(f.IO, writeResult{Project: p.Ref, Terms: counts})
	}
	return nil
}

// nTerms is "1 term", "2 terms".
func nTerms(n int) string {
	if n == 1 {
		return "1 term"
	}
	return fmt.Sprintf("%d terms", n)
}

// notFound is the note for terms the API did not find.
func notFound(n int) string {
	if n <= 0 {
		return ""
	}
	verb := "were"
	if n == 1 {
		verb = "was"
	}
	return fmt.Sprintf("%s %s not found: a term is matched by its text and its context (--context).", nTerms(n), verb)
}
