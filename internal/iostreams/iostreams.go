// Package iostreams carries the input/output streams a command writes to, so
// tests can swap them for buffers.
package iostreams

import (
	"bytes"
	"io"
	"os"
)

// IOStreams is the set of streams a command reads from and writes to.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

// System returns the process's real stdin, stdout and stderr.
func System() *IOStreams {
	return &IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
}

// Test returns streams backed by buffers: stdin, stdout, stderr.
func Test() (*IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in, out, errOut := &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}
	return &IOStreams{In: in, Out: out, ErrOut: errOut}, in, out, errOut
}
