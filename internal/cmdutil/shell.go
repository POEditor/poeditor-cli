package cmdutil

import "strings"

// ShellJoin writes args back as a command line to copy, quoting what the
// shell would split or expand.
func ShellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		if arg == "" || strings.ContainsAny(arg, " \t\n'\"\\$`|&;<>()*?[]{}~#!") {
			arg = "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
		}
		quoted[i] = arg
	}
	return strings.Join(quoted, " ")
}
