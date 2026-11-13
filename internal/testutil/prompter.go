package testutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/prompt"
)

// Scripted answers with a special meaning, for any kind of prompt.
const (
	Default = "\x00default" // accept the prompt's default
	Cancel  = "\x00cancel"  // the user pressed Ctrl+C: prompt.ErrCancelled
)

// Prompter answers prompts from a script, in order:
//   - Select: the option's exact label, else a substring matching exactly one option;
//   - Input: the text ("" is an empty answer); a rejected answer is recorded in
//     Rejected and the next answer is used, like a person retyping;
//   - Confirm: "y" or "n".
//
// Default accepts the default and Cancel simulates Ctrl+C, for any kind.
// Running out of answers fails the test.
type Prompter struct {
	T        testing.TB
	Answers  []string
	Asked    []string            // every prompt title, in order
	Offered  map[string][]string // Select title → the options it offered
	Rejected []string            // validation messages shown for rejected Input answers
}

func (p *Prompter) next(title string) (string, error) {
	p.Asked = append(p.Asked, title)
	if len(p.Answers) == 0 {
		p.T.Errorf("no scripted answer for prompt %q", title)
		return "", fmt.Errorf("no scripted answer for prompt %q", title)
	}
	a := p.Answers[0]
	p.Answers = p.Answers[1:]
	if a == Cancel {
		return "", prompt.ErrCancelled
	}
	return a, nil
}

func (p *Prompter) Select(title string, options []string, def int) (int, error) {
	if p.Offered == nil {
		p.Offered = map[string][]string{}
	}
	p.Offered[title] = options
	a, err := p.next(title)
	if err != nil {
		return 0, err
	}
	if a == Default {
		return def, nil
	}
	match := -1
	for i, o := range options {
		if o == a {
			return i, nil
		}
		if strings.Contains(o, a) {
			if match >= 0 {
				p.T.Errorf("answer %q matches several options of %q: %v", a, title, options)
				return 0, fmt.Errorf("ambiguous answer %q", a)
			}
			match = i
		}
	}
	if match < 0 {
		p.T.Errorf("answer %q is not an option of %q: %v", a, title, options)
		return 0, fmt.Errorf("no option %q", a)
	}
	return match, nil
}

func (p *Prompter) Input(title, def string, validate func(string) error) (string, error) {
	for {
		a, err := p.next(title)
		if err != nil {
			return "", err
		}
		if a == Default {
			a = def
		}
		if validate != nil {
			if verr := validate(a); verr != nil {
				p.Rejected = append(p.Rejected, verr.Error())
				continue
			}
		}
		return a, nil
	}
}

func (p *Prompter) Confirm(title string, def bool) (bool, error) {
	a, err := p.next(title)
	if err != nil {
		return false, err
	}
	switch a {
	case Default:
		return def, nil
	case "y":
		return true, nil
	case "n":
		return false, nil
	}
	p.T.Errorf("confirm %q: answer %q, want y or n", title, a)
	return false, fmt.Errorf("bad answer %q", a)
}
