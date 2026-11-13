package testutil

import (
	"sort"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/POEditor/poeditor-cli/internal/api"
	"github.com/POEditor/poeditor-cli/internal/cmd/root"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

// Result is what a command run produced.
type Result struct {
	Stdout string
	Stderr string
	Err    error
}

// Run executes `poeditor <args...>` through the real root command, with the
// API pointed at fake. A nil fake means "no token configured".
// Every run is recorded for the option-coverage check (see Main).
func Run(t testing.TB, fake *FakeAPI, args ...string) Result {
	t.Helper()
	ios, _, stdout, stderr := iostreams.Test()

	f := &cmdutil.Factory{
		IO: ios,
		APIClient: func() (*api.Client, error) {
			if fake == nil {
				return nil, cmdutil.ErrNoToken
			}
			return api.NewClient(fake.URL, TestToken, "poeditor-cli/test"), nil
		},
	}

	cmd := root.NewCmdRoot(f)
	cmd.SetArgs(args)
	executed, err := cmd.ExecuteC()
	if executed != nil {
		record(executed)
	}
	return Result{Stdout: stdout.String(), Stderr: stderr.String(), Err: err}
}

var (
	coverageMu sync.Mutex
	ranCmds    = map[string]bool{}            // command path → ran at least once
	usedFlags  = map[string]map[string]bool{} // command path → flag names set
)

func record(cmd *cobra.Command) {
	coverageMu.Lock()
	defer coverageMu.Unlock()
	path := cmd.CommandPath()
	ranCmds[path] = true
	if usedFlags[path] == nil {
		usedFlags[path] = map[string]bool{}
	}
	cmd.Flags().Visit(func(fl *pflag.Flag) { usedFlags[path][fl.Name] = true })
}

func recorded(path string) (ran bool, flags []string) {
	coverageMu.Lock()
	defer coverageMu.Unlock()
	for name := range usedFlags[path] {
		flags = append(flags, name)
	}
	sort.Strings(flags)
	return ranCmds[path], flags
}
