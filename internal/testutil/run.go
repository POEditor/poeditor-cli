package testutil

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/POEditor/poeditor-cli/internal/cmd/root"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/config"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
	"github.com/POEditor/poeditor-cli/internal/keyring"
)

// Result is what a command run produced.
type Result struct {
	Stdout string
	Stderr string
	Err    error
}

// Env is an isolated machine for one test: its own config dir, in-memory
// keyring, environment variables and stdin, with the API pointed at Fake.
type Env struct {
	T         testing.TB
	Fake      *FakeAPI // nil: the API is unreachable
	Keyring   *keyring.Memory
	ConfigDir string
	Vars      map[string]string // environment variables seen by the CLI
	Stdin     string
	TTY       bool   // stdout is a terminal (default) or a pipe (false)
	Width     int    // terminal width in columns; 0 (default) = tables are not fitted
	WorkDir   string // the "current directory" (default: a temp dir)
	// Prompter answers prompts; nil means not interactive. Use Answer().
	Prompter *Prompter
	// Sleeps records every wait the command asked for; nobody actually waits.
	Sleeps []time.Duration
}

// Answer makes the run interactive, answering prompts from answers in order.
func (e *Env) Answer(answers ...string) *Env {
	e.Prompter = &Prompter{T: e.T, Answers: answers}
	return e
}

// NewEnv returns an Env with no accounts, stdout a terminal.
func NewEnv(t testing.TB, fake *FakeAPI) *Env {
	t.Helper()
	return &Env{T: t, Fake: fake, Keyring: &keyring.Memory{}, ConfigDir: t.TempDir(), Vars: map[string]string{}, TTY: true, WorkDir: t.TempDir()}
}

// Login sets up account name with token in the keyring, as `auth login` would.
// The first account becomes the default.
func (e *Env) Login(name, token string) *Env {
	e.T.Helper()
	cfg := e.Config()
	if cfg.Accounts == nil {
		cfg.Accounts = map[string]*config.Account{}
	}
	cfg.Accounts[name] = &config.Account{}
	if cfg.DefaultAccount == "" {
		cfg.DefaultAccount = name
	}
	if err := cfg.Save(); err != nil {
		e.T.Fatal(err)
	}
	if err := e.Keyring.Set(name, token); err != nil {
		e.T.Fatal(err)
	}
	return e
}

// WriteRepoConfig writes .poeditor.yml into WorkDir.
func (e *Env) WriteRepoConfig(content string) *Env {
	e.T.Helper()
	if err := os.WriteFile(filepath.Join(e.WorkDir, ".poeditor.yml"), []byte(content), 0o644); err != nil {
		e.T.Fatal(err)
	}
	return e
}

// Config reads the config file as it is on disk now.
func (e *Env) Config() *config.Config {
	e.T.Helper()
	cfg, err := config.Load(e.ConfigDir)
	if err != nil {
		e.T.Fatal(err)
	}
	return cfg
}

// Run executes `poeditor <args...>` through the real root command.
// Every run is recorded for the option-coverage check (see Main).
func (e *Env) Run(args ...string) Result {
	e.T.Helper()
	ios, stdin, stdout, stderr := iostreams.Test()
	stdin.WriteString(e.Stdin)
	ios.SetStdoutTTY(e.TTY)
	ios.SetTerminalWidth(e.Width)

	baseURL := "http://127.0.0.1:1/unreachable/"
	if e.Fake != nil {
		baseURL = e.Fake.URL
	}
	f := &cmdutil.Factory{
		IO:        ios,
		BaseURL:   baseURL,
		ConfigDir: e.ConfigDir,
		WorkDir:   e.WorkDir,
		Getenv:    func(k string) string { return e.Vars[k] },
		Keyring:   e.Keyring,
	}
	if e.Prompter != nil {
		f.Prompter = e.Prompter
	}
	f.Sleep = func(_ context.Context, d time.Duration) error {
		e.Sleeps = append(e.Sleeps, d)
		return nil
	}

	cmd := root.NewCmdRoot(f)
	cmd.SetArgs(args)
	executed, err := cmd.ExecuteC()
	if executed != nil {
		record(executed)
	}
	return Result{Stdout: stdout.String(), Stderr: stderr.String(), Err: err}
}

// Run is the short form for commands that just need to be logged in: one
// account "default" with TestToken against fake. A nil fake means no account.
func Run(t testing.TB, fake *FakeAPI, args ...string) Result {
	t.Helper()
	env := NewEnv(t, fake)
	if fake != nil {
		env.Login("default", TestToken)
	}
	return env.Run(args...)
}

// Lines splits output into trimmed lines, for asserting on stderr messages.
func Lines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
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
