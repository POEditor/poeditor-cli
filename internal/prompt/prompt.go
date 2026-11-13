// Package prompt asks the user questions. Commands depend on the Prompter
// interface; the real one draws with charmbracelet/huh, tests use a scripted
// one (testutil.Prompter).
package prompt

import (
	"errors"
	"io"

	"github.com/charmbracelet/huh"
)

// ErrCancelled is returned when the user aborts a prompt (Ctrl+C / Esc).
var ErrCancelled = errors.New("cancelled")

// Prompter asks one question at a time.
type Prompter interface {
	// Select returns the index of the chosen option; def is preselected.
	Select(title string, options []string, def int) (int, error)
	// Input returns the typed text, starting from def. validate (optional)
	// rejects an answer with a message and asks again.
	Input(title, def string, validate func(string) error) (string, error)
	// Confirm returns yes/no, starting from def.
	Confirm(title string, def bool) (bool, error)
}

// Huh draws prompts on out (normally stderr, so stdout stays clean) and reads
// keys from in.
type Huh struct {
	In  io.Reader
	Out io.Writer
}

func (h Huh) run(field huh.Field) error {
	err := huh.NewForm(huh.NewGroup(field)).WithInput(h.In).WithOutput(h.Out).WithShowHelp(true).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrCancelled
	}
	return err
}

func (h Huh) Select(title string, options []string, def int) (int, error) {
	opts := make([]huh.Option[int], len(options))
	for i, label := range options {
		opts[i] = huh.NewOption(label, i)
	}
	choice := def
	field := huh.NewSelect[int]().Title(title).Options(opts...).Value(&choice).Height(min(len(options)+2, 14))
	if len(options) > 10 {
		field = field.Filtering(true).Description("Type / to search")
	}
	return choice, h.run(field)
}

func (h Huh) Input(title, def string, validate func(string) error) (string, error) {
	value := def
	field := huh.NewInput().Title(title).Value(&value)
	if validate != nil {
		field = field.Validate(validate)
	}
	return value, h.run(field)
}

func (h Huh) Confirm(title string, def bool) (bool, error) {
	value := def
	return value, h.run(huh.NewConfirm().Title(title).Value(&value))
}
