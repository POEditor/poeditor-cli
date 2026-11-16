// Package iostreams carries the input/output streams a command writes to, so
// tests can swap them for buffers, and knows whether stdout is a terminal and
// how wide it is.
package iostreams

import (
	"bytes"
	"io"
	"os"
	"strconv"

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
	width     int // terminal columns; 0 = unknown, nothing is fitted
}

// DefaultWidth is used for a terminal whose size can't be read (and $COLUMNS
// is not set), e.g. a forced TTY under mintty.
const DefaultWidth = 80

// IsStdoutTTY reports whether stdout is a terminal (a person is reading)
// rather than a pipe or file (a program is reading).
func (s *IOStreams) IsStdoutTTY() bool { return s.stdoutTTY }

// SetStdoutTTY overrides the terminal detection (tests).
func (s *IOStreams) SetStdoutTTY(tty bool) { s.stdoutTTY = tty }

// TerminalWidth is the terminal's width in columns, or 0 when stdout is not
// a terminal or the width is unknown (tests): output is then not fitted.
func (s *IOStreams) TerminalWidth() int { return s.width }

// SetTerminalWidth overrides the width (tests).
func (s *IOStreams) SetTerminalWidth(columns int) { s.width = columns }

// System returns the process's real stdin, stdout and stderr.
func System(getenv func(string) string) *IOStreams {
	s := &IOStreams{
		In:        os.Stdin,
		Out:       os.Stdout,
		ErrOut:    os.Stderr,
		stdoutTTY: forced(getenv(ForceTTYEnv)) || term.IsTerminal(int(os.Stdout.Fd())),
	}
	if s.stdoutTTY {
		s.width = terminalWidth(getenv)
	}
	return s
}

// terminalWidth asks the terminal, then $COLUMNS, then uses DefaultWidth.
func terminalWidth(getenv func(string) string) int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	if w, err := strconv.Atoi(getenv("COLUMNS")); err == nil && w > 0 {
		return w
	}
	return DefaultWidth
}

func forced(v string) bool { return v != "" && v != "0" }

// Test returns streams backed by buffers: stdin, stdout, stderr. Stdout is
// not a terminal until SetStdoutTTY(true).
func Test() (*IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in, out, errOut := &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}
	return &IOStreams{In: in, Out: out, ErrOut: errOut}, in, out, errOut
}
