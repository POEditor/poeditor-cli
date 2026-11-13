// Package iostreams carries the input/output streams a command writes to, so
// tests can swap them for buffers, and knows whether stdout is a terminal.
package iostreams

import (
	"bytes"
	"io"
	"os"

	"golang.org/x/term"
)

// ForceTTYEnv set to a non-empty value other than "0" makes stdout count as a
// terminal (e.g. Git Bash/mintty on Windows, which reports a pipe).
const ForceTTYEnv = "POEDITOR_FORCE_TTY"

// IOStreams is the set of streams a command reads from and writes to.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer

	stdoutTTY bool
}

// IsStdoutTTY reports whether stdout is a terminal (a person is reading)
// rather than a pipe or file (a program is reading).
func (s *IOStreams) IsStdoutTTY() bool { return s.stdoutTTY }

// SetStdoutTTY overrides the terminal detection (tests).
func (s *IOStreams) SetStdoutTTY(tty bool) { s.stdoutTTY = tty }

// System returns the process's real stdin, stdout and stderr.
func System(getenv func(string) string) *IOStreams {
	return &IOStreams{
		In:        os.Stdin,
		Out:       os.Stdout,
		ErrOut:    os.Stderr,
		stdoutTTY: forced(getenv(ForceTTYEnv)) || term.IsTerminal(int(os.Stdout.Fd())),
	}
}

func forced(v string) bool { return v != "" && v != "0" }

// Test returns streams backed by buffers: stdin, stdout, stderr. Stdout is
// not a terminal until SetStdoutTTY(true).
func Test() (*IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in, out, errOut := &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}
	return &IOStreams{In: in, Out: out, ErrOut: errOut}, in, out, errOut
}
