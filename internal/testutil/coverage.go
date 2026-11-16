package testutil

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/POEditor/poeditor-cli/internal/cmd/root"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

// Main is the TestMain of every command package:
//
//	func TestMain(m *testing.M) { testutil.Main(m) }
//
// It runs the package's tests and, if they pass, fails the package unless
// every command implemented in it was run at least once through Run, and
// every one of its flags was set in at least one of those runs.
func Main(m *testing.M) {
	pc, _, _, _ := runtime.Caller(1)
	pkg := strings.TrimSuffix(packageOf(runtime.FuncForPC(pc).Name()), "_test")

	code := m.Run()
	if code == 0 {
		if missing := Uncovered(pkg); len(missing) > 0 {
			fmt.Fprintln(os.Stderr, "FAIL option coverage: every command and flag needs a test case (testutil.Run)")
			for _, line := range missing {
				fmt.Fprintln(os.Stderr, "  -", line)
			}
			code = 1
		}
	}
	os.Exit(code)
}

// Uncovered lists the commands of package pkg that never ran and the flags
// that were never set.
func Uncovered(pkg string) []string {
	var missing []string
	for _, cmd := range Leaves() {
		if CommandPackage(cmd) != pkg {
			continue
		}
		path := cmd.CommandPath()
		ran, used := recorded(path)
		if !ran {
			missing = append(missing, fmt.Sprintf("%q has no test", path))
			continue
		}
		set := map[string]bool{}
		for _, name := range used {
			set[name] = true
		}
		cmd.LocalFlags().VisitAll(func(fl *pflag.Flag) {
			if fl.Name != "help" && !set[fl.Name] {
				missing = append(missing, fmt.Sprintf("%q: flag --%s is never used in a test", path, fl.Name))
			}
		})
	}
	return missing
}

// Leaves returns every runnable command of the tree that has no subcommands
// (groups only route to them), cobra's built-ins excluded.
func Leaves() []*cobra.Command {
	ios, _, _, _ := iostreams.Test()
	var out []*cobra.Command
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if c.Runnable() && !c.HasSubCommands() && !strings.HasPrefix(CommandPackage(c), "github.com/spf13/") {
			out = append(out, c)
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root.NewCmdRoot(&cmdutil.Factory{IO: ios}))
	return out
}

// CommandPackage is the import path of the package that implements cmd's run function.
func CommandPackage(cmd *cobra.Command) string {
	var fn any
	switch {
	case cmd.RunE != nil:
		fn = cmd.RunE
	case cmd.Run != nil:
		fn = cmd.Run
	default:
		return ""
	}
	return packageOf(runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name())
}

// packageOf turns "a/b/pkg.Func.func1" into "a/b/pkg".
func packageOf(funcName string) string {
	slash := strings.LastIndex(funcName, "/")
	dot := strings.Index(funcName[slash+1:], ".")
	if dot < 0 {
		return funcName
	}
	return funcName[:slash+1+dot]
}
