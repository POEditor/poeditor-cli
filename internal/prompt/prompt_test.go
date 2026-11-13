package prompt_test

import (
	"io"
	"strings"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/prompt"
)

// keys feeds keystrokes to the real huh forms, without a terminal.
func keys(s string) prompt.Huh { return prompt.Huh{In: strings.NewReader(s), Out: io.Discard} }

func TestHuh_selectKeepsDefaultOnEnter(t *testing.T) {
	got, err := keys("\r").Select("Pick", []string{"a", "b", "c"}, 1)
	if err != nil || got != 1 {
		t.Fatalf("got %d, %v; want 1", got, err)
	}
}

func TestHuh_selectMovesDown(t *testing.T) {
	got, err := keys("j\r").Select("Pick", []string{"a", "b", "c"}, 0)
	if err != nil || got != 1 {
		t.Fatalf("got %d, %v; want 1", got, err)
	}
}

func TestHuh_inputStartsFromDefault(t *testing.T) {
	got, err := keys("-x\r").Input("Name", "web", nil)
	if err != nil || got != "web-x" {
		t.Fatalf("got %q, %v; want web-x", got, err)
	}
}

func TestHuh_confirm(t *testing.T) {
	got, err := keys("y").Confirm("Sure?", false)
	if err != nil || !got {
		t.Fatalf("got %v, %v; want true", got, err)
	}
}

func TestHuh_ctrlCIsCancelled(t *testing.T) {
	_, err := keys("\x03").Select("Pick", []string{"a"}, 0)
	if err != prompt.ErrCancelled {
		t.Fatalf("got %v, want ErrCancelled", err)
	}
}
